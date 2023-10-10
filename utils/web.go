package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func UploadEncode(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("出现错误: %s", err.Error()))
		return
	}
	parentDir := c.PostForm("parentDir")
	LogPrintln("", "读取到 parentDir:", parentDir)
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		return
	}
	// 获取所有文件
	files := form.File["files"]
	// 遍历所有文件
	for _, file := range files {
		// 上传文件至指定目录
		dst := filepath.Join(LumikaWorkDirPath, parentDir, file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("上传失败: %s", err.Error()))
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("文件上传成功: 已上传 %d 个文件", len(files))})
}

func UploadDecode(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("出现错误: %s", err.Error())})
		return
	}
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		return
	}

	folderName := c.PostForm("folderName")
	if folderName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "上传的文件夹名称不能为空"})
		return
	}
	folderName = ReplaceInvalidCharacters(folderName, '-')

	// 创建目标文件夹（如果不存在）
	targetDir := filepath.Join(LumikaWorkDirPath, parentDir, folderName)
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
		dst := filepath.Join(LumikaWorkDirPath, parentDir, folderName, file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("上传失败: %s", err.Error()))
			return
		}
	}
	c.JSON(http.StatusOK, fmt.Sprintf("目录上传成功: 已上传 %d 个文件", len(files)))
}

func GetFileFromURL(c *gin.Context) {
	urla := c.PostForm("url")
	if urla == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL 不能为空"})
		return
	}
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		return
	}
	fileName := c.PostForm("fileName")
	if fileName == "" {
		fileName = GetFileNameFromURL(urla)
	}
	DownloadThreadNumInputData := c.PostForm("DownloadThreadNumInputData")
	LogPrintln("", "读取到 DownloadThreadNumInputData:", DownloadThreadNumInputData)
	gor := VarSettingsVariable.DefaultBiliDownloadGoRoutines
	gors, err := strconv.Atoi(DownloadThreadNumInputData)
	if err != nil {
		if DownloadThreadNumInputData == "" {
			gors = VarSettingsVariable.DefaultBiliDownloadGoRoutines
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "线程参数错误"})
			return
		}
	}
	if gors > 0 {
		gor = gors
	}
	fileName = ReplaceInvalidCharacters(fileName, '-')
	filePath := filepath.Join(LumikaWorkDirPath, parentDir, fileName)
	go DlAddTask(urla, filePath, "", "", gor)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s, 使用线程数: %d", fileName, gor)})
}

func GetFileFromBiliID(c *gin.Context) {
	AVOrBVStr := c.PostForm("biliId")
	if AVOrBVStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BV/av号不能为空"})
		return
	}
	baseStr := c.PostForm("baseStr")
	parentDir := c.PostForm("parentDir")
	if parentDir != "encode" && parentDir != "encodeOutput" && parentDir != "decode" && parentDir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "父目录参数不正确"})
		return
	}
	go BDlAddTask(AVOrBVStr, baseStr, parentDir)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s", AVOrBVStr)})
}

func GetDlTaskList(c *gin.Context) {
	var DlTaskListArray []*DlTaskListData
	for _, kq := range DlTaskList {
		DlTaskListArray = append(DlTaskListArray, kq)
	}
	var BDlTaskListArray []*BDlTaskListData
	for _, kq := range BDlTaskList {
		BDlTaskListArray = append(BDlTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"dlTaskList": DlTaskListArray, "bdlTaskList": BDlTaskListArray})
}

func GetFileList(c *gin.Context) {
	EncodeDirData, err := GetDirectoryJSON(LumikaEncodePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		return
	}
	EncodeOutputDirData, err := GetDirectoryJSON(LumikaEncodeOutputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		return
	}
	DecodeDirData, err := GetDirectoryJSON(LumikaDecodePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		return
	}
	DecodeOutputDirData, err := GetDirectoryJSON(LumikaDecodeOutputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"读取文件出现错误:": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"encode": EncodeDirData, "encodeOutput": EncodeOutputDirData, "decode": DecodeDirData, "decodeOutput": DecodeOutputDirData})
}

