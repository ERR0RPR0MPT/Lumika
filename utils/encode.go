package utils

import (
	"bytes"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/klauspost/reedsolomon"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Encode(fileDir string, videoSize int, outputFPS int, maxSeconds int, MGValue int, KGValue int, encodeThread int, encodeFFmpegMode string, auto bool, UUID string) (segmentLength int64, err error) {
	ep, err := os.Executable()
	if err != nil {
		LogPrintln(UUID, EnStr, ErStr, "无法获取运行目录:", err)
		return 0, &CommonError{Msg: "无法获取运行目录"}
	}
	epPath := filepath.Dir(ep)

	if videoSize%8 != 0 {
		LogPrintln(UUID, EnStr, ErStr, "视频大小必须是8的倍数")
		return 0, &CommonError{Msg: "视频大小必须是8的倍数"}
	}

	if KGValue > MGValue {
		LogPrintln(UUID, EnStr, ErStr, "KG值不能大于MG值")
		return 0, &CommonError{Msg: "KG值不能大于MG值"}
	}

	// 当没有检测到videoFileDir时，自动匹配
	if fileDir == "" {
		LogPrintln(UUID, EnStr, "自动使用程序所在目录作为输入目录")
		fd, err := os.Executable()
		if err != nil {
			LogPrintln(UUID, EnStr, ErStr, "获取程序所在目录失败:", err)
			return 0, &CommonError{Msg: "获取程序所在目录失败"}
		}
		fileDir = filepath.Dir(fd)
	}

	// 检查输入文件夹是否存在
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		LogPrintln(UUID, EnStr, ErStr, "输入文件夹不存在:", err)
		return 0, &CommonError{Msg: "输入文件夹不存在"}
	}

	LogPrintln(UUID, EnStr, "当前目录:", fileDir)

	fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
	if err != nil {
		LogPrintln(UUID, EnStr, ErStr, "无法生成文件列表:", err)
		return 0, &CommonError{Msg: "无法生成文件列表"}
	}
	filePathList := make([]string, 0)
	for {
		if len(fileDict) == 0 {
			LogPrintln(UUID, EnStr, ErStr, "当前目录下没有.fec文件，请将需要编码的文件放到当前目录下")
			return 0, &CommonError{Msg: "当前目录下没有.fec文件，请将需要编码的文件放到当前目录下"}
		}
		LogPrintln(UUID, EnStr, "请选择需要编码的.fec文件，输入索引并回车来选择")
		LogPrintln(UUID, EnStr, "如果需要编码当前目录下的所有.fec文件，请直接输入回车")
		for index := 0; index < len(fileDict); index++ {
			LogPrintln(UUID, EnStr, strconv.Itoa(index)+":", fileDict[index])
		}
		var result string
		if auto {
			result = ""
		} else {
			result = GetUserInput("")
		}
		if result == "" {
			LogPrintln(UUID, EnStr, "注意：开始编码当前目录下的所有.fec文件")
			for _, filePath := range fileDict {
				filePathList = append(filePathList, filePath)
			}
			break
		} else {
			index, err := strconv.Atoi(result)
			if err != nil {
				LogPrintln(UUID, EnStr, ErStr, "输入索引不是数字，请重新输入")
				continue
			}
			if index < 0 || index >= len(fileDict) {
				LogPrintln(UUID, EnStr, ErStr, "输入索引超出范围，请重新输入")
				continue
			}
			filePathList = append(filePathList, fileDict[index])
			break
		}
	}

	isPaused := false
	isRuntime := true
	if UUID == "" {
		go func() {
			LogPrintln(UUID, EnStr, "按下回车键暂停/继续运行")
			for {
				GetUserInput("")
				if !isRuntime {
					return
				}
				isPaused = !isPaused
				LogPrintln(UUID, EnStr, "当前是否正在运行：", !isPaused)
			}
		}()
	}

	// 启动多个goroutine
	var wg sync.WaitGroup
	maxGoroutines := encodeThread // 最大同时运行的协程数量
	LogPrintln(UUID, EnStr, "最大同时运行的协程数量:", maxGoroutines)
	semaphore := make(chan struct{}, maxGoroutines)
	allStartTime := time.Now()

	// 定义错误通道和计数器
	errorChan := make(chan error)
	errorCount := 0
	var errorError error = nil

	go func() {
		for errorCount < len(filePathList) {
			err2 := <-errorChan
			if err2 != nil {
				errorError = err2
				return
			}
			errorCount++
		}
		close(errorChan)
	}()

	// 遍历需要处理的文件列表
	for fileIndexNum, filePath := range filePathList {
		if errorError != nil {
			return 0, errorError
		}
		LogPrintln(UUID, EnStr, "开始编码第", fileIndexNum+1, "个文件，路径:", filePath)
		wg.Add(1)               // 增加计数器
		semaphore <- struct{}{} // 协程获取信号量，若已满则阻塞
		go func(fileIndexNum int, filePath string) {
			defer func() {
				<-semaphore // 协程释放信号量
				wg.Done()
			}()

			// 读取文件
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				LogPrintln(UUID, EnStr, ErStr, "无法打开文件:", err)
				errorChan <- &CommonError{Msg: "无法打开文件:" + err.Error()}
				return
			}

			outputFilePath := AddOutputToFileName(filePath, ".mp4")                    // 输出文件路径
			fileLength := GetFileSize(filePath)                                        // 输入文件长度
			dataSliceLen := videoSize * videoSize / 8                                  // 每帧存储的有效数据
			allFrameNum := int(math.Ceil(float64(fileLength) / float64(dataSliceLen))) // 生成总帧数
			allSeconds := int(math.Ceil(float64(allFrameNum) / float64(outputFPS)))    // 总时长(秒)

			// 检查时长是否超过限制
			if allSeconds > maxSeconds {
				LogPrintln(UUID, EnStr, ErStr, "警告：生成的段视频时长超过限制("+strconv.Itoa(allSeconds)+"s>"+strconv.Itoa(maxSeconds)+"s)")
				LogPrintln(UUID, EnStr, ErStr, "请调整M值、K值、输出帧率、最大生成时长来满足要求")
				GetUserInput("请按回车键继续...")
				os.Exit(0)
			}

			segmentLength = fileLength

			LogPrintln(UUID, EnStr, "开始运行")
			LogPrintln(UUID, EnStr, "使用配置：")
			LogPrintln(UUID, EnStr, "  ---------------------------")
			LogPrintln(UUID, EnStr, "  输入文件:", filePath)
			LogPrintln(UUID, EnStr, "  输出文件:", outputFilePath)
			LogPrintln(UUID, EnStr, "  输入文件长度:", fileLength)
			LogPrintln(UUID, EnStr, "  每帧数据长度:", dataSliceLen)
			LogPrintln(UUID, EnStr, "  每帧索引数据长度:", 4)
			LogPrintln(UUID, EnStr, "  每帧真实数据长度:", dataSliceLen-4)
			LogPrintln(UUID, EnStr, "  帧大小:", videoSize)
			LogPrintln(UUID, EnStr, "  输出帧率:", outputFPS)
			LogPrintln(UUID, EnStr, "  生成总帧数:", allFrameNum)
			LogPrintln(UUID, EnStr, "  总时长: ", strconv.Itoa(allSeconds)+"s")
			LogPrintln(UUID, EnStr, "  FFmpeg 预设:", encodeFFmpegMode)
			LogPrintln(UUID, EnStr, "  ---------------------------")

			// 检查是否有 FFmpeg 在程序目录下
			FFmpegPath := SearchFileNameInDir(epPath, "ffmpeg")
			if FFmpegPath == "" || FFmpegPath != "" && !strings.Contains(filepath.Base(FFmpegPath), "ffmpeg") {
				LogPrintln(UUID, EnStr, "使用系统环境变量中的 FFmpeg")
				FFmpegPath = "ffmpeg"
			} else {
				LogPrintln(UUID, EnStr, "使用找到 FFmpeg 程序:", FFmpegPath)
			}

			FFmpegCmd := []string{
				"-y",
				"-f", "image2pipe",
				"-vcodec", "png",
				"-r", fmt.Sprintf("%d", outputFPS),
				"-i", "-",
				"-c:v", "libx264",
				"-preset", encodeFFmpegMode,
				"-crf", "18",
				"-s", strconv.Itoa(videoSize) + "x" + strconv.Itoa(videoSize),
				outputFilePath,
			}

			FFmpegProcess := exec.Command(FFmpegPath, FFmpegCmd...)
			stdin, err := FFmpegProcess.StdinPipe()
			if err != nil {
				LogPrintln(UUID, EnStr, ErStr, "无法创建 FFmpeg 的标准输入管道:", err)
				errorChan <- &CommonError{Msg: "无法创建 FFmpeg 的标准输入管道:" + err.Error()}
				return
			}
			err = FFmpegProcess.Start()
			if err != nil {
				LogPrintln(UUID, EnStr, ErStr, "无法启动 FFmpeg 子进程:", err)
				errorChan <- &CommonError{Msg: "无法启动 FFmpeg 子进程:" + err.Error()}
				return
			}

			// 生成空白帧
			blankData := make([]byte, dataSliceLen)
			for j := 0; j < dataSliceLen; j++ {
				blankData[j] = DefaultBlankByte
			}
			imgBlank := Data2Image(blankData, videoSize)
			allBlankFrameNum := 0

			// 生成起始帧
			blankStartData := make([]byte, dataSliceLen)
			for j := 0; j < dataSliceLen; j++ {
				blankStartData[j] = DefaultBlankStartByte
			}
			imgBlankStart := Data2Image(blankStartData, videoSize)

			// 生成终止帧
			blankEndData := make([]byte, dataSliceLen)
			for j := 0; j < dataSliceLen; j++ {
				blankEndData[j] = DefaultBlankEndByte
			}
			imgBlankEnd := Data2Image(blankEndData, videoSize)

			i := 0
			// 启动进度条
			bar := pb.StartNew(int(fileLength))

			// 创建一个shards
			shardsInsideNum := 0
			shards := make([][]byte, MGValue)
			for ig := range shards {
				shards[ig] = make([]byte, dataSliceLen-4)
			}
			enc, err := reedsolomon.New(KGValue, MGValue-KGValue)
			if err != nil {
				LogPrintln(UUID, EnStr, ErStr, "无法创建RS编码器:", err)
				errorChan <- &CommonError{Msg: "无法创建RS编码器:" + err.Error()}
				return
			}

			// 创建图像缓冲区(公用)
			imageBuffer := new(bytes.Buffer)

			// 为规避某些编码器会自动在视频的前后删除某些帧，导致解码失败，这里在视频的前后各添加defaultBlankSeconds秒的空白帧
			// 由于视频的前后各添加了defaultBlankSeconds秒的空白帧，所以总时长需要加上4秒
			for k := 0; k < outputFPS*DefaultBlankSeconds; k++ {
				// 生成带空白数据的图像
				err = png.Encode(imageBuffer, imgBlank)
				if err != nil {
					errorChan <- &CommonError{Msg: "无法生成带空白数据的图像:" + err.Error()}
					return
				}
				_, err = stdin.Write(imageBuffer.Bytes())
				if err != nil {
					LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
					errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
					return
				}
				imageBuffer.Reset()
				allBlankFrameNum++
				i++
			}

			fileNowLength := 0
			for {
				// 检测是否暂停
				if UUID != "" {
					_, exist := AddTaskList[UUID]
					if exist {
						if AddTaskList[UUID].IsPaused {
							time.Sleep(time.Second)
							continue
						}
					} else {
						LogPrintln(UUID, EnStr, ErStr, "当前任务被用户删除", err)
						errorChan <- &CommonError{Msg: "当前任务被用户删除"}
						return
					}
				} else {
					if isPaused {
						time.Sleep(time.Second)
						continue
					}
				}
				// 从文件读取数据
				if len(fileData) == 0 {
					if shardsInsideNum != 0 {
						// 生成空数据帧
						blankData2 := make([]byte, dataSliceLen-4)
						for j := 0; j < dataSliceLen-4; j++ {
							blankData2[j] = 0
						}
						dataPackageBlankData2 := make([]byte, dataSliceLen-4)
						copy(dataPackageBlankData2, blankData2)
						for l := shardsInsideNum; l < KGValue; l++ {
							shards[shardsInsideNum] = dataPackageBlankData2
						}
						shardsInsideNum = 0
						// 创建冗余数据
						err = enc.Encode(shards)
						if err != nil {
							LogPrintln(UUID, EnStr, ErStr, "无法创建冗余数据:", err)
							errorChan <- &CommonError{Msg: "无法创建冗余数据:" + err.Error()}
							return
						}
						// 创建完整数据
						allShards := make([][]byte, MGValue)
						for jk := range shards {
							// 给数据写入索引信息，同时写入数据
							allShards[jk] = append(IntToByteArray(uint32(jk)), shards[jk]...)
						}
						// 输入开始帧
						err = png.Encode(imageBuffer, imgBlankStart)
						if err != nil {
							errorChan <- &CommonError{Msg: "无法生成带空白数据的图像:" + err.Error()}
							return
						}
						_, err = stdin.Write(imageBuffer.Bytes())
						if err != nil {
							LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
							errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
							return
						}
						imageBuffer.Reset()
						// 遍历 allShards
						for _, shardData := range allShards {
							// 生成带数据的图像
							img := Data2Image(shardData, videoSize)
							err := png.Encode(imageBuffer, img)
							if err != nil {
								errorChan <- &CommonError{Msg: "无法生成带数据的图像:" + err.Error()}
								return
							}
							_, err = stdin.Write(imageBuffer.Bytes())
							if err != nil {
								LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
								errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
								return
							}
							imageBuffer.Reset()
						}
						// 输入终止帧
						err = png.Encode(imageBuffer, imgBlankEnd)
						if err != nil {
							errorChan <- &CommonError{Msg: "无法生成带空白数据的图像:" + err.Error()}
							return
						}
						_, err = stdin.Write(imageBuffer.Bytes())
						if err != nil {
							LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
							errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
							return
						}
						imageBuffer.Reset()
					}
					break
				}
				data := make([]byte, dataSliceLen-4)
				if len(fileData) >= dataSliceLen-4 {
					data = fileData[:dataSliceLen-4]
					fileData = fileData[dataSliceLen-4:]
				} else {
					data = fileData
					fileData = nil
				}

				dataPackage := make([]byte, dataSliceLen-4)
				copy(dataPackage, data)
				shards[shardsInsideNum] = dataPackage

				// 判断shards是否被填满
				if shardsInsideNum == KGValue-1 {
					shardsInsideNum = 0
					// 创建冗余数据
					err = enc.Encode(shards)
					if err != nil {
						LogPrintln(UUID, EnStr, ErStr, "无法创建冗余数据:", err)
						errorChan <- &CommonError{Msg: "无法创建冗余数据:" + err.Error()}
						return
					}
					// 创建完整数据
					allShards := make([][]byte, MGValue)
					for jk := range shards {
						// 给数据写入索引信息，同时写入数据
						allShards[jk] = append(IntToByteArray(uint32(jk)), shards[jk]...)
					}
					// 输入开始帧
					err = png.Encode(imageBuffer, imgBlankStart)
					if err != nil {
						errorChan <- &CommonError{Msg: "无法生成带空白数据的图像:" + err.Error()}
						return
					}
					_, err = stdin.Write(imageBuffer.Bytes())
					if err != nil {
						LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
						errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
						return
					}
					imageBuffer.Reset()
					// 遍历 allShards
					for _, shardData := range allShards {
						// 生成带数据的图像
						img := Data2Image(shardData, videoSize)
						err := png.Encode(imageBuffer, img)
						if err != nil {
							errorChan <- &CommonError{Msg: "无法生成带数据的图像:" + err.Error()}
							return
						}
						_, err = stdin.Write(imageBuffer.Bytes())
						if err != nil {
							LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
							errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
							return
						}
						imageBuffer.Reset()
					}
					// 输入终止帧
					err = png.Encode(imageBuffer, imgBlankEnd)
					if err != nil {
						errorChan <- &CommonError{Msg: "无法生成带空白数据的图像:" + err.Error()}
						return
					}
					_, err = stdin.Write(imageBuffer.Bytes())
					if err != nil {
						LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
						errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
						return
					}
					imageBuffer.Reset()
				} else {
					shardsInsideNum++
				}

				i++
				fileNowLength += len(data)

				bar.SetCurrent(int64(fileNowLength))
				if i%30000 == 0 {
					LogPrintf(UUID, "\nEncode: 构建帧 %d, 已构建数据 %d, 总数据 %d\n", i, fileNowLength, fileLength)
				}
			}
			bar.Finish()

			// 为规避某些编码器会自动在视频的前后删除某些帧，导致解码失败，这里在视频的前后各添加defaultBlankSeconds秒的空白帧
			// 或者直接生成后一半的空白视频来阻止编码器删除数据帧
			for k := 0; k < outputFPS*DefaultBlankSeconds; k++ {
				err := png.Encode(imageBuffer, imgBlank)
				if err != nil {
					errorChan <- &CommonError{Msg: "无法生成带空白数据的图像:" + err.Error()}
					return
				}
				_, err = stdin.Write(imageBuffer.Bytes())
				if err != nil {
					LogPrintln(UUID, EnStr, ErStr, "无法写入帧数据到 FFmpeg:", err)
					errorChan <- &CommonError{Msg: "无法写入帧数据到 FFmpeg:" + err.Error()}
					return
				}
				imageBuffer.Reset()
				allBlankFrameNum++
			}
			LogPrintln(UUID, EnStr, "添加完成，总共添加", allBlankFrameNum, "帧空白帧")

			// 关闭 FFmpeg 的标准输入管道，等待子进程完成
			err = stdin.Close()
			if err != nil {
				LogPrintln(UUID, EnStr, ErStr, "无法关闭 FFmpeg 的标准输入管道:", err)
				errorChan <- &CommonError{Msg: "无法关闭 FFmpeg 的标准输入管道:" + err.Error()}
				return
			}
			if err := FFmpegProcess.Wait(); err != nil {
				LogPrintln(UUID, EnStr, ErStr, "FFmpeg 子进程执行失败:", err)
				errorChan <- &CommonError{Msg: "FFmpeg 子进程执行失败:" + err.Error()}
				return
			}

			if UUID != "" {
				_, exist := AddTaskList[UUID]
				if exist {
					// 为全局 ProgressRate 变量赋值
					AddTaskList[UUID].ProgressRate++
					// 计算正确的 progressNum
					AddTaskList[UUID].ProgressNum = float64(AddTaskList[UUID].ProgressRate) / float64(len(filePathList)) * 100
				} else {
					LogPrintln(UUID, EnStr, ErStr, "当前任务被用户删除", err)
					errorChan <- &CommonError{Msg: "当前任务被用户删除"}
					return
				}
			}

			LogPrintln(UUID, EnStr, "完成")
			LogPrintln(UUID, EnStr, "使用配置：")
			LogPrintln(UUID, EnStr, "  ---------------------------")
			LogPrintln(UUID, EnStr, "  输入文件:", filePath)
			LogPrintln(UUID, EnStr, "  输出文件:", outputFilePath)
			LogPrintln(UUID, EnStr, "  输入文件长度:", fileLength)
			LogPrintln(UUID, EnStr, "  每帧数据长度:", dataSliceLen)
			LogPrintln(UUID, EnStr, "  每帧索引数据长度:", 4)
			LogPrintln(UUID, EnStr, "  每帧真实数据长度:", dataSliceLen-4)
			LogPrintln(UUID, EnStr, "  帧大小:", videoSize)
			LogPrintln(UUID, EnStr, "  输出帧率:", outputFPS)
			LogPrintln(UUID, EnStr, "  生成总帧数:", allFrameNum)
			LogPrintln(UUID, EnStr, "  总时长: ", strconv.Itoa(allSeconds)+"s")
			LogPrintln(UUID, EnStr, "  FFmpeg 预设:", encodeFFmpegMode)
			LogPrintln(UUID, EnStr, "  ---------------------------")
			errorChan <- nil
			return
		}(fileIndexNum, filePath)
	}
	wg.Wait()
	isRuntime = false
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	LogPrintf(UUID, EnStr+" 总共耗时%f秒\n", allDuration.Seconds())
	LogPrintln(UUID, EnStr, "所有选择的.fec文件已编码完成，编码结束")
	return segmentLength, nil
}
