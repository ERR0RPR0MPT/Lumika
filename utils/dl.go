package utils

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/google/uuid"
	"io"
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
	dt := &DlTaskListData{
		UUID:      uuidd,
		TimeStamp: time.Now().Format("2006-01-02 15:04:05"),
		TaskInfo: &DlTaskInfo{
			Url:        url,
			FileName:   filePath,
			Referer:    referer,
			UserAgent:  userAgent,
			NumThreads: numThreads,
		},
		FileName:     filepath.Base(filePath),
		ProgressRate: 0,
	}
	DlTaskList = append(DlTaskList, dt)
	DlTaskQueue <- dt
}

func DlTaskWorker(id int) {
	for task := range DlTaskQueue {
		LogPrintf(task.UUID, "DlTaskWorker %d 处理下载任务：%v\n", id, task.TaskInfo.Url)
		i := 0
		for kp, kq := range DlTaskList {
			if kq.UUID == task.UUID {
				i = kp
				break
			}
		}
		DlTaskList[i].Status = "正在执行"
		DlTaskList[i].StatusMsg = "正在执行"
		err := Dl(task.TaskInfo.Url, task.TaskInfo.FileName, task.TaskInfo.Referer, task.TaskInfo.UserAgent, task.TaskInfo.NumThreads, task.UUID)
		if err != nil {
			LogPrintf(task.UUID, "DlTaskWorker %d 处理下载任务(%v)失败：%v\n", id, task.TaskInfo.Url, err)
			DlTaskList[i].Status = "执行失败"
			DlTaskList[i].StatusMsg = err.Error()
			continue
		}
		DlTaskList[i].Status = "已完成"
		DlTaskList[i].StatusMsg = "已完成"
		DlTaskList[i].ProgressNum = 100.0
	}
}

func DlTaskWorkerInit() {
	DlTaskQueue = make(chan *DlTaskListData)
	DlTaskList = make([]*DlTaskListData, 0)
	// 启动多个 DlTaskWorker 协程来处理任务
	for i := 0; i < DefaultTaskWorkerGoRoutines; i++ {
		go DlTaskWorker(i)
	}
}

func Dl(url string, filePath string, referer string, userAgent string, numThreads int, UUID string) error {
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &CommonError{Msg: err.Error()}
	}

	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return &CommonError{Msg: err.Error()}
	}
	defer resp.Body.Close()

	totalSize := resp.ContentLength
	threadSize := totalSize / int64(numThreads)

	// 创建临时文件和线程信息
	threads := make([]ThreadInfo, numThreads)
	for i := 0; i < numThreads; i++ {
		tempFilePath := fmt.Sprintf("%s.lpart%d", filePath, i)

		startOffset := int64(i) * threadSize
		endOffset := startOffset + threadSize - 1

		// 最后一个线程负责处理剩余的字节
		if i == numThreads-1 {
			endOffset = totalSize - 1
		}

		threads[i] = ThreadInfo{
			threadIndex:  i,
			startOffset:  startOffset,
			endOffset:    endOffset,
			tempFilePath: tempFilePath,
		}
	}

	// 启动多个线程进行下载
	wg := sync.WaitGroup{}

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(threadIndex int) {
			defer wg.Done()

			// 重试机制
			for di := 0; di < DefaultBiliDownloadMaxRetries; di++ {
				thread := threads[threadIndex]

				tempFile, err := os.Create(thread.tempFilePath)
				if err != nil {
					LogPrint(UUID, "临时文件创建失败")
					// 尝试删除临时文件
					err := os.Remove(thread.tempFilePath)
					if err != nil {
						LogPrint(UUID, "尝试删除临时文件时出现错误，删除失败")
						tempFile.Close()
						return
					}
					continue
				}

				// 创建一个新的HTTP请求
				req2, err := http.NewRequest("GET", url, nil)
				if err != nil {
					LogPrint(UUID, "创建新的HTTP请求时出现错误:", err)
					continue
				}

				rangeHeader := fmt.Sprintf("bytes=%d-%d", thread.startOffset, thread.endOffset)
				req2.Header.Set("Range", rangeHeader)
				req2.Header.Set("Referer", referer)
				req2.Header.Set("User-Agent", userAgent)

				resp2, err := client.Do(req2)
				if err != nil {
					LogPrint(UUID, "下载中出现错误:", err)
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
					LogPrint(UUID, "下载中出现 io.Copy 错误:", err)
					tempFile.Close()
					progressBar.Finish()
					continue
				}

				tempFile.Close()
				resp2.Body.Close()
				progressBar.Finish()
				break
			}

			// 为全局 ProgressRate 变量赋值
			for kp, kq := range DlTaskList {
				if kq.UUID == UUID {
					DlTaskList[kp].ProgressRate++
					// 计算正确的 progressNum
					DlTaskList[kp].ProgressNum = float64(DlTaskList[kp].ProgressRate) / float64(numThreads) * 100
					break
				}
			}

		}(i)
	}

	// 等待所有线程下载完成
	wg.Wait()

	file, err := os.Create(filePath)
	if err != nil {
		return &CommonError{Msg: err.Error()}
	}
	defer file.Close()

	for _, thread := range threads {
		// 读取临时文件
		tempFile, err := os.Open(thread.tempFilePath)
		if err != nil {
			LogPrint(UUID, "尝试打开临时文件时出现错误：", err)
			return &CommonError{Msg: err.Error()}
		}

		_, err = tempFile.Seek(0, 0)
		if err != nil {
			LogPrint(UUID, "尝试将临时文件指针移动到文件开头时出现错误：", err)
			return &CommonError{Msg: err.Error()}
		}

		_, err = io.Copy(file, tempFile)
		if err != nil {
			LogPrint(UUID, "从临时文件复制数据到目标文件时出现错误：", err)
			return &CommonError{Msg: err.Error()}
		}

		for di := 0; di < DefaultBiliDownloadMaxRetries; di++ {
			tempFile.Close()
			err = os.Remove(tempFile.Name())
			if err != nil {
				if di >= 10 {
					LogPrint(UUID, "下载完成后尝试删除临时文件时出现错误，准备重试：", err)
				}
				time.Sleep(250 * time.Millisecond)
				continue
			}
			break
		}
	}
	return nil
}