func AddEncodeTask(c *gin.Context) {
	var ed *AddTaskInfo
	if err := c.ShouldBindJSON(&ed); err != nil {
		c.JSON(400, gin.H{"msg": "AddEncodeTask JSON 解析错误", "error": err.Error()})
		return
	}
	if ed.DefaultM == 0 {
		LogPrintln("", AddStr, ErStr, "错误: M 的值不能为 0，自动设置 M = "+strconv.Itoa(AddMLevel))
		ed.DefaultM = AddMLevel
	}
	if ed.DefaultK == 0 {
		LogPrintln("", AddStr, ErStr, "错误: K 的值不能为 0，自动设置 K = "+strconv.Itoa(AddKLevel))
		ed.DefaultK = AddKLevel
	}
	if ed.DefaultK > ed.DefaultM {
		LogPrintln("", AddStr, ErStr, "错误: K 的值不能大于 M 的值，自动设置 K = M = "+strconv.Itoa(ed.DefaultM))
		ed.DefaultK = ed.DefaultM
	}
	if ed.MGValue == 0 {
		LogPrintln("", AddStr, ErStr, "错误: MG 的值不能为 0，自动设置 MG = "+strconv.Itoa(AddMGLevel))
		ed.MGValue = AddMGLevel
	}
	if ed.KGValue == 0 {
		LogPrintln("", AddStr, ErStr, "错误: KG 的值不能为 0，自动设置 KG = "+strconv.Itoa(AddKGLevel))
		ed.KGValue = AddKGLevel
	}
	if ed.KGValue > ed.MGValue {
		LogPrintln("", AddStr, ErStr, "错误: KG 的值不能大于 MG 的值，自动设置 KG = MG = "+strconv.Itoa(ed.MGValue))
		ed.KGValue = ed.MGValue
	}
	if ed.VideoSize <= 0 {
		LogPrintln("", AddStr, ErStr, "错误: 分辨率大小不能小于等于 0，自动设置分辨率大小为", EncodeVideoSizeLevel)
		ed.VideoSize = EncodeVideoSizeLevel
	}
	if ed.OutputFPS <= 0 {
		LogPrintln("", AddStr, ErStr, "错误: 输出帧率不能小于等于 0，自动设置输出帧率为", EncodeOutputFPSLevel)
		ed.OutputFPS = EncodeOutputFPSLevel
	}
	if ed.EncodeMaxSeconds <= 0 {
		LogPrintln("", AddStr, ErStr, "错误: 最大编码时间不能小于等于 0，自动设置最大编码时间为", EncodeMaxSecondsLevel)
		ed.EncodeMaxSeconds = EncodeMaxSecondsLevel
	}
	if ed.EncodeThread <= 0 {
		LogPrintln("", AddStr, ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", VarSettingsVariable.DefaultMaxThreads)
		ed.EncodeThread = VarSettingsVariable.DefaultMaxThreads
	}
	go AddAddTask(ed.FileNameList, ed.DefaultM, ed.DefaultK, ed.MGValue, ed.KGValue, ed.VideoSize, ed.OutputFPS, ed.EncodeMaxSeconds, ed.EncodeThread, ed.EncodeFFmpegMode, ed.DefaultSummary)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加编码任务")})
}

func GetAddTaskList(c *gin.Context) {
	var AddTaskListArray []*AddTaskListData
	for _, kq := range AddTaskList {
		AddTaskListArray = append(AddTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"encodeTaskList": AddTaskListArray})
}

func AddDecodeTask(c *gin.Context) {
	var ed *GetTaskInfo
	if err := c.ShouldBindJSON(&ed); err != nil {
		c.JSON(400, gin.H{"msg": "AddDecodeTask JSON 解析错误", "error": err.Error()})
		return
	}
	if ed.DirName == "" {
		c.JSON(400, gin.H{"msg": "AddDecodeTask: DirName 参数错误，任务创建失败"})
		return
	}
	if ed.DecodeThread <= 0 {
		LogPrintln("", AddStr, ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", VarSettingsVariable.DefaultMaxThreads)
		ed.DecodeThread = VarSettingsVariable.DefaultMaxThreads
	}
	go AddGetTask(ed.DirName, ed.DecodeThread, ed.BaseStr)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加解码任务")})
}

func GetGetTaskList(c *gin.Context) {
	var GetTaskListArray []*GetTaskListData
	for _, kq := range GetTaskList {
		GetTaskListArray = append(GetTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"decodeTaskList": GetTaskListArray})
}

func GetLogCat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"logcat": LogVariable.String()})
}

func DeleteFileFromAPI(c *gin.Context) {
	dir := c.PostForm("dir")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		return
	}
	fileName := c.PostForm("file")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file 参数错误：请输入正确的文件/目录名"})
		return
	}
	filePath := filepath.Join(LumikaWorkDirPath, dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件/目录不存在"})
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件/目录不存在"})
		return
	}
	if fileInfo.IsDir() {
		err = os.RemoveAll(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除目录失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "目录删除成功"})
		return
	} else {
		err = os.Remove(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "文件删除成功"})
		return
	}
}

