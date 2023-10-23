package utils

import (
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/gin-gonic/gin"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func UploadEncode(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("出现错误: %s", err.Error()))
		common.LogPrintln("", common.ErStr, "出现错误:", err.Error())
		return
	}
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		common.LogPrintln("", common.ErStr, "父目录参数不正确")
		return
	}
	// 获取所有文件
	files := form.File["files"]
	// 遍历所有文件
	for _, file := range files {
		// 上传文件至指定目录
		dst := filepath.Join(common.LumikaWorkDirPath, parentDir, file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("上传失败: %s", err.Error()))
			common.LogPrintln("", common.ErStr, "上传失败:", err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("文件上传成功: 已上传 %d 个文件", len(files))})
	common.LogPrintln("", "文件上传成功: 已上传", len(files), "个文件")
}

func UploadDecode(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("出现错误: %s", err.Error())})
		common.LogPrintln("", common.ErStr, "出现错误:", err.Error())
		return
	}
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		common.LogPrintln("", common.ErStr, "父目录参数不正确")
		return
	}

	folderName := c.PostForm("folderName")
	if folderName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "上传的文件夹名称不能为空"})
		common.LogPrintln("", common.ErStr, "上传的文件夹名称不能为空")
		return
	}
	folderName = ReplaceInvalidCharacters(folderName, '-')

	// 创建目标文件夹（如果不存在）
	targetDir := filepath.Join(common.LumikaWorkDirPath, parentDir, folderName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err := os.Mkdir(targetDir, 0755)
		if err != nil {
			return
		}
	}

	// 获取所有文件
	files := form.File["directory"]
	// 遍历所有文件
	for _, file := range files {
		// 上传文件至指定目录
		dst := filepath.Join(common.LumikaWorkDirPath, parentDir, folderName, file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("上传失败: %s", err.Error()))
			common.LogPrintln("", common.ErStr, "上传失败:", err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, fmt.Sprintf("目录上传成功: 已上传 %d 个文件", len(files)))
	common.LogPrintln("", "目录上传成功: 已上传", len(files), "个文件")
}

func GetFileFromURL(c *gin.Context) {
	urla := c.PostForm("url")
	if urla == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL 不能为空"})
		common.LogPrintln("", common.ErStr, "URL 不能为空")
		return
	}
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		common.LogPrintln("", common.ErStr, "父目录参数不正确")
		return
	}
	fileName := c.PostForm("fileName")
	if fileName == "" {
		fileName = GetFileNameFromURL(urla)
	}
	DownloadThreadNumInputData := c.PostForm("DownloadThreadNumInputData")
	common.LogPrintln("", "读取到 DownloadThreadNumInputData:", DownloadThreadNumInputData)
	gor := common.VarSettingsVariable.DefaultBiliDownloadGoRoutines
	gors, err := strconv.Atoi(DownloadThreadNumInputData)
	if err != nil {
		if DownloadThreadNumInputData == "" {
			gors = common.VarSettingsVariable.DefaultBiliDownloadGoRoutines
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "线程参数错误"})
			common.LogPrintln("", common.ErStr, "线程参数错误:", err.Error())
			return
		}
	}
	if gors > 0 {
		gor = gors
	}
	fileName = ReplaceInvalidCharacters(fileName, '-')
	filePath := filepath.Join(common.LumikaWorkDirPath, parentDir, fileName)
	go DlAddTask(urla, filePath, "", "", "", gor)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s, 使用线程数: %d", fileName, gor)})
	common.LogPrintln("", "成功添加下载任务:", fileName, "使用线程数:", gor)
}

func GetFileFromBiliID(c *gin.Context) {
	AVOrBVStr := c.PostForm("biliId")
	if AVOrBVStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BV/av号不能为空"})
		common.LogPrintln("", common.ErStr, "BV/av号不能为空")
		return
	}
	baseStr := c.PostForm("baseStr")
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		common.LogPrintln("", common.ErStr, "父目录参数不正确")
		return
	}
	go BDlAddTask(AVOrBVStr, baseStr, parentDir)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s", AVOrBVStr)})
	common.LogPrintln("", "成功添加下载任务:", AVOrBVStr)
}

func GetDlTaskList(c *gin.Context) {
	var DlTaskListArray []*common.DlTaskListData
	for _, kq := range common.DlTaskList {
		DlTaskListArray = append(DlTaskListArray, kq)
	}
	var BDlTaskListArray []*common.BDlTaskListData
	for _, kq := range common.BDlTaskList {
		BDlTaskListArray = append(BDlTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"dlTaskList": DlTaskListArray, "bdlTaskList": BDlTaskListArray})
}

