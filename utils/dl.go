package utils

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io"
	"net/http"
	"os"
	"sync"
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
	threadIndex int
	startOffset int64
	endOffset   int64
	tempFile    *os.File
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
		tempFile, err := os.Create(fmt.Sprintf("%s.part%d", filePath, i))
		if err != nil {
			return err
		}

		startOffset := int64(i) * threadSize
		endOffset := startOffset + threadSize - 1

		// 最后一个线程负责处理剩余的字节
		if i == numThreads-1 {
			endOffset = totalSize - 1
		}

		threads[i] = threadInfo{
			threadIndex: i,
			startOffset: startOffset,
			endOffset:   endOffset,
			tempFile:    tempFile,
		}
	}

	// 启动多个线程进行下载
	wg := sync.WaitGroup{}

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(threadIndex int) {
			defer wg.Done()

			thread := threads[threadIndex]

			// 创建一个新的HTTP请求
			req2, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Printf("Error creating request for thread %d: %s\n", threadIndex, err)
				return
			}

			rangeHeader := fmt.Sprintf("bytes=%d-%d", thread.startOffset, thread.endOffset)
			req2.Header.Set("Range", rangeHeader)
			req2.Header.Set("Referer", referer)
			req2.Header.Set("User-Agent", userAgent)

			resp2, err := client.Do(req2)
			if err != nil {
				fmt.Printf("Error downloading thread %d: %s\n", threadIndex, err)
				return
			}
			defer resp2.Body.Close()

			progressBar := pb.Full.Start64(resp2.ContentLength)
			progressBar.Set(pb.Bytes, true)
			progressBar.SetWriter(os.Stdout)

			writer := &progressBarWriter{
				writer: thread.tempFile,
				bar:    progressBar,
			}

			_, err = io.Copy(writer, resp2.Body)
			if err != nil {
				fmt.Printf("Error writing data for thread %d: %s\n", threadIndex, err)
				return
			}

			progressBar.Finish()
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
		_, err = thread.tempFile.Seek(0, 0)
		if err != nil {
			fmt.Printf("Error seeking temp file for part %d: %s\n", thread.threadIndex, err)
			return nil
		}

		_, err = io.Copy(file, thread.tempFile)
		if err != nil {
			fmt.Printf("Error writing temp file to final file for part %d: %s\n", thread.threadIndex, err)
			return nil
		}

		thread.tempFile.Close()
		err = os.Remove(thread.tempFile.Name())
		if err != nil {
			fmt.Printf("Error removing temp file for part %d: %s\n", thread.threadIndex, err)
			return nil
		}
	}
	return nil
}