func ReNameFileFromAPI(c *gin.Context) {
	dir := c.PostForm("dir")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		return
	}
	originName := c.PostForm("originName")
	if originName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 参数错误：请输入正确的文件/目录名"})
		return
	}
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 参数错误：请输入正确的文件/目录名"})
		return
	}
	originFilePath := filepath.Join(LumikaWorkDirPath, dir, originName)
	filePath := filepath.Join(LumikaWorkDirPath, dir, name)
	if _, err := os.Stat(originFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "originName 的文件/目录不存在"})
		return
	}
	if _, err := os.Stat(filePath); os.IsExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 的文件/目录已经存在，请换个名称"})
		return
	}
	err := os.Rename(originFilePath, filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "重命名失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "重命名成功"})
}

func DownloadFileFromAPI(c *gin.Context) {
	dir := c.Param("dir")
	fileName := c.Param("file")
	if dir != "encode" && dir != "encodeOutput" && dir != "decode" && dir != "decodeOutput" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dir 参数错误：请指定正确的目录"})
		return
	}
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file 参数错误：请输入正确的文件名"})
		return
	}
	filePath := filepath.Join(LumikaWorkDirPath, dir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不存在"})
		return
	}
	if !fileInfo.IsDir() {
		ext := filepath.Ext(fileName)
		contentType := mime.TypeByExtension(ext)
		if contentType != "" {
			c.Writer.Header().Set("Content-Type", contentType)
		}
		//c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		//c.Writer.Header().Set("Content-Disposition", "inline")
		c.File(filePath)
		return
	} else {
		zipFilePath := filepath.Join(filepath.Dir(filePath), fileName+".zip")
		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			err = ZipDirectory(filePath, zipFilePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "目录压缩失败"})
				return
			}
		}
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(zipFilePath)))
		c.File(zipFilePath)
		return
	}
}

func ClearLogCat(c *gin.Context) {
	LogVariable.Reset()
	c.JSON(http.StatusOK, gin.H{"message": "日志清除成功"})
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
	DlTaskList = make(map[string]*DlTaskListData)
	BDlTaskList = make(map[string]*BDlTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "下载任务列表清除成功"})
}
func ClearAddTaskList(c *gin.Context) {
	//for _, kq := range AddTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除编码任务列表"})
	//		return
	//	}
	//}
	AddTaskList = make(map[string]*AddTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "编码任务列表清除成功"})
}

func ClearGetTaskList(c *gin.Context) {
	//for _, kq := range GetTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除解码任务列表"})
	//		return
	//	}
	//}
	GetTaskList = make(map[string]*GetTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "解码任务列表清除成功"})
}

func PauseAddTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := AddTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	if AddTaskList[uuidd].IsPaused {
		AddTaskList[uuidd].IsPaused = false
		AddTaskList[uuidd].Status = "正在执行"
		AddTaskList[uuidd].StatusMsg = "正在执行"
		c.JSON(http.StatusOK, gin.H{"message": "成功启动任务"})
		return
	} else {
		AddTaskList[uuidd].IsPaused = true
		AddTaskList[uuidd].Status = "已暂停"
		AddTaskList[uuidd].StatusMsg = "任务已暂停"
		c.JSON(http.StatusOK, gin.H{"message": "成功暂停任务"})
		return
	}
}

func PauseGetTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := GetTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	if GetTaskList[uuidd].IsPaused {
		GetTaskList[uuidd].IsPaused = false
		GetTaskList[uuidd].Status = "正在执行"
		GetTaskList[uuidd].StatusMsg = "正在执行"
		c.JSON(http.StatusOK, gin.H{"message": "成功启动任务"})
		return
	} else {
		GetTaskList[uuidd].IsPaused = true
		GetTaskList[uuidd].Status = "已暂停"
		GetTaskList[uuidd].StatusMsg = "任务已暂停"
		c.JSON(http.StatusOK, gin.H{"message": "成功暂停任务"})
		return
	}
}

func DeleteDlTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := DlTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	delete(DlTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	return
}

func DeleteBDlTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := BDlTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	delete(BDlTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	return
}

func DeleteAddTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := AddTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	delete(AddTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	return
}

func DeleteGetTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := GetTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	delete(GetTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	return
}

func GetServerStatus(c *gin.Context) {
	usage, err := GetSystemResourceUsage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取服务器资源使用情况失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": usage})
	return
}