func GetFileList(c *gin.Context) {
	EncodeDirData, err := GetDirectoryJSON(common.LumikaEncodePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		common.LogPrintln("", common.ErStr, "读取文件出现错误:", err.Error())
		return
	}
	EncodeOutputDirData, err := GetDirectoryJSON(common.LumikaEncodeOutputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		common.LogPrintln("", common.ErStr, "读取文件出现错误:", err.Error())
		return
	}
	DecodeDirData, err := GetDirectoryJSON(common.LumikaDecodePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		common.LogPrintln("", common.ErStr, "读取文件出现错误:", err.Error())
		return
	}
	DecodeOutputDirData, err := GetDirectoryJSON(common.LumikaDecodeOutputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		common.LogPrintln("", common.ErStr, "读取文件出现错误:", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"encode": EncodeDirData, "encodeOutput": EncodeOutputDirData, "decode": DecodeDirData, "decodeOutput": DecodeOutputDirData})
}

func AddEncodeTask(c *gin.Context) {
	var ed *common.AddTaskInfo
	if err := c.ShouldBindJSON(&ed); err != nil {
		c.JSON(400, gin.H{"msg": "AddEncodeTask JSON 解析错误", "error": err.Error()})
		common.LogPrintln("", common.AddStr, common.ErStr, "AddEncodeTask JSON 解析错误:", err.Error())
		return
	}
	if ed.DefaultM == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: M 的值不能为 0，自动设置 M = "+strconv.Itoa(common.AddMLevel))
		ed.DefaultM = common.AddMLevel
	}
	if ed.DefaultK == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: K 的值不能为 0，自动设置 K = "+strconv.Itoa(common.AddKLevel))
		ed.DefaultK = common.AddKLevel
	}
	if ed.DefaultK > ed.DefaultM {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: K 的值不能大于 M 的值，自动设置 K = M = "+strconv.Itoa(ed.DefaultM))
		ed.DefaultK = ed.DefaultM
	}
	if ed.MGValue == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: MG 的值不能为 0，自动设置 MG = "+strconv.Itoa(common.AddMGLevel))
		ed.MGValue = common.AddMGLevel
	}
	if ed.KGValue == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: KG 的值不能为 0，自动设置 KG = "+strconv.Itoa(common.AddKGLevel))
		ed.KGValue = common.AddKGLevel
	}
	if ed.KGValue > ed.MGValue {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: KG 的值不能大于 MG 的值，自动设置 KG = MG = "+strconv.Itoa(ed.MGValue))
		ed.KGValue = ed.MGValue
	}
	if ed.VideoSize <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 分辨率大小不能小于等于 0，自动设置分辨率大小为", common.EncodeVideoSizeLevel)
		ed.VideoSize = common.EncodeVideoSizeLevel
	}
	if ed.OutputFPS <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 输出帧率不能小于等于 0，自动设置输出帧率为", common.EncodeOutputFPSLevel)
		ed.OutputFPS = common.EncodeOutputFPSLevel
	}
	if ed.EncodeMaxSeconds <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 最大编码时间不能小于等于 0，自动设置最大编码时间为", common.EncodeMaxSecondsLevel)
		ed.EncodeMaxSeconds = common.EncodeMaxSecondsLevel
	}
	if ed.EncodeThread <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", common.VarSettingsVariable.DefaultMaxThreads)
		ed.EncodeThread = common.VarSettingsVariable.DefaultMaxThreads
	}
	if ed.EncodeVersion != 3 && ed.EncodeVersion != 4 && ed.EncodeVersion != 5 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 编码方法版本号只能设置为3, 4或5，自动设置处理使用的线程数量为", common.EncodeVersion)
		ed.EncodeVersion = common.EncodeVersion
	}
	if ed.EncodeVer5ColorGA < 0 || ed.EncodeVer5ColorGA > 255 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 编码方法版本5的颜色GA通道值只能设置为0-255，自动设置为", common.EncodeVer5ColorGA)
		ed.EncodeVer5ColorGA = common.EncodeVer5ColorGA
	}
	if ed.EncodeVer5ColorBA < 0 || ed.EncodeVer5ColorBA > 255 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 编码方法版本5的颜色BA通道值只能设置为0-255，自动设置为", common.EncodeVer5ColorBA)
		ed.EncodeVer5ColorBA = common.EncodeVer5ColorBA
	}
	if ed.EncodeVer5ColorGB < 0 || ed.EncodeVer5ColorGB > 255 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 编码方法版本5的颜色GA通道值只能设置为0-255，自动设置为", common.EncodeVer5ColorGB)
		ed.EncodeVer5ColorGB = common.EncodeVer5ColorGB
	}
	if ed.EncodeVer5ColorBB < 0 || ed.EncodeVer5ColorBB > 255 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 编码方法版本5的颜色B通道值只能设置为0-255，自动设置为", common.EncodeVer5ColorBB)
		ed.EncodeVer5ColorBB = common.EncodeVer5ColorBB
	}
	go AddAddTask(ed.FileNameList, ed.DefaultM, ed.DefaultK, ed.MGValue, ed.KGValue, ed.VideoSize, ed.OutputFPS, ed.EncodeMaxSeconds, ed.EncodeThread, ed.EncodeVersion, ed.EncodeVer5ColorGA, ed.EncodeVer5ColorBA, ed.EncodeVer5ColorGB, ed.EncodeVer5ColorBB, ed.EncodeFFmpegMode, ed.DefaultSummary)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加编码任务")})
	common.LogPrintln("", common.AddStr, "成功添加编码任务")
}

