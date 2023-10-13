package utils

import (
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/biliup"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/google/uuid"
	bg "github.com/iyear/biligo"
	"github.com/tidwall/gjson"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func BUlAddTask(bulTaskInfo *common.BUlTaskInfo) {
	uuidd := uuid.New().String()
	dt := &common.BUlTaskListData{
		UUID:         uuidd,
		TimeStamp:    time.Now().Format("2006-01-02 15:04:05"),
		FileName:     bulTaskInfo.FileName,
		TaskInfo:     bulTaskInfo,
		ProgressRate: 0,
		Duration:     "",
	}
	common.BUlTaskList[uuidd] = dt
	common.BUlTaskQueue <- dt
}

func BDlAddTask(AVOrBVStr string, baseStr string, parentDir string) {
	uuidd := uuid.New().String()
	dt := &common.BDlTaskListData{
		UUID:       uuidd,
		TimeStamp:  time.Now().Format("2006-01-02 15:04:05"),
		ResourceID: AVOrBVStr,
		TaskInfo: &common.BDlTaskInfo{
			ResourceID: AVOrBVStr,
			ParentDir:  parentDir,
			BaseStr:    baseStr,
		},
		BaseStr:      baseStr,
		ProgressRate: 0,
		Duration:     "",
	}
	common.BDlTaskList[uuidd] = dt
	common.BDlTaskQueue <- dt
}

func BUlTaskWorker(id int) {
	for task := range common.BUlTaskQueue {
		allStartTime := time.Now()
		common.LogPrintf(task.UUID, "BUlTaskWorker %d 处理哔哩源上传任务：%v\n", id, task)
		_, exist := common.BUlTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "BUlTaskWorker %d 上传任务被用户删除\n", id)
			continue
		}
		common.BUlTaskList[task.UUID].Status = "正在执行"
		common.BUlTaskList[task.UUID].StatusMsg = "正在执行"
		bvid, err := BUl(filepath.Join(common.LumikaEncodeOutputPath, task.FileName), biliup.User(*task.TaskInfo.Cookie), task.TaskInfo.UploadLines, task.TaskInfo.Threads, biliup.VideoInfos(task.TaskInfo.VideoInfos), task.UUID)
		_, exist = common.BUlTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "BUlTaskWorker %d 上传任务被用户删除\n", id)
			continue
		}
		if err != nil {
			common.LogPrintf(task.UUID, "BUlTaskWorker %d 哔哩源上传任务执行失败\n", id)
			common.BUlTaskList[task.UUID].Status = "执行失败"
			common.BUlTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		common.LogPrintln(task.UUID, "BUlTaskWorker", "获取到上传视频的 BV 号:", bvid)
		common.BUlTaskList[task.UUID].BVID = bvid
		common.BUlTaskList[task.UUID].Status = "已完成"
		common.BUlTaskList[task.UUID].StatusMsg = "已完成"
		common.BUlTaskList[task.UUID].ProgressNum = 100.0
		common.BUlTaskList[task.UUID].Duration = fmt.Sprintf("%vs", int64(math.Floor(time.Now().Sub(allStartTime).Seconds())))
	}
}

func BDlTaskWorker(id int) {
	for task := range common.BDlTaskQueue {
		allStartTime := time.Now()
		common.LogPrintf(task.UUID, "BDlTaskWorker %d 处理哔哩源下载任务：%v\n", id, task)
		_, exist := common.BDlTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "BDlTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		common.BDlTaskList[task.UUID].Status = "正在执行"
		common.BDlTaskList[task.UUID].StatusMsg = "正在执行"
		err := BDl(task.ResourceID, task.TaskInfo.ParentDir, task.UUID)
		_, exist = common.BDlTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "BDlTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		if err != nil {
			common.LogPrintf(task.UUID, "BDlTaskWorker %d 哔哩源下载任务执行失败\n", id)
			common.BDlTaskList[task.UUID].Status = "执行失败"
			common.BDlTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		common.BDlTaskList[task.UUID].Status = "已完成"
		common.BDlTaskList[task.UUID].StatusMsg = "已完成"
		common.BDlTaskList[task.UUID].ProgressNum = 100.0
		common.BDlTaskList[task.UUID].Duration = fmt.Sprintf("%vs", int64(math.Floor(time.Now().Sub(allStartTime).Seconds())))
	}
}