func RestartServer(c *gin.Context) {
	err := RestartProgram()
	if err != nil {
		LogPrintln("", WebStr, "重启服务器后端进程失败：", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重启服务器后端进程失败"})
		return
	}
	LogPrintln("", WebStr, "服务器后端进程重启成功")
	c.JSON(http.StatusOK, gin.H{"error": "服务器后端进程重启成功"})
	return
}

func GetBUlTaskList(c *gin.Context) {
	var BUlTaskListArray []*BUlTaskListData
	for _, kq := range BUlTaskList {
		BUlTaskListArray = append(BUlTaskListArray, kq)
	}
	c.JSON(http.StatusOK, gin.H{"bulTaskList": BUlTaskListArray})
}

func DeleteBUlTask(c *gin.Context) {
	uuidd := c.PostForm("uuid")
	if uuidd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务的 UUID 不能为空"})
		return
	}
	_, exist := BUlTaskList[uuidd]
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有找到指定的任务"})
		return
	}
	delete(BUlTaskList, uuidd)
	c.JSON(http.StatusOK, gin.H{"message": "成功删除任务"})
	return
}

func AddBUlTask(c *gin.Context) {
	var ed *BUlTaskInfo
	if err := c.ShouldBindJSON(&ed); err != nil {
		c.JSON(400, gin.H{"msg": "AddBUlTask JSON 解析错误", "error": err.Error()})
		return
	}
	if ed.FileName == "" {
		c.JSON(400, gin.H{"msg": "AddBUlTask: FileName 参数错误，任务创建失败"})
		return
	}
	if ed.UploadLines == "" {
		ed.UploadLines = DefaultBiliUploadLines
	}
	if ed.Threads <= 0 || ed.Threads > 256 {
		ed.Threads = DefaultBiliUploadThreads
	}
	if ed.VideoInfos.Title == "" {
		c.JSON(400, gin.H{"msg": "AddBUlTask: VideoInfos.Title 参数错误，任务创建失败"})
		return
	}
	if ed.VideoInfos.Copyright != 1 && ed.VideoInfos.Copyright != 2 {
		ed.VideoInfos.Copyright = 1
	}
	go BUlAddTask(ed)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加解码任务")})
}

func ClearBUlTaskList(c *gin.Context) {
	//for _, kq := range BUlTaskList {
	//	if kq.Status == "正在执行" || kq.Status == "已暂停" {
	//		c.JSON(http.StatusBadRequest, gin.H{"error": "有正在执行的任务，无法清除解码任务列表"})
	//		return
	//	}
	//}
	BUlTaskList = make(map[string]*BUlTaskListData)
	c.JSON(http.StatusOK, gin.H{"message": "任务列表清除成功"})
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
	c.JSON(http.StatusOK, gin.H{"version": LumikaVersionString})
	return
}

func SetVarSettingsConfig(c *gin.Context) {
	var vs *VarSettings
	if err := c.ShouldBindJSON(&vs); err != nil {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig JSON 解析错误", "error": err.Error()})
		return
	}
	if vs.DefaultMaxThreads <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultMaxThreads 参数错误，任务创建失败"})
		return
	}
	if vs.DefaultBiliDownloadGoRoutines <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultBiliDownloadGoRoutines 参数错误，任务创建失败"})
		return
	}
	if vs.DefaultBiliDownloadsMaxQueueNum <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultBiliDownloadsMaxQueueNum 参数错误，任务创建失败"})
		return
	}
	if vs.DefaultTaskWorkerGoRoutines <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultTaskWorkerGoRoutines 参数错误，任务创建失败"})
		return
	}
	if vs.DefaultDbCrontabSeconds <= 0 {
		c.JSON(400, gin.H{"msg": "SetVarSettingsConfig: DefaultDbCrontabSeconds 参数错误，任务创建失败"})
		return
	}
	VarSettingsVariable = *vs
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功修改设置")})
}

func GetVarSettingsConfig(c *gin.Context) {
	c.JSON(http.StatusOK, VarSettingsVariable)
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
	GetSystemResourceUsageInit()
	if !DefaultWebServerDebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20
	r.Use(Cors())

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui")
	})
	r.StaticFS("/ui", http.FS(UISubFiles))
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

	p := port
	for {
		if CheckPort(p) {
			LogPrintln("", WebStr, p, "端口已被占用")
			rand.Seed(time.Now().UnixNano())
			p = rand.Intn(DefaultWebServerRandomPortMax-DefaultWebServerRandomPortMin+1) + DefaultWebServerRandomPortMin
			LogPrintln("", WebStr, "尝试在", p, "端口上重新启动 Web Server")
			continue
		}
		break
	}
	LogPrintln("", WebStr, "Web Server 在 "+host+":"+strconv.Itoa(p)+" 上监听")
	LogPrintln("", WebStr, "尝试访问管理面板: http://localhost:"+strconv.Itoa(p)+"/ui/")
	err := r.Run(host + ":" + strconv.Itoa(p))
	if err != nil {
		LogPrintln("", WebStr, "Web Server 启动失败：", err)
		return
	}
}

func WebServer(host string, port int) {
	WebServerInit(host, port)
	<-make(chan int)
}