func GetAddTaskList(c *gin.Context) {
	var AddTaskListArray []*common.AddTaskListData
	for _, kq := range common.AddTaskList {
		AddTaskListArray = append(AddTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"encodeTaskList": AddTaskListArray})
}

func AddDecodeTask(c *gin.Context) {
	var ed *common.GetTaskInfo
	if err := c.ShouldBindJSON(&ed); err != nil {
		c.JSON(400, gin.H{"msg": "AddDecodeTask JSON 解析错误", "error": err.Error()})
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: AddDecodeTask JSON 解析错误:", err.Error())
		return
	}
	if ed.DirName == "" {
		c.JSON(400, gin.H{"msg": "AddDecodeTask: DirName 参数错误，任务创建失败"})
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: DirName 参数错误，任务创建失败")
		return
	}
	if ed.DecodeThread <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为本机线程数:", common.VarSettingsVariable.DefaultMaxThreads)
		ed.DecodeThread = common.VarSettingsVariable.DefaultMaxThreads
	}
	go AddGetTask(ed.DirName, ed.DecodeThread, ed.BaseStr)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加解码任务")})
	common.LogPrintln("", common.AddStr, "成功添加解码任务")
}

func GetGetTaskList(c *gin.Context) {
	var GetTaskListArray []*common.GetTaskListData
	for _, kq := range common.GetTaskList {
		GetTaskListArray = append(GetTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"decodeTaskList": GetTaskListArray})
}

func GetLogCat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"logcat": common.LogVariable.String()})
}

func DeleteFileFromAPI(c *gin.Context) {
	dir := c.PostForm("dir")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "dir 参数错误：请指定正确的目录")
		return
	}
	fileName := c.PostForm("file")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file 参数错误：请输入正确的文件/目录名"})
		return
	}
	filePath := filepath.Join(common.LumikaWorkDirPath, dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件/目录不存在"})
		common.LogPrintln("", common.ErStr, "文件/目录不存在:", filePath)
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件/目录不存在"})
		common.LogPrintln("", common.ErStr, "文件/目录不存在:", filePath)
		return
	}
	if fileInfo.IsDir() {
		osFile, err := os.Open(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "打开目录失败"})
			common.LogPrintln("", common.ErStr, "打开目录失败:", err.Error())
			return
		}
		defer osFile.Close()
		_, err = osFile.Readdir(1)
		if err == nil {
			err = os.RemoveAll(filePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "删除目录失败"})
				common.LogPrintln("", common.ErStr, "删除目录失败:", err.Error())
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "目录删除成功"})
			common.LogPrintln("", "目录删除成功:", filePath)
			return
		} else {
			err = os.Remove(filePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "删除目录失败"})
				common.LogPrintln("", common.ErStr, "删除目录失败:", err.Error())
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "目录删除成功"})
			common.LogPrintln("", "目录删除成功:", filePath)
			return
		}
	} else {
		err = os.Remove(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
			common.LogPrintln("", common.ErStr, "删除文件失败:", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "文件删除成功"})
		common.LogPrintln("", "文件删除成功:", filePath)
		return
	}
}

