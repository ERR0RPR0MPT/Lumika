package utils

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io"
	"net/http"
	"os"
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

type threadInfo struct {
	threadIndex  int
	startOffset  int64
	endOffset    int64
	tempFilePath string
}

func Dl(url string, filePath string, referer string, userAgent string, numThreads int) error {
	if referer == "" {
		referer = DefaultBiliDownloadReferer
	}
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	totalSize := resp.ContentLength
	threadSize := totalSize / int64(numThreads)

	// 创建临时文件和线程信息
	threads := make([]threadInfo, numThreads)
	for i := 0; i < numThreads; i++ {
		tempFilePath := fmt.Sprintf("%s.lpart%d", filePath, i)

		startOffset := int64(i) * threadSize
		endOffset := startOffset + threadSize - 1

		// 最后一个线程负责处理剩余的字节
		if i == numThreads-1 {
			endOffset = totalSize - 1
		}

		threads[i] = threadInfo{
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
					fmt.Println("临时文件创建失败")
					// 尝试删除临时文件
					err := os.Remove(thread.tempFilePath)
					if err != nil {
						fmt.Println("尝试删除临时文件时出现错误，删除失败")
						tempFile.Close()
						return
					}
					continue
				}

				// 创建一个新的HTTP请求
				req2, err := http.NewRequest("GET", url, nil)
				if err != nil {
					fmt.Println("创建新的HTTP请求时出现错误:", err)
					continue
				}

				rangeHeader := fmt.Sprintf("bytes=%d-%d", thread.startOffset, thread.endOffset)
				req2.Header.Set("Range", rangeHeader)
				req2.Header.Set("Referer", referer)
				req2.Header.Set("User-Agent", userAgent)

				resp2, err := client.Do(req2)
				if err != nil {
					fmt.Println("下载中出现错误:", err)
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
					fmt.Println("下载中出现 io.Copy 错误:", err)
					tempFile.Close()
					progressBar.Finish()
					continue
				}

				tempFile.Close()
				resp2.Body.Close()
				progressBar.Finish()
				break
			}
		}(i)
	}

	// 等待所有线程下载完成
	wg.Wait()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, thread := range threads {
		// 读取临时文件
		tempFile, err := os.Open(thread.tempFilePath)
		if err != nil {
			fmt.Println("尝试打开临时文件时出现错误：", err)
			return nil
		}

		_, err = tempFile.Seek(0, 0)
		if err != nil {
			fmt.Println("尝试将临时文件指针移动到文件开头时出现错误：", err)
			return nil
		}

		_, err = io.Copy(file, tempFile)
		if err != nil {
			fmt.Println("从临时文件复制数据到目标文件时出现错误：", err)
			return nil
		}

		for di := 0; di < DefaultBiliDownloadMaxRetries; di++ {
			tempFile.Close()
			err = os.Remove(tempFile.Name())
			if err != nil {
				if di >= 10 {
					fmt.Println("下载完成后尝试删除临时文件时出现错误，准备重试：", err)
				}
				time.Sleep(250 * time.Millisecond)
				continue
			}
			break
		}
	}
	return nil
}
