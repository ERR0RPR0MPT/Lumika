package utils

import (
	"github.com/google/uuid"
	bg "github.com/iyear/biligo"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func BDlAddTask(AVOrBVStr string) {
	uuidd := uuid.New().String()
	dt := &BDlTaskListData{
		UUID:       uuidd,
		TimeStamp:  time.Now().Format("2006-01-02 15:04:05"),
		ResourceID: AVOrBVStr,
		TaskInfo: &BDlTaskInfo{
			ResourceID: AVOrBVStr,
		},
		ProgressRate: 0,
	}
	BDlTaskList = append(BDlTaskList, dt)
	BDlTaskQueue <- dt
}

func BDlTaskWorker(id int) {
	for task := range BDlTaskQueue {
		LogPrintf(task.UUID, "BDlTaskWorker %d 处理哔哩源下载任务：%v\n", id, task)
		i := 0
		for kp, kq := range BDlTaskList {
			if kq.UUID == task.UUID {
				i = kp
				break
			}
		}
		BDlTaskList[i].Status = "正在执行"
		BDlTaskList[i].StatusMsg = "正在执行"
		err := BDl(task.ResourceID, task.UUID)
		if err != nil {
			LogPrintf(task.UUID, "BDlTaskWorker %d 哔哩源下载任务执行失败\n", id)
			BDlTaskList[i].Status = "执行失败"
			BDlTaskList[i].StatusMsg = err.Error()
			return
		}
		BDlTaskList[i].Status = "已完成"
		BDlTaskList[i].StatusMsg = "已完成"
		BDlTaskList[i].ProgressNum = 100.0
	}
}

func BDlTaskWorkerInit() {
	BDlTaskQueue = make(chan *BDlTaskListData)
	BDlTaskList = make([]*BDlTaskListData, 0)
	// 启动多个 BDlTaskWorker 协程来处理任务
	for i := 0; i < DefaultTaskWorkerGoRoutines; i++ {
		go BDlTaskWorker(i)
	}
}
func BDl(AVOrBVStr string, UUID string) error {
	var aid int64
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
	SuitableFileName := ReplaceInvalidCharacters(info.Title, '-')
	// 检查是否已经存在下载目录
	if _, err := os.Stat(filepath.Join(LumikaDecodePath, SuitableFileName)); err == nil {
		LogPrintln(UUID, BDlStr, "下载目录已存在，跳过创建下载目录")
	} else if os.IsNotExist(err) {
		LogPrintln(UUID, BDlStr, "下载目录不存在，创建下载目录")
		// 创建目录
		err = os.Mkdir(filepath.Join(LumikaDecodePath, SuitableFileName), 0644)
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
			filePath := filepath.Join(LumikaDecodePath, SuitableFileName, videoName)
			LogPrintln(UUID, BDlStr, "视频地址:", durl)
			LogPrintln(UUID, BDlStr, "尝试下载视频...")
			err = Dl(durl, filePath, DefaultBiliDownloadReferer, DefaultUserAgent, DefaultBiliDownloadGoRoutines, "")
			if err != nil {
				LogPrintln(UUID, BDlStr, "下载视频("+videoName+")失败，跳过本分P视频:", err)
				return
			}
			// 为全局 ProgressRate 变量赋值
			for kp, kq := range BDlTaskList {
				if kq.UUID == UUID {
					BDlTaskList[kp].ProgressRate++
					// 计算正确的 progressNum
					BDlTaskList[kp].ProgressNum = float64(BDlTaskList[kp].ProgressRate) / float64(len(info.Pages)) * 100
					break
				}
			}
		}(pi)
	}
	wg.Wait()
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	LogPrintf(UUID, BDlStr+" 总共耗时%f秒\n", allDuration.Seconds())
	LogPrintln(UUID, BDlStr, "视频全部下载完成")
	return nil
}
