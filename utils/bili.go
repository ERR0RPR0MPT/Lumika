package utils

import (
	"github.com/ERR0RPR0MPT/Lumika/biliup"
	"github.com/google/uuid"
	bg "github.com/iyear/biligo"
	"github.com/tidwall/gjson"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func BUlAddTask(bulTaskInfo *BUlTaskInfo) {
	uuidd := uuid.New().String()
	dt := &BUlTaskListData{
		UUID:         uuidd,
		TimeStamp:    time.Now().Format("2006-01-02 15:04:05"),
		FileName:     bulTaskInfo.FileName,
		TaskInfo:     bulTaskInfo,
		ProgressRate: 0,
	}
	BUlTaskList[uuidd] = dt
	BUlTaskQueue <- dt
}

func BDlAddTask(AVOrBVStr string, baseStr string, parentDir string) {
	uuidd := uuid.New().String()
	dt := &BDlTaskListData{
		UUID:       uuidd,
		TimeStamp:  time.Now().Format("2006-01-02 15:04:05"),
		ResourceID: AVOrBVStr,
		TaskInfo: &BDlTaskInfo{
			ResourceID: AVOrBVStr,
			ParentDir:  parentDir,
			BaseStr:    baseStr,
		},
		BaseStr:      baseStr,
		ProgressRate: 0,
	}
	BDlTaskList[uuidd] = dt
	BDlTaskQueue <- dt
}

func BUlTaskWorker(id int) {
	for task := range BUlTaskQueue {
		LogPrintf(task.UUID, "BUlTaskWorker %d 处理哔哩源上传任务：%v\n", id, task)
		_, exist := BUlTaskList[task.UUID]
		if !exist {
			LogPrintf(task.UUID, "BUlTaskWorker %d 上传任务被用户删除\n", id)
			continue
		}
		BUlTaskList[task.UUID].Status = "正在执行"
		BUlTaskList[task.UUID].StatusMsg = "正在执行"
		bvid, err := BUl(filepath.Join(LumikaEncodeOutputPath, task.FileName), *task.TaskInfo.Cookie, task.TaskInfo.UploadLines, task.TaskInfo.Threads, task.TaskInfo.VideoInfos, task.UUID)
		_, exist = BUlTaskList[task.UUID]
		if !exist {
			LogPrintf(task.UUID, "BUlTaskWorker %d 上传任务被用户删除\n", id)
			continue
		}
		if err != nil {
			LogPrintf(task.UUID, "BUlTaskWorker %d 哔哩源上传任务执行失败\n", id)
			BUlTaskList[task.UUID].Status = "执行失败"
			BUlTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		LogPrintln(task.UUID, "BUlTaskWorker", "获取到上传视频的 BV 号:", bvid)
		BUlTaskList[task.UUID].BVID = bvid
		BUlTaskList[task.UUID].Status = "已完成"
		BUlTaskList[task.UUID].StatusMsg = "已完成"
		BUlTaskList[task.UUID].ProgressNum = 100.0
	}
}

func BDlTaskWorker(id int) {
	for task := range BDlTaskQueue {
		LogPrintf(task.UUID, "BDlTaskWorker %d 处理哔哩源下载任务：%v\n", id, task)
		_, exist := BDlTaskList[task.UUID]
		if !exist {
			LogPrintf(task.UUID, "BDlTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		BDlTaskList[task.UUID].Status = "正在执行"
		BDlTaskList[task.UUID].StatusMsg = "正在执行"
		err := BDl(task.ResourceID, task.TaskInfo.ParentDir, task.UUID)
		_, exist = BDlTaskList[task.UUID]
		if !exist {
			LogPrintf(task.UUID, "BDlTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		if err != nil {
			LogPrintf(task.UUID, "BDlTaskWorker %d 哔哩源下载任务执行失败\n", id)
			BDlTaskList[task.UUID].Status = "执行失败"
			BDlTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		BDlTaskList[task.UUID].Status = "已完成"
		BDlTaskList[task.UUID].StatusMsg = "已完成"
		BDlTaskList[task.UUID].ProgressNum = 100.0
	}
}

func BUlTaskWorkerInit() {
	BUlTaskQueue = make(chan *BUlTaskListData)
	BUlTaskList = make(map[string]*BUlTaskListData)
	if len(database.BUlTaskList) != 0 {
		BUlTaskList = database.BUlTaskList
		for kp, kq := range BUlTaskList {
			if kq.Status == "正在执行" {
				BUlTaskList[kp].Status = "执行失败"
				BUlTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				BUlTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 BUlTaskWorker 协程来处理任务
	for i := 0; i < DefaultTaskWorkerGoRoutines; i++ {
		go BUlTaskWorker(i)
	}
}

func BDlTaskWorkerInit() {
	BDlTaskQueue = make(chan *BDlTaskListData)
	BDlTaskList = make(map[string]*BDlTaskListData)
	if len(database.BDlTaskList) != 0 {
		BDlTaskList = database.BDlTaskList
		for kp, kq := range BDlTaskList {
			if kq.Status == "正在执行" {
				BDlTaskList[kp].Status = "执行失败"
				BDlTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				BDlTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 BDlTaskWorker 协程来处理任务
	for i := 0; i < DefaultTaskWorkerGoRoutines; i++ {
		go BDlTaskWorker(i)
	}
}

func BUl(filePath string, bu biliup.User, uploadLines string, threads int, videoInfos biliup.VideoInfos, UUID string) (string, error) {
	b, err := biliup.New(bu)
	if err != nil {
		LogPrintln(UUID, BUlStr, "Cookie 校验失败，请检查 Cookie 是否失效:", err)
		return "", &CommonError{Msg: "Cookie 校验失败，请检查 Cookie 是否失效: " + err.Error()}
	}
	b.UploadLines = uploadLines
	b.Threads = threads
	b.VideoInfos = videoInfos

	LogPrintln(UUID, BUlStr, "Cookie 校验成功，开始上传编码视频列表")
	LogPrintln(UUID, BUlStr, "注意：此过程没有进度回显，请耐心等待执行完毕")
	reqBody, _, err := biliup.UploadFolderWithSubmit(filePath, *b)
	if err != nil {
		LogPrintln(UUID, BUlStr, "上传失败", err)
		return "", &CommonError{Msg: "上传失败: " + err.Error()}
	}
	reqBodyBytes, err := InterfaceToBytes(reqBody)
	if err != nil {
		LogPrintln(UUID, BUlStr, "获取视频 BV 号失败", err)
		return "", &CommonError{Msg: "获取视频 BV 号失败: " + err.Error()}
	}
	bvidStr := ""
	result := gjson.Get(string(reqBodyBytes), "data.bvid")
	if result.Exists() {
		bvidStr = result.String()
		LogPrintln(UUID, BUlStr, "上传成功，获取到 BV 号:", bvidStr)
	} else {
		bvidStr = "未知"
		LogPrintln(UUID, BUlStr, "上传出错，没有获取到 BV 号")
	}
	LogPrintln(UUID, BUlStr, "上传成功")
	return bvidStr, nil
}

func BDl(AVOrBVStr string, parentDir, UUID string) error {
	var aid int64
	if len(AVOrBVStr) <= 2 {
		LogPrintln(UUID, BDlStr, "未知的视频编号:", AVOrBVStr)
		return &CommonError{Msg: "未知的视频编号"}
	}
	if !strings.Contains(AVOrBVStr[:2], "av") {
		if len(AVOrBVStr) != 12 {
			LogPrintln(UUID, BDlStr, "未知的视频编号:", AVOrBVStr)
			return &CommonError{Msg: "未知的视频编号"}
		}
		aid = bg.BV2AV(AVOrBVStr)
	} else {
		anum, err := strconv.ParseInt(AVOrBVStr[2:], 10, 64)
		if err != nil {
			LogPrintln(UUID, BDlStr, "转换失败:", err)
			return &CommonError{Msg: "转换失败:" + err.Error()}
		}
		aid = anum
	}
	b := bg.NewCommClient(&bg.CommSetting{})
	info, err := b.VideoGetInfo(aid)
	if err != nil {
		LogPrintln(UUID, BDlStr, "VideoGetInfo 失败:", err)
		return &CommonError{Msg: "VideoGetInfo 失败:" + err.Error()}
	}
	LogPrintln(UUID, BDlStr, "视频标题:", info.Title)
	LogPrintln(UUID, BDlStr, "视频 aid:", info.AID)
	LogPrintln(UUID, BDlStr, "视频 BVid:", info.BVID)
	LogPrintln(UUID, BDlStr, "视频 cid:", info.CID)
	LogPrintln(UUID, BDlStr, "视频简介", info.Desc)
	LogPrintln(UUID, BDlStr, "总时长:", info.Duration)
	LogPrintln(UUID, BDlStr, "视频分 P 数量:", len(info.Pages))

	LogPrintln(UUID, BDlStr, "创建下载目录...")
	SuitableDirName := ReplaceInvalidCharacters(info.BVID, '-')
	SuitableFileName := ReplaceInvalidCharacters(info.Title, '-')
	// 检查是否已经存在下载目录
	if _, err := os.Stat(filepath.Join(LumikaWorkDirPath, parentDir, SuitableDirName)); err == nil {
		LogPrintln(UUID, BDlStr, "下载目录已存在，跳过创建下载目录")
	} else if os.IsNotExist(err) {
		LogPrintln(UUID, BDlStr, "下载目录不存在，创建下载目录")
		// 创建目录
		err = os.Mkdir(filepath.Join(LumikaWorkDirPath, parentDir, SuitableDirName), 0755)
		if err != nil {
			LogPrintln(UUID, BDlStr, "创建下载目录失败:", err)
			return &CommonError{Msg: "创建下载目录失败:" + err.Error()}
		}
	} else {
		LogPrintln(UUID, BDlStr, "检查下载目录失败:", err)
		return &CommonError{Msg: "检查下载目录失败:" + err.Error()}
	}

	LogPrintln(UUID, BDlStr, "遍历所有分 P ...")
	// 启动多个goroutine
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, DefaultBiliDownloadsMaxQueueNum)
	allStartTime := time.Now()
	for pi := range info.Pages {
		LogPrintln(UUID, BDlStr, "尝试获取 "+strconv.Itoa(pi+1)+"P 的视频地址...")
		wg.Add(1)               // 增加计数器
		semaphore <- struct{}{} // 协程获取信号量，若已满则阻塞
		go func(pi int) {
			defer func() {
				<-semaphore // 协程释放信号量
				wg.Done()
			}()
			videoPlayURLResult, err := b.VideoGetPlayURL(aid, info.Pages[pi].CID, 16, 1)
			if err != nil {
				LogPrintln(UUID, BDlStr, "获取视频地址失败，跳过本分P视频")
				return
			}
			durl := videoPlayURLResult.DURL[0].URL
			videoName := strconv.Itoa(pi+1) + "-" + SuitableFileName + ".mp4"
			filePath := filepath.Join(LumikaWorkDirPath, parentDir, SuitableDirName, videoName)
			LogPrintln(UUID, BDlStr, "视频地址:", durl)
			LogPrintln(UUID, BDlStr, "尝试下载视频...")
			err = Dl(durl, filePath, DefaultBiliDownloadReferer, DefaultUserAgent, DefaultBiliDownloadGoRoutines, "")
			if err != nil {
				LogPrintln(UUID, BDlStr, "下载视频("+videoName+")失败，跳过本分P视频:", err)
				return
			}
			if UUID != "" {
				_, exist := BDlTaskList[UUID]
				if exist {
					// 为全局 ProgressRate 变量赋值
					BDlTaskList[UUID].ProgressRate++
					// 计算正确的 progressNum
					BDlTaskList[UUID].ProgressNum = float64(BDlTaskList[UUID].ProgressRate) / float64(len(info.Pages)) * 100
				} else {
					LogPrintln(UUID, BDlStr, ErStr, "当前任务被用户删除", err)
					return
				}
			}
		}(pi)
	}
	wg.Wait()
	if UUID != "" {
		_, exist := BDlTaskList[UUID]
		if !exist {
			LogPrintln(UUID, BDlStr, ErStr, "当前任务被用户删除", err)
			return &CommonError{Msg: "当前任务被用户删除"}
		}
	}
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	LogPrintf(UUID, BDlStr+" 总共耗时%f秒\n", allDuration.Seconds())
	LogPrintln(UUID, BDlStr, "视频全部下载完成")
	return nil
}
