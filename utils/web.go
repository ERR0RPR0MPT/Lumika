package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

func UploadEncode(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("出现错误: %s", err.Error()))
		return
	}
	// 获取所有文件
	files := form.File["files"]
	// 遍历所有文件
	for _, file := range files {
		// 上传文件至指定目录
		dst := filepath.Join(LumikaEncodePath, file.Filename)
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

	folderName := c.PostForm("folderName")
	if folderName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "上传的文件夹名称不能为空"})
		return
	}
	folderName = ReplaceInvalidCharacters(folderName, '-')

	// 创建目标文件夹（如果不存在）
	targetDir := filepath.Join(LumikaDecodePath, folderName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err := os.Mkdir(targetDir, 0644)
		if err != nil {
			return
		}
	}

	// 获取所有文件
	files := form.File["directory"]
	// 遍历所有文件
	for _, file := range files {
		// 上传文件至指定目录
		dst := filepath.Join(LumikaDecodePath, folderName, file.Filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("上传失败: %s", err.Error()))
			return
		}
	}
	c.JSON(http.StatusOK, fmt.Sprintf("目录上传成功: 已上传 %d 个文件", len(files)))
}

func GetFileFromURL(c *gin.Context) {
	url := c.PostForm("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL 不能为空"})
		return
	}
	fileName := c.PostForm("fileName")
	if fileName == "" {
		fileName = GetFileNameFromURL(url)
	}
	DownloadThreadNumInputData := c.PostForm("DownloadThreadNumInputData")
	LogPrint("", "读取到 DownloadThreadNumInputData:", DownloadThreadNumInputData)
	gor := DefaultBiliDownloadGoRoutines
	gors, err := strconv.Atoi(DownloadThreadNumInputData)
	if err != nil {
		if DownloadThreadNumInputData == "" {
			gors = DefaultBiliDownloadGoRoutines
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "线程参数错误"})
			return
		}
	}
	if gors > 0 {
		gor = gors
	}
	fileName = ReplaceInvalidCharacters(fileName, '-')
	filePath := filepath.Join(LumikaEncodePath, fileName)
	DlAddTask(url, filePath, "", DefaultUserAgent, gor)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s, 使用线程数: %d", fileName, gor)})
}

func GetFileFromBiliID(c *gin.Context) {
	AVOrBVStr := c.PostForm("bili-id")
	if AVOrBVStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BV/av号不能为空"})
		return
	}
	BDlAddTask(AVOrBVStr)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s", AVOrBVStr)})
}

func GetDlTaskList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"dlTaskList": DlTaskList, "bdlTaskList": BDlTaskList})
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
		LogPrint("", AddStr, ErStr, "错误: M 的值不能为 0，自动设置 M = "+strconv.Itoa(AddMLevel))
		ed.DefaultM = AddMLevel
	}
	if ed.DefaultK == 0 {
		LogPrint("", AddStr, ErStr, "错误: K 的值不能为 0，自动设置 K = "+strconv.Itoa(AddKLevel))
		ed.DefaultK = AddKLevel
	}
	if ed.DefaultK > ed.DefaultM {
		LogPrint("", AddStr, ErStr, "错误: K 的值不能大于 M 的值，自动设置 K = M = "+strconv.Itoa(ed.DefaultM))
		ed.DefaultK = ed.DefaultM
	}
	if ed.MGValue == 0 {
		LogPrint("", AddStr, ErStr, "错误: MG 的值不能为 0，自动设置 MG = "+strconv.Itoa(AddMGLevel))
		ed.MGValue = AddMGLevel
	}
	if ed.KGValue == 0 {
		LogPrint("", AddStr, ErStr, "错误: KG 的值不能为 0，自动设置 KG = "+strconv.Itoa(AddKGLevel))
		ed.KGValue = AddKGLevel
	}
	if ed.KGValue > ed.MGValue {
		LogPrint("", AddStr, ErStr, "错误: KG 的值不能大于 MG 的值，自动设置 KG = MG = "+strconv.Itoa(ed.MGValue))
		ed.KGValue = ed.MGValue
	}
	if ed.VideoSize <= 0 {
		LogPrint("", AddStr, ErStr, "错误: 分辨率大小不能小于等于 0，自动设置分辨率大小为", EncodeVideoSizeLevel)
		ed.VideoSize = EncodeVideoSizeLevel
	}
	if ed.OutputFPS <= 0 {
		LogPrint("", AddStr, ErStr, "错误: 输出帧率不能小于等于 0，自动设置输出帧率为", EncodeOutputFPSLevel)
		ed.OutputFPS = EncodeOutputFPSLevel
	}
	if ed.EncodeMaxSeconds <= 0 {
		LogPrint("", AddStr, ErStr, "错误: 最大编码时间不能小于等于 0，自动设置最大编码时间为", EncodeMaxSecondsLevel)
		ed.EncodeMaxSeconds = EncodeMaxSecondsLevel
	}
	if ed.EncodeThread <= 0 {
		LogPrint("", AddStr, ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", runtime.NumCPU())
		ed.EncodeThread = runtime.NumCPU()
	}
	AddAddTask(ed.FileNameList, ed.DefaultM, ed.DefaultK, ed.MGValue, ed.KGValue, ed.VideoSize, ed.OutputFPS, ed.EncodeMaxSeconds, ed.EncodeThread, ed.EncodeFFmpegMode, ed.DefaultSummary)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加编码任务")})
}

func GetAddTaskList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"encodeTaskList": AddTaskList})
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
		LogPrint("", AddStr, ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", runtime.NumCPU())
		ed.DecodeThread = runtime.NumCPU()
	}
	AddGetTask(ed.DirName, ed.DecodeThread, ed.BaseStr)
	c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("成功添加解码任务")})
}

func GetGetTaskList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"decodeTaskList": GetTaskList})
}

func GetLogCat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"logcat": LogVariable.String()})
}

func TaskWorkerInit() {
	DlTaskWorkerInit()
	BDlTaskWorkerInit()
	AddTaskWorkerInit()
	GetTaskWorkerInit()
}

func WebServerInit() {
	TaskWorkerInit()
	if !DefaultWebServerDebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui")
	})
	r.StaticFS("/ui", http.Dir("./ui"))
	r.POST("/api/upload/encode", UploadEncode)
	r.POST("/api/upload/decode", UploadDecode)
	r.POST("/api/get-file-from-url", GetFileFromURL)
	r.POST("/api/get-encoded-video-files", GetFileFromBiliID)
	r.GET("/api/get-dl-task-list", GetDlTaskList)
	r.GET("/api/get-file-list", GetFileList)
	r.POST("/api/add-encode-task", AddEncodeTask)
	r.GET("/api/get-add-task-list", GetAddTaskList)
	r.POST("/api/add-decode-task", AddDecodeTask)
	r.GET("/api/get-get-task-list", GetGetTaskList)
	r.GET("/api/get-logcat", GetLogCat)
	LogPrint("", WebStr, "Web Server 在 "+DefaultWebServerBindAddress+" 上监听")
	LogPrint("", WebStr, "尝试访问管理面板: http://127.0.0.1:7860/")
	err := r.Run(DefaultWebServerBindAddress)
	if err != nil {
		LogPrint("", WebStr, "Web Server 启动失败：", err)
		return
	}
}

func WebServer() {
	WebServerInit()
	<-make(chan int)
}
