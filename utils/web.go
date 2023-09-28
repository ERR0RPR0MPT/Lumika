package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
		dst := path.Join("./lumika_data/encode", file.Filename)
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
	targetDir := filepath.Join("lumika_data", "decode", folderName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		err := os.Mkdir(targetDir, os.ModePerm)
		if err != nil {
			return
		}
	}

	// 获取所有文件
	files := form.File["directory"]
	// 遍历所有文件
	for _, file := range files {
		// 上传文件至指定目录
		dst := filepath.Join("lumika_data", "decode", folderName, file.Filename)
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
	useSingleThreadToDownload := c.PostForm("useSingleThreadToDownload")
	fmt.Println("读取到 useSingleThreadToDownload:", useSingleThreadToDownload)
	ua := DefaultBiliDownloadGoRoutines
	if useSingleThreadToDownload == "true" {
		ua = 1
	}
	fileName = ReplaceInvalidCharacters(fileName, '-')
	filePath := filepath.Join("lumika_data", "encode", fileName)
	DlAddTask(url, filePath, "", DefaultUserAgent, ua)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("成功添加下载任务: %s, 使用线程数: %d", fileName, ua)})
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
	c.JSON(http.StatusOK, gin.H{"data": DlTaskList})
}

func WebServerInit() {
	DlTaskWorkerInit()
	BDlTaskWorkerInit()
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
	fmt.Println(WebStr, "Web Server 在 "+DefaultWebServerBindAddress+" 上监听")
	fmt.Println(WebStr, "尝试访问管理面板: http://127.0.0.1:7860/")
	err := r.Run(DefaultWebServerBindAddress)
	if err != nil {
		fmt.Println(WebStr, "Web Server 启动失败：", err)
		return
	}
}

func WebServer() {
	WebServerInit()
	<-make(chan int)
}
