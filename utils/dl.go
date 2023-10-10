package utils

import (
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/cheggaaa/pb/v3"
	"github.com/google/uuid"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type progressBarWriter struct {
	writer io.Writer
	bar    *pb.ProgressBar
}

func (pw *progressBarWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if err != nil {
		return n, err
	}
	pw.bar.Add(n)
	return n, nil
}

func DlAddTask(url string, filePath string, referer string, userAgent string, numThreads int) {
	uuidd := uuid.New().String()
	dt := &common.DlTaskListData{
		UUID:      uuidd,
		TimeStamp: time.Now().Format("2006-01-02 15:04:05"),
		TaskInfo: &common.DlTaskInfo{
			Url:        url,
			FileName:   filePath,
			Referer:    referer,
			UserAgent:  userAgent,
			NumThreads: numThreads,
		},
		FileName:     filepath.Base(filePath),
		ProgressRate: 0,
		Duration:     "",
	}
	common.DlTaskList[uuidd] = dt
	common.DlTaskQueue <- dt
}

func DlTaskWorker(id int) {
	for task := range common.DlTaskQueue {
		allStartTime := time.Now()
		common.LogPrintf(task.UUID, "DlTaskWorker %d 处理下载任务：%v\n", id, task.TaskInfo.Url)
		_, exist := common.DlTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "DlTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		common.DlTaskList[task.UUID].Status = "正在执行"
		common.DlTaskList[task.UUID].StatusMsg = "正在执行"
		err := Dl(task.TaskInfo.Url, task.TaskInfo.FileName, task.TaskInfo.Referer, task.TaskInfo.UserAgent, task.TaskInfo.NumThreads, task.UUID)
		_, exist = common.DlTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "DlTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		if err != nil {
			common.LogPrintf(task.UUID, "DlTaskWorker %d 处理下载任务(%v)失败：%v\n", id, task.TaskInfo.Url, err)
			common.DlTaskList[task.UUID].Status = "执行失败"
			common.DlTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		common.DlTaskList[task.UUID].Status = "已完成"
		common.DlTaskList[task.UUID].StatusMsg = "已完成"
		common.DlTaskList[task.UUID].ProgressNum = 100.0
		common.DlTaskList[task.UUID].Duration = fmt.Sprintf("%vs", int64(math.Floor(time.Now().Sub(allStartTime).Seconds())))
	}
}

func DlTaskWorkerInit() {
	common.DlTaskQueue = make(chan *common.DlTaskListData)
	common.DlTaskList = make(map[string]*common.DlTaskListData)
	if len(common.DatabaseVariable.DlTaskList) != 0 {
		common.DlTaskList = common.DatabaseVariable.DlTaskList
		for kp, kq := range common.DlTaskList {
			if kq.Status == "正在执行" {
				common.DlTaskList[kp].Status = "执行失败"
				common.DlTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				common.DlTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 DlTaskWorker 协程来处理任务
	for i := 0; i < common.VarSettingsVariable.DefaultTaskWorkerGoRoutines; i++ {
		go DlTaskWorker(i)
	}
}

func Dl(url string, filePath string, referer string, userAgent string, numThreads int, UUID string) error {
	if userAgent == "" {
		userAgent = browser.Random()
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &common.CommonError{Msg: err.Error()}
	}

	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)

	var totalSize int64
	var threadSize int64
	isOK := false
	for di := 0; di < common.DefaultDlMaxRetries; di++ {
		resp, err := client.Do(req)
		if err != nil {
			return &common.CommonError{Msg: err.Error()}
		}

		statusCode := resp.StatusCode
		if (statusCode/100) != 2 && (statusCode/100) != 3 {
			common.LogPrintln(UUID, common.DlStr, "请求异常，出现非正常返回码:", statusCode, "，等待 1 秒后重试")
			time.Sleep(time.Second)
			continue
		}

		totalSize = resp.ContentLength
		threadSize = totalSize / int64(numThreads)
		resp.Body.Close()
		isOK = true
		break
	}
	if !isOK {
		return &common.CommonError{Msg: "请求异常，出现非正常返回码"}
	}

	// 创建临时文件和线程信息
	threads := make([]common.ThreadInfo, numThreads)
	for i := 0; i < numThreads; i++ {
		tempFilePath := fmt.Sprintf("%s.lpart%d", filePath, i)

		startOffset := int64(i) * threadSize
		endOffset := startOffset + threadSize - 1

		// 最后一个线程负责处理剩余的字节
		if i == numThreads-1 {
			endOffset = totalSize - 1
		}

		threads[i] = common.ThreadInfo{
			ThreadIndex:  i,
			StartOffset:  startOffset,
			EndOffset:    endOffset,
			TempFilePath: tempFilePath,
		}
	}

	// 启动多个线程进行下载
	wg := sync.WaitGroup{}

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		proc := func(threadIndex int) error {
			defer wg.Done()

			// 重试机制
			isOK = false
			for di := 0; di < common.DefaultDlMaxRetries; di++ {
				thread := threads[threadIndex]

				tempFile, err := os.Create(thread.TempFilePath)
				if err != nil {
					common.LogPrintln(UUID, common.DlStr, "临时文件创建失败")
					// 尝试删除临时文件
					err := os.Remove(thread.TempFilePath)
					if err != nil {
						common.LogPrintln(UUID, common.DlStr, "尝试删除临时文件时出现错误，删除失败")
						tempFile.Close()
						return &common.CommonError{Msg: "尝试删除临时文件时出现错误，删除失败"}
					}
					continue
				}

				// 创建一个新的HTTP请求
				req2, err := http.NewRequest("GET", url, nil)
				if err != nil {
					common.LogPrintln(UUID, common.DlStr, "创建新的HTTP请求时出现错误:", err)
					continue
				}

				rangeHeader := fmt.Sprintf("bytes=%d-%d", thread.StartOffset, thread.EndOffset)
				req2.Header.Set("Range", rangeHeader)
				req2.Header.Set("Referer", referer)
				req2.Header.Set("User-Agent", userAgent)

				resp2, err := client.Do(req2)
				if err != nil {
					common.LogPrintln(UUID, common.DlStr, "下载中出现错误:", err)
					continue
				}

				statusCode := resp2.StatusCode
				if (statusCode/100) != 2 && (statusCode/100) != 3 {
					common.LogPrintln(UUID, common.DlStr, "请求异常，出现非正常返回码:", statusCode, "，等待 1 秒后重试")
					time.Sleep(time.Second)
					continue
				}

				progressBar := pb.Full.Start64(resp2.ContentLength)
				progressBar.Set(pb.Bytes, true)
				progressBar.SetWriter(os.Stdout)

				writer := &progressBarWriter{
					writer: tempFile,
					bar:    progressBar,
				}

				_, err = io.Copy(writer, resp2.Body)
				if err != nil {
					common.LogPrintln(UUID, common.DlStr, "下载中出现 io.Copy 错误:", err)
					tempFile.Close()
					progressBar.Finish()
					continue
				}

				tempFile.Close()
				resp2.Body.Close()
				progressBar.Finish()
				isOK = true
				break
			}

			if !isOK {
				return &common.CommonError{Msg: "请求异常，出现非正常返回码"}
			}

			if UUID != "" {
				_, exist := common.DlTaskList[UUID]
				if exist {
					// 为全局 ProgressRate 变量赋值
					common.DlTaskList[UUID].ProgressRate++
					// 计算正确的 progressNum
					common.DlTaskList[UUID].ProgressNum = float64(common.DlTaskList[UUID].ProgressRate) / float64(numThreads) * 100
				} else {
					common.LogPrintln(UUID, common.DlStr, common.ErStr, "当前任务被用户删除", err)
					return &common.CommonError{Msg: "当前任务被用户删除"}
				}
			}
			return nil
		}(i)
		if proc != nil {
			common.LogPrintln(UUID, common.DlStr, "下载中出现错误:", proc)
			return proc
		}
	}

	// 等待所有线程下载完成
	wg.Wait()

	if UUID != "" {
		_, exist := common.DlTaskList[UUID]
		if !exist {
			common.LogPrintln(UUID, common.DlStr, common.ErStr, "当前任务被用户删除", err)
			return &common.CommonError{Msg: "当前任务被用户删除"}
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		common.LogPrintln(UUID, common.DlStr, "文件创建错误:", err)
		return &common.CommonError{Msg: "文件创建错误:" + err.Error()}
	}
	defer file.Close()

	for _, thread := range threads {
		// 读取临时文件
		tempFile, err := os.Open(thread.TempFilePath)
		if err != nil {
			common.LogPrintln(UUID, common.DlStr, "尝试打开临时文件时出现错误：", err)
			return &common.CommonError{Msg: "尝试打开临时文件时出现错误:" + err.Error()}
		}

		_, err = tempFile.Seek(0, 0)
		if err != nil {
			common.LogPrintln(UUID, common.DlStr, "尝试将临时文件指针移动到文件开头时出现错误：", err)
			return &common.CommonError{Msg: "尝试将临时文件指针移动到文件开头时出现错误:" + err.Error()}
		}

		_, err = io.Copy(file, tempFile)
		if err != nil {
			common.LogPrintln(UUID, common.DlStr, "从临时文件复制数据到目标文件时出现错误：", err)
			return &common.CommonError{Msg: "从临时文件复制数据到目标文件时出现错误:" + err.Error()}
		}

		for di := 0; di < 100; di++ {
			tempFile.Close()
			err = os.Remove(tempFile.Name())
			if err != nil {
				if di >= 10 {
					common.LogPrintln(UUID, common.DlStr, "下载完成后尝试删除临时文件时出现错误，准备重试：", err)
				}
				time.Sleep(250 * time.Millisecond)
				continue
			}
			break
		}
	}
	return nil
}