func BUlTaskWorkerInit() {
	common.BUlTaskQueue = make(chan *common.BUlTaskListData)
	common.BUlTaskList = make(map[string]*common.BUlTaskListData)
	if len(common.DatabaseVariable.BUlTaskList) != 0 {
		common.BUlTaskList = common.DatabaseVariable.BUlTaskList
		for kp, kq := range common.BUlTaskList {
			if kq.Status == "正在执行" {
				common.BUlTaskList[kp].Status = "执行失败"
				common.BUlTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				common.BUlTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 BUlTaskWorker 协程来处理任务
	for i := 0; i < common.VarSettingsVariable.DefaultTaskWorkerGoRoutines; i++ {
		go BUlTaskWorker(i)
	}
}

func BDlTaskWorkerInit() {
	common.BDlTaskQueue = make(chan *common.BDlTaskListData)
	common.BDlTaskList = make(map[string]*common.BDlTaskListData)
	if len(common.DatabaseVariable.BDlTaskList) != 0 {
		common.BDlTaskList = common.DatabaseVariable.BDlTaskList
		for kp, kq := range common.BDlTaskList {
			if kq.Status == "正在执行" {
				common.BDlTaskList[kp].Status = "执行失败"
				common.BDlTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				common.BDlTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 BDlTaskWorker 协程来处理任务
	for i := 0; i < common.VarSettingsVariable.DefaultTaskWorkerGoRoutines; i++ {
		go BDlTaskWorker(i)
	}
}

func BUl(filePath string, bu biliup.User, uploadLines string, threads int, videoInfos biliup.VideoInfos, UUID string) (string, error) {
	b, err := biliup.New(bu)
	if err != nil {
		common.LogPrintln(UUID, common.BUlStr, "Cookie 校验失败，请检查 Cookie 是否失效:", err)
		return "", &common.CommonError{Msg: "Cookie 校验失败，请检查 Cookie 是否失效: " + err.Error()}
	}
	b.UploadLines = uploadLines
	b.Threads = threads
	b.VideoInfos = videoInfos

	common.LogPrintln(UUID, common.BUlStr, "Cookie 校验成功，开始上传编码视频列表")
	reqBody, _, err := biliup.UploadFolderWithSubmit(filePath, *b, UUID)
	if err != nil {
		common.LogPrintln(UUID, common.BUlStr, "上传失败", err)
		return "", &common.CommonError{Msg: "上传失败: " + err.Error()}
	}
	reqBodyBytes, err := InterfaceToBytes(reqBody)
	if err != nil {
		common.LogPrintln(UUID, common.BUlStr, "获取视频 BV 号失败", err)
		return "", &common.CommonError{Msg: "获取视频 BV 号失败: " + err.Error()}
	}
	bvidStr := ""
	result := gjson.Get(string(reqBodyBytes), "data.bvid")
	if result.Exists() {
		bvidStr = result.String()
		common.LogPrintln(UUID, common.BUlStr, "上传成功，获取到 BV 号:", bvidStr)
	} else {
		bvidStr = "未知"
		common.LogPrintln(UUID, common.BUlStr, "上传出错，没有获取到 BV 号")
	}
	common.LogPrintln(UUID, common.BUlStr, "上传成功")
	return bvidStr, nil
}

func BDl(AVOrBVStr string, parentDir, UUID string) error {
	var aid int64
	if len(AVOrBVStr) <= 2 {
		common.LogPrintln(UUID, common.BDlStr, "未知的视频编号:", AVOrBVStr)
		return &common.CommonError{Msg: "未知的视频编号"}
	}
	if !strings.Contains(AVOrBVStr[:2], "av") {
		if len(AVOrBVStr) != 12 {
			common.LogPrintln(UUID, common.BDlStr, "未知的视频编号:", AVOrBVStr)
			return &common.CommonError{Msg: "未知的视频编号"}
		}
		aid = bg.BV2AV(AVOrBVStr)
	} else {
		anum, err := strconv.ParseInt(AVOrBVStr[2:], 10, 64)
		if err != nil {
			common.LogPrintln(UUID, common.BDlStr, "转换失败:", err)
			return &common.CommonError{Msg: "转换失败:" + err.Error()}
		}
		aid = anum
	}
	refererURL := "https://www.bilibili.com/video/" + AVOrBVStr + "/"
	b := bg.NewCommClient(&bg.CommSetting{})
	info, err := b.VideoGetInfo(aid)
	if err != nil {
		common.LogPrintln(UUID, common.BDlStr, "VideoGetInfo 失败:", err)
		return &common.CommonError{Msg: "VideoGetInfo 失败:" + err.Error()}
	}
	common.LogPrintln(UUID, common.BDlStr, "视频标题:", info.Title)
	common.LogPrintln(UUID, common.BDlStr, "视频 aid:", info.AID)
	common.LogPrintln(UUID, common.BDlStr, "视频 BVid:", info.BVID)
	common.LogPrintln(UUID, common.BDlStr, "视频 cid:", info.CID)
	common.LogPrintln(UUID, common.BDlStr, "视频简介", info.Desc)
	common.LogPrintln(UUID, common.BDlStr, "总时长:", info.Duration)
	common.LogPrintln(UUID, common.BDlStr, "视频分 P 数量:", len(info.Pages))

	common.LogPrintln(UUID, common.BDlStr, "创建下载目录...")
	SuitableDirName := ReplaceInvalidCharacters(info.BVID, '-')
	SuitableFileName := ReplaceInvalidCharacters(info.Title, '-')
	// 检查是否已经存在下载目录
	if _, err := os.Stat(filepath.Join(common.LumikaWorkDirPath, parentDir, SuitableDirName)); err == nil {
		common.LogPrintln(UUID, common.BDlStr, "下载目录已存在，跳过创建下载目录")
	} else if os.IsNotExist(err) {
		common.LogPrintln(UUID, common.BDlStr, "下载目录不存在，创建下载目录")
		// 创建目录
		err = os.Mkdir(filepath.Join(common.LumikaWorkDirPath, parentDir, SuitableDirName), 0755)
		if err != nil {
			common.LogPrintln(UUID, common.BDlStr, "创建下载目录失败:", err)
			return &common.CommonError{Msg: "创建下载目录失败:" + err.Error()}
		}
	} else {
		common.LogPrintln(UUID, common.BDlStr, "检查下载目录失败:", err)
		return &common.CommonError{Msg: "检查下载目录失败:" + err.Error()}
	}

	common.LogPrintln(UUID, common.BDlStr, "遍历所有分 P ...")
	// 启动多个goroutine
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, common.VarSettingsVariable.DefaultBiliDownloadsMaxQueueNum)
	allStartTime := time.Now()
	for pi := range info.Pages {
		common.LogPrintln(UUID, common.BDlStr, "尝试获取 "+strconv.Itoa(pi+1)+"P 的视频地址...")
		wg.Add(1)               // 增加计数器
		semaphore <- struct{}{} // 协程获取信号量，若已满则阻塞
		go func(pi int) {
			defer func() {
				<-semaphore // 协程释放信号量
				wg.Done()
			}()
			isSuccess := false
			for i := 0; i < common.DefaultBiliDownloadMaxRetryTimes; i++ {
				videoPlayURLResult, err := b.VideoGetPlayURL(aid, info.Pages[pi].CID, 16, 1)
				if err != nil {
					common.LogPrintln(UUID, common.BDlStr, "获取视频地址失败")
					continue
				}
				durl := videoPlayURLResult.DURL[0].URL
				videoName := strconv.Itoa(pi+1) + "-" + SuitableFileName + ".mp4"
				filePath := filepath.Join(common.LumikaWorkDirPath, parentDir, SuitableDirName, videoName)
				common.LogPrintln(UUID, common.BDlStr, "视频地址:", durl)
				common.LogPrintln(UUID, common.BDlStr, "尝试下载视频...")
				//if !common.MobileMode {
				//	err = Dl(durl, filePath, refererURL, common.DefaultBiliDownloadOrigin, "", common.VarSettingsVariable.DefaultBiliDownloadGoRoutines, "")
				//} else {
				//	err = DlForAndroid(durl, filePath, refererURL, common.DefaultBiliDownloadOrigin, "", common.VarSettingsVariable.DefaultBiliDownloadGoRoutines, "")
				//}
				err = Dl(durl, filePath, refererURL, common.DefaultBiliDownloadOrigin, "", common.VarSettingsVariable.DefaultBiliDownloadGoRoutines, "")
				if err != nil {
					common.LogPrintln(UUID, common.BDlStr, "下载视频("+videoName+")失败，准备重试:", err)
					continue
				}
				if UUID != "" {
					_, exist := common.BDlTaskList[UUID]
					if exist {
						// 为全局 ProgressRate 变量赋值
						common.BDlTaskList[UUID].ProgressRate++
						// 计算正确的 progressNum
						common.BDlTaskList[UUID].ProgressNum = float64(common.BDlTaskList[UUID].ProgressRate) / float64(len(info.Pages)) * 100
					} else {
						common.LogPrintln(UUID, common.BDlStr, common.ErStr, "当前任务被用户删除", err)
						return
					}
				}
				isSuccess = true
				break
			}
			if !isSuccess {
				common.LogPrintln(UUID, common.BDlStr, "下载视频("+strconv.Itoa(pi+1)+")失败，跳过下载")
			}
		}(pi)
	}
	wg.Wait()
	if UUID != "" {
		_, exist := common.BDlTaskList[UUID]
		if !exist {
			common.LogPrintln(UUID, common.BDlStr, common.ErStr, "当前任务被用户删除", err)
			return &common.CommonError{Msg: "当前任务被用户删除"}
		}
	}
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	common.LogPrintf(UUID, common.BDlStr+" 总共耗时%f秒\n", allDuration.Seconds())
	common.LogPrintln(UUID, common.BDlStr, "视频全部下载完成")
	return nil
}