func ReNameFileFromAPI(c *gin.Context) {
	dir := c.PostForm("dir")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "dir 参数错误：请指定正确的目录")
		return
	}
	originName := c.PostForm("originName")
	if originName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 参数错误：请输入正确的文件/目录名"})
		common.LogPrintln("", common.ErStr, "originName 参数错误：请输入正确的文件/目录名")
		return
	}
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 参数错误：请输入正确的文件/目录名"})
		common.LogPrintln("", common.ErStr, "originName 参数错误：请输入正确的文件/目录名")
		return
	}
	originFilePath := filepath.Join(common.LumikaWorkDirPath, dir, originName)
	filePath := filepath.Join(common.LumikaWorkDirPath, dir, name)
	if _, err := os.Stat(originFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 的文件/目录不存在"})
		common.LogPrintln("", common.ErStr, "originName 的文件/目录不存在")
		return
	}
	if _, err := os.Stat(filePath); os.IsExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 的文件/目录已经存在，请换个名称"})
		common.LogPrintln("", common.ErStr, "name 的文件/目录已经存在，请换个名称")
		return
	}
	err := os.Rename(originFilePath, filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重命名失败"})
		common.LogPrintln("", common.ErStr, "重命名失败:", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "重命名成功"})
	common.LogPrintln("", "重命名成功:", originFilePath, "=>", filePath)
}

func zipFileFromAPI(c *gin.Context) {
	if runtime.GOOS != "linux" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持压缩"})
		common.LogPrintln("", common.ErStr, "当前系统不支持压缩")
		return
	}
	dir := c.PostForm("dir")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "dir 参数错误：请指定正确的目录")
		return
	}
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 参数错误：请输入正确的文件/目录名"})
		common.LogPrintln("", common.ErStr, "originName 参数错误：请输入正确的文件/目录名")
		return
	}
	zipsSize := c.PostForm("zipsSize")
	if zipsSize == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zipsSize 参数错误：请输入正确的文件/目录名"})
		common.LogPrintln("", common.ErStr, "zipsSize 参数错误：请输入正确的文件/目录名")
		return
	}
	zipPwd := c.PostForm("zipPwd")
	filePath := filepath.Join(common.LumikaWorkDirPath, dir, name)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		common.LogPrintln("", common.ErStr, "文件不存在:", filePath)
		return
	}

	zipFilePath := filePath + ".zip"
	if _, err := os.Stat(zipFilePath); os.IsExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "压缩文件已存在，请先删除后重试"})
		return
	}
	if zipPwd == "" || zipPwd == "undefined" {
		zipCommand := exec.Command("/bin/sh", "-c", strings.Join([]string{"$(which zip)", "-r", "-s", fmt.Sprintf("%v", zipsSize), zipFilePath, "\"" + filepath.Base(filePath) + "\""}, " "))
		zipCommand.Dir = filepath.Dir(filePath)
		err := zipCommand.Run()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "压缩失败(无密码)"})
			common.LogPrintln("", common.ErStr, "压缩失败(无密码):", err.Error())
			return
		}
	} else {
		zipCommand := exec.Command("/bin/sh", "-c", strings.Join([]string{"$(which zip)", "-r", "-s", fmt.Sprintf("%v", zipsSize), "-P", zipPwd, zipFilePath, "\"" + filepath.Base(filePath) + "\""}, " "))
		zipCommand.Dir = filepath.Dir(filePath)
		err := zipCommand.Run()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "压缩失败(有密码)"})
			common.LogPrintln("", common.ErStr, "压缩失败(有密码):", err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "压缩成功"})
	common.LogPrintln("", "压缩成功:", zipFilePath)
}

func CopyToOtherFolderFromAPI(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 参数错误：请输入正确的文件/目录名"})
		common.LogPrintln("", common.ErStr, "name 参数错误：请输入正确的文件/目录名")
		return
	}
	sourceDir := c.PostForm("sourceDir")
	if sourceDir != "encode" && sourceDir != "encodeOutput" && sourceDir != "decode" && sourceDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sourceDir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "sourceDir 参数错误：请指定正确的目录")
		return
	}
	targetDir := c.PostForm("targetDir")
	if targetDir != "encode" && targetDir != "encodeOutput" && targetDir != "decode" && targetDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "targetDir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "targetDir 参数错误：请指定正确的目录")
		return
	}
	filePath := filepath.Join(common.LumikaWorkDirPath, sourceDir, name)
	targetFilePath := filepath.Join(common.LumikaWorkDirPath, targetDir, name)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		common.LogPrintln("", common.ErStr, "文件不存在:", filePath)
		return
	}
	if _, err := os.Stat(targetFilePath); os.IsExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件已存在，请先删除后重试"})
		common.LogPrintln("", common.ErStr, "文件已存在，请先删除后重试:", targetFilePath)
		return
	}
	sourceInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件信息读取失败"})
		common.LogPrintln("", common.ErStr, "文件信息读取失败:", err.Error())
		return
	}
	if sourceInfo.IsDir() {
		err = CopyDir(filePath, targetFilePath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "复制目录出现错误"})
			common.LogPrintln("", common.ErStr, "复制目录出现错误:", err.Error())
			return
		}
	} else {
		err = CopyFile(filePath, targetFilePath)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "复制文件出现错误"})
			common.LogPrintln("", common.ErStr, "复制文件出现错误:", err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "压缩成功"})
	common.LogPrintln("", "复制成功:", filePath, "=>", targetFilePath)
}

func DownloadFileFromAPI(c *gin.Context) {
	dir := c.Param("dir")
	fileName := c.Param("file")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "dir 参数错误：请指定正确的目录")
		return
	}
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file 参数错误：请输入正确的文件名"})
		common.LogPrintln("", common.ErStr, "file 参数错误：请输入正确的文件名")
		return
	}
	filePath := filepath.Join(common.LumikaWorkDirPath, dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		common.LogPrintln("", common.ErStr, "文件不存在:", filePath)
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		common.LogPrintln("", common.ErStr, "文件不存在:", filePath)
		return
	}
	if !fileInfo.IsDir() {
		ext := filepath.Ext(fileName)
		contentType := mime.TypeByExtension(ext)
		if contentType != "" {
			c.Writer.Header().Set("Content-Type", contentType)
		}
		c.File(filePath)
		return
	} else {
		zipFilePath := filepath.Join(filepath.Dir(filePath), fileName+".zip")
		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			err = Zip(zipFilePath, filePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "目录压缩失败"})
				common.LogPrintln("", common.ErStr, "目录压缩失败:", err.Error())
				return
			}
		}
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(zipFilePath)))
		c.File(zipFilePath)
		return
	}
}

func UpdateFromAPI(c *gin.Context) {
	latestVersion, latestVersionSummary := GetUpdateDaemon(false)
	c.JSON(http.StatusOK, gin.H{"latestVersion": latestVersion, "latestVersionSummary": latestVersionSummary})
	common.LogPrintln("", "已发送更新请求")
}

func UpdateRequestFromAPI(c *gin.Context) {
	go GetUpdateDaemon(true)
	c.JSON(http.StatusOK, gin.H{"message": "已发送更新请求，请稍后查看更新日志"})
	common.LogPrintln("", "已发送更新请求")
}

func UnzipFromAPI(c *gin.Context) {
	dir := c.PostForm("dir")
	fileName := c.PostForm("file")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		common.LogPrintln("", common.ErStr, "dir 参数错误：请指定正确的目录")
		return
	}
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file 参数错误：请输入正确的文件名"})
		common.LogPrintln("", common.ErStr, "file 参数错误：请输入正确的文件名")
		return
	}
	filePath := filepath.Join(common.LumikaWorkDirPath, dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		common.LogPrintln("", common.ErStr, "文件不存在:", filePath)
		return
	}
	err := Unzip(filePath, filepath.Dir(filePath))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解压失败"})
		common.LogPrintln("", common.ErStr, "解压失败:", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "解压成功"})
	common.LogPrintln("", "解压成功:", filePath)
}

func ClearLogCat(c *gin.Context) {
	common.LogVariable.Reset()
	c.JSON(http.StatusOK, gin.H{"message": "日志清除成功"})
	common.LogPrintln("", "日志清除成功")
}

func ClearDlTaskList(c *gin.Context) {
	//for _, kq := range DlTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除下载任务列表"})
	//		return
	//	}
	//}
	//for _, kq := range BDlTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除下载任务列表"})
	//		return
	//	}
	//}
	common.DlTaskList = make(map[string]*common.DlTaskListData)
	common.BDlTaskList = make(map[string]*common.BDlTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "下载任务列表清除成功"})
	common.LogPrintln("", "下载任务列表清除成功")
}
func ClearAddTaskList(c *gin.Context) {
	//for _, kq := range AddTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除编码任务列表"})
	//		return
	//	}
	//}
	common.AddTaskList = make(map[string]*common.AddTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "编码任务列表清除成功"})
	common.LogPrintln("", "编码任务列表清除成功")
}

func ClearGetTaskList(c *gin.Context) {
	//for _, kq := range GetTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除解码任务列表"})
	//		return
	//	}
	//}
	common.GetTaskList = make(map[string]*common.GetTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "解码任务列表清除成功"})
	common.LogPrintln("", "解码任务列表清除成功")
}

func PauseAddTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.AddTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	if common.AddTaskList[uuidd].IsPaused {
		common.AddTaskList[uuidd].IsPaused = false
		common.AddTaskList[uuidd].Status = "正在执行"
		common.AddTaskList[uuidd].StatusMsg = "正在执行"
		c.JSON(http.StatusOK, gin.H{"message": "成功启动任务"})
		common.LogPrintln("", common.AddStr, "成功启动任务")
		return
	} else {
		common.AddTaskList[uuidd].IsPaused = true
		common.AddTaskList[uuidd].Status = "已暂停"
		common.AddTaskList[uuidd].StatusMsg = "任务已暂停"
		c.JSON(http.StatusOK, gin.H{"message": "成功暂停任务"})
		common.LogPrintln("", common.AddStr, "成功暂停任务")
		return
	}
}

func PauseGetTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.GetTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	if common.GetTaskList[uuidd].IsPaused {
		common.GetTaskList[uuidd].IsPaused = false
		common.GetTaskList[uuidd].Status = "正在执行"
		common.GetTaskList[uuidd].StatusMsg = "正在执行"
		c.JSON(http.StatusOK, gin.H{"message": "成功启动任务"})
		common.LogPrintln("", common.WebStr, "成功启动任务")
		return
	} else {
		common.GetTaskList[uuidd].IsPaused = true
		common.GetTaskList[uuidd].Status = "已暂停"
		common.GetTaskList[uuidd].StatusMsg = "任务已暂停"
		c.JSON(http.StatusOK, gin.H{"message": "成功暂停任务"})
		common.LogPrintln("", common.WebStr, "成功暂停任务")
		return
	}
}

func DeleteDlTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.DlTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	delete(common.DlTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	common.LogPrintln("", common.DlStr, "成功删除任务")
	return
}

func DeleteBDlTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.BDlTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	delete(common.BDlTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	common.LogPrintln("", common.DlStr, "成功删除任务")
	return
}

func DeleteAddTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.AddTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	delete(common.AddTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	common.LogPrintln("", common.AddStr, "成功删除任务")
	return
}

func DeleteGetTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.GetTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	delete(common.GetTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	common.LogPrintln("", common.WebStr, "成功删除任务")
	return
}

func GetServerStatus(c *gin.Context) {
	usage, err := GetSystemResourceUsage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取服务器资源使用情况失败"})
		common.LogPrintln("", common.WebStr, "获取服务器资源使用情况失败:", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": usage})
	return
}

func RestartServer(c *gin.Context) {
	err := RestartProgram()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重启服务器后端进程失败"})
		common.LogPrintln("", common.WebStr, "重启服务器后端进程失败:", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": "服务器后端进程重启成功"})
	common.LogPrintln("", common.WebStr, "服务器后端进程重启成功")
	return
}

func GetBUlTaskList(c *gin.Context) {
	var BUlTaskListArray []*common.BUlTaskListData
	for _, kq := range common.BUlTaskList {
		BUlTaskListArray = append(BUlTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"bulTaskList": BUlTaskListArray})
	common.LogPrintln("", "已发送解码任务列表")
}

func DeleteBUlTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		common.LogPrintln("", common.ErStr, "错误: 任务的 UUID 不能为空")
		return
	}
	_, exist := common.BUlTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		common.LogPrintln("", common.ErStr, "错误: 没有找到指定的任务")
		return
	}
	delete(common.BUlTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	common.LogPrintln("", common.BUlStr, "成功删除任务")
	return
}

func AddBUlTask(c *gin.Context) {
	var ed *common.BUlTaskInfo
	if err := c.ShouldBindJSON(&ed); err != nil {
		c.JSON(400, gin.H{"msg": "AddBUlTask JSON 解析错误", "error": err.Error()})
		common.LogPrintln("", common.ErStr, "AddBUlTask JSON 解析错误:", err.Error())
		return
	}
	if ed.FileName == "" {
		c.JSON(400, gin.H{"msg": "AddBUlTask: FileName 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "AddBUlTask: FileName 参数错误，任务创建失败")
		return
	}
	if ed.UploadLines == "" {
		ed.UploadLines = common.DefaultBiliUploadLines
	}
	if ed.Threads <= 0 || ed.Threads > 256 {
		ed.Threads = common.DefaultBiliUploadThreads
	}
	if ed.VideoInfos.Title == "" {
		c.JSON(400, gin.H{"msg": "AddBUlTask: VideoInfos.Title 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "AddBUlTask: VideoInfos.Title 参数错误，任务创建失败")
		return
	}
	if ed.VideoInfos.Copyright != 1 && ed.VideoInfos.Copyright != 2 {
		ed.VideoInfos.Copyright = 1
	}
	go BUlAddTask(ed)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加解码任务")})
	common.LogPrintln("", common.BUlStr, "成功添加解码任务")
}

func ClearBUlTaskList(c *gin.Context) {
	//for _, kq := range BUlTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除解码任务列表"})
	//		return
	//	}
	//}
	common.BUlTaskList = make(map[string]*common.BUlTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "任务列表清除成功"})
	common.LogPrintln("", common.BUlStr, "任务列表清除成功")
}

func GetBilibiliQRCode(c *gin.Context) {
	values := url.Values{}
	values.Set("local_id", "0")
	values.Set("ts", strconv.FormatInt(time.Now().Unix(), 10))

	// 添加 sign (使用云视听版本的 appKey 和 appsec)
	appkey := "4409e2ce8ffd12b8"
	appsec := "59b43e04ad6965f34319062b478f83dd"
	signedValues := GetBilibiliAppSign(values, appkey, appsec)

	resp, err := http.PostForm("https://passport.bilibili.com/x/passport-tv-login/qrcode/auth_code", signedValues)
	if err != nil {
		fmt.Println("获取哔哩哔哩二维码数据 POST 请求出错:", err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("获取哔哩哔哩二维码数据 POST 请求出错:", err)
		return
	}
	c.String(http.StatusOK, string(body))
}

func GetBilibiliPoll(c *gin.Context) {
	authCode := c.Param("ac")
	if authCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth_code 参数错误"})
		common.LogPrintln("", common.ErStr, "auth_code 参数错误")
		return
	}
	values := url.Values{}
	values.Set("auth_code", authCode)
	values.Set("local_id", "0")
	values.Set("ts", strconv.FormatInt(time.Now().Unix(), 10))

	// 添加 sign (使用云视听版本的 appKey 和 appsec)
	appkey := "4409e2ce8ffd12b8"
	appsec := "59b43e04ad6965f34319062b478f83dd"
	signedValues := GetBilibiliAppSign(values, appkey, appsec)

	resp, err := http.PostForm("https://passport.bilibili.com/x/passport-tv-login/qrcode/poll", signedValues)
	if err != nil {
		fmt.Println("获取 Poll 登录数据 POST 请求出错:", err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("获取 Poll 登录数据 POST 请求出错:", err)
		return
	}
	c.String(http.StatusOK, string(body))
}

func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": common.LumikaVersionString})
	common.LogPrintln("", "已发送版本信息")
	return
}

func SetVarSettingsConfig(c *gin.Context) {
	var vs *common.VarSettings
	if err := c.ShouldBindJSON(&vs); err != nil {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig JSON 解析错误", "error": err.Error()})
		common.LogPrintln("", common.ErStr, "SetVarSettingsConfig JSON 解析错误:", err.Error())
		return
	}
	if vs.DefaultMaxThreads <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultMaxThreads 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "SetVarSettingsConfig: DefaultMaxThreads 参数错误，任务创建失败")
		return
	}
	if vs.DefaultBiliDownloadGoRoutines <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultBiliDownloadGoRoutines 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "SetVarSettingsConfig: DefaultBiliDownloadGoRoutines 参数错误，任务创建失败")
		return
	}
	if vs.DefaultBiliDownloadsMaxQueueNum <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultBiliDownloadsMaxQueueNum 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "SetVarSettingsConfig: DefaultBiliDownloadsMaxQueueNum 参数错误，任务创建失败")
		return
	}
	if vs.DefaultTaskWorkerGoRoutines <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultTaskWorkerGoRoutines 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "SetVarSettingsConfig: DefaultTaskWorkerGoRoutines 参数错误，任务创建失败")
		return
	}
	if vs.DefaultDbCrontabSeconds <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultDbCrontabSeconds 参数错误，任务创建失败"})
		common.LogPrintln("", common.ErStr, "SetVarSettingsConfig: DefaultDbCrontabSeconds 参数错误，任务创建失败")
		return
	}
	common.VarSettingsVariable = *vs
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功修改设置")})
	common.LogPrintln("", common.WebStr, "成功修改设置")
}

func GetVarSettingsConfig(c *gin.Context) {
	c.JSON(http.StatusOK, common.VarSettingsVariable)
	return
}

func TaskWorkerInit() {
	DlTaskWorkerInit()
	BDlTaskWorkerInit()
	AddTaskWorkerInit()
	GetTaskWorkerInit()
	BUlTaskWorkerInit()
}

func Cors() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method
		context.Header("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Headers", "*")
		context.Header("Access-Control-Allow-Methods", "*")
		context.Header("Access-Control-Expose-Headers", "*")
		context.Header("Access-Control-Allow-Credentials", "true")
		// 允许放行OPTIONS请求
		if method == "OPTIONS" {
			context.AbortWithStatus(http.StatusNoContent)
		}
		context.Next()
	}
}

func WebServerInit(host string, port int) {
	DbInit()
	TaskWorkerInit()
	GetUpdate()
	GetSystemResourceUsageInit()
	if !common.DefaultWebServerDebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20
	r.Use(Cors())

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui")
	})
	r.StaticFS("/ui", http.FS(common.UISubFiles))
	r.POST("/api/upload/encode", UploadEncode)
	r.POST("/api/upload/decode", UploadDecode)
	r.POST("/api/get-file-from-url", GetFileFromURL)
	r.POST("/api/get-bili-encoded-video-files", GetFileFromBiliID)
	r.GET("/api/get-dl-task-list", GetDlTaskList)
	r.GET("/api/get-file-list", GetFileList)
	r.POST("/api/add-encode-task", AddEncodeTask)
	r.GET("/api/get-add-task-list", GetAddTaskList)
	r.POST("/api/add-decode-task", AddDecodeTask)
	r.GET("/api/get-get-task-list", GetGetTaskList)
	r.GET("/api/get-logcat", GetLogCat)
	r.POST("/api/delete-file", DeleteFileFromAPI)
	r.POST("/api/rename-file", ReNameFileFromAPI)
	r.POST("/api/zip-file", zipFileFromAPI)
	r.POST("/api/copy-to-other-folder", CopyToOtherFolderFromAPI)
	r.POST("/api/unzip", UnzipFromAPI)
	r.GET("/api/dl/:dir/:file", DownloadFileFromAPI)
	r.GET("/api/clear-logcat", ClearLogCat)
	r.GET("/api/clear-dl-task-list", ClearDlTaskList)
	r.GET("/api/clear-add-task-list", ClearAddTaskList)
	r.GET("/api/clear-get-task-list", ClearGetTaskList)
	r.POST("/api/pause-add-task", PauseAddTask)
	r.POST("/api/pause-get-task", PauseGetTask)
	r.POST("/api/delete-dl-task", DeleteDlTask)
	r.POST("/api/delete-bdl-task", DeleteBDlTask)
	r.POST("/api/delete-add-task", DeleteAddTask)
	r.POST("/api/delete-get-task", DeleteGetTask)
	r.GET("/api/get-server-status", GetServerStatus)
	r.GET("/api/restart-server", RestartServer)
	r.GET("/api/get-bul-task-list", GetBUlTaskList)
	r.POST("/api/delete-bul-task", DeleteBUlTask)
	r.POST("/api/add-bul-task", AddBUlTask)
	r.GET("/api/clear-bul-task-list", ClearBUlTaskList)
	r.GET("/api/bilibili/qrcode", GetBilibiliQRCode)
	r.GET("/api/bilibili/poll/:ac", GetBilibiliPoll)
	r.GET("/api/version", GetVersion)
	r.POST("/api/set-var-settings-config", SetVarSettingsConfig)
	r.GET("/api/get-var-settings-config", GetVarSettingsConfig)
	r.POST("/api/get-update", UpdateFromAPI)
	r.POST("/api/update", UpdateRequestFromAPI)

	p := port
	for {
		if CheckPort(p) {
			common.LogPrintln("", common.WebStr, p, "端口已被占用")
			rand.Seed(time.Now().UnixNano())
			p = rand.Intn(common.DefaultWebServerRandomPortMax-common.DefaultWebServerRandomPortMin+1) + common.DefaultWebServerRandomPortMin
			common.LogPrintln("", common.WebStr, "尝试在", p, "端口上重新启动 Web Server")
			continue
		}
		break
	}
	common.LogPrintln("", common.WebStr, "Web Server 在 "+host+":"+strconv.Itoa(p)+" 上监听")
	common.LogPrintln("", common.WebStr, "尝试访问管理面板: http://localhost:"+strconv.Itoa(p)+"/ui/")
	err := r.Run(host + ":" + strconv.Itoa(p))
	if err != nil {
		common.LogPrintln("", common.WebStr, "Web Server 启动失败：", err)
		return
	}
}

func WebServer(host string, port int) {
	WebServerInit(host, port)
	<-make(chan int)
}
