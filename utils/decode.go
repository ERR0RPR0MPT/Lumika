package utils

import (
	"github.com/cheggaaa/pb/v3"
	"github.com/klauspost/reedsolomon"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Decode(videoFileDir string, segmentLength int64, filePathList []string, MGValue int, KGValue int, decodeThread int, UUID string) {
	if KGValue > MGValue {
		LogPrintln(UUID, DeStr, ErStr, "KG值不能大于MG值")
		return
	}

	// 当没有检测到videoFileDir时，自动匹配
	if videoFileDir == "" {
		LogPrintln(UUID, DeStr, "自动使用程序所在目录作为输入目录")
		fd, err := os.Executable()
		if err != nil {
			LogPrintln(UUID, DeStr, ErStr, "获取程序所在目录失败:", err)
			return
		}
		videoFileDir = filepath.Dir(fd)
	}

	// 检查输入文件夹是否存在
	if _, err := os.Stat(videoFileDir); os.IsNotExist(err) {
		LogPrintln(UUID, DeStr, ErStr, "输入文件夹不存在:", err)
		return
	}

	// 创建输出目录
	videoFileOutputDir := filepath.Join(LumikaDecodeOutputPath, filepath.Base(videoFileDir))
	if _, err := os.Stat(videoFileOutputDir); os.IsNotExist(err) {
		LogPrintln(UUID, DeStr, "创建输出目录:", videoFileOutputDir)
		err = os.Mkdir(videoFileOutputDir, 0644)
		if err != nil {
			LogPrintln(UUID, DeStr, ErStr, "创建输出目录失败:", err)
			return
		}
	}

	LogPrintln(UUID, DeStr, "当前检测目录:", videoFileDir)
	LogPrintln(UUID, DeStr, "当前输出目录:", videoFileOutputDir)

	fileDict, err := GenerateFileDxDictionary(videoFileDir, ".mp4")
	if err != nil {
		LogPrintln(UUID, DeStr, ErStr, "无法生成视频列表:", err)
		return
	}

	if filePathList == nil {
		filePathList = make([]string, 0)
		for {
			if len(fileDict) == 0 {
				LogPrintln(UUID, DeStr, ErStr, "当前目录下没有.mp4文件，请将需要解码的视频文件放到当前目录下")
				return
			}
			LogPrintln(UUID, DeStr, "请选择需要编码的.mp4文件，输入索引并回车来选择")
			LogPrintln(UUID, DeStr, "如果需要编码当前目录下的所有.mp4文件，请直接输入回车")
			for index := 0; index < len(fileDict); index++ {
				LogPrintln(UUID, "Encode:", strconv.Itoa(index)+":", fileDict[index])
			}
			result := GetUserInput("")
			if result == "" {
				LogPrintln(UUID, DeStr, "注意：开始解码当前目录下的所有.mp4文件")
				for _, filePath := range fileDict {
					filePathList = append(filePathList, filePath)
				}
				break
			} else {
				index, err := strconv.Atoi(result)
				if err != nil {
					LogPrintln(UUID, DeStr, ErStr, "输入索引不是数字，请重新输入")
					continue
				}
				if index < 0 || index >= len(fileDict) {
					LogPrintln(UUID, DeStr, ErStr, "输入索引超出范围，请重新输入")
					continue
				}
				filePathList = append(filePathList, fileDict[index])
				break
			}
		}
	}

	// 错误数据数量
	errorDataNum := 0

	indexU := 0
	isPaused := false
	isRuntime := true
	if UUID != "" {
		for kp, kq := range GetTaskList {
			if kq.UUID == UUID {
				indexU = kp
				break
			}
		}
	} else {
		// 启动监控进程
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

	var wg sync.WaitGroup
	maxGoroutines := decodeThread // 最大同时运行的协程数量
	semaphore := make(chan struct{}, maxGoroutines)

	// 遍历解码所有文件
	allStartTime := time.Now()
	for filePathIndex, filePath := range filePathList {
		wg.Add(1)               // 增加计数器
		semaphore <- struct{}{} // 协程获取信号量，若已满则阻塞
		go func(filePathIndex int, filePath string) {
			defer func() {
				<-semaphore // 协程释放信号量
				wg.Done()
			}()
			LogPrintln(UUID, DeStr, "开始解码第", filePathIndex+1, "个编码文件:", filePath)

			// 检查是否有 FFprobe 在程序目录下
			FFprobePath := SearchFileNameInDir(EpPath, "ffprobe")
			if FFprobePath == "" || FFprobePath != "" && !strings.Contains(filepath.Base(FFprobePath), "ffprobe") {
				LogPrintln(UUID, DeStr, "使用系统环境变量中的 FFprobe")
				FFprobePath = "ffprobe"
			} else {
				LogPrintln(UUID, DeStr, "使用找到 FFprobe 程序:", FFprobePath)
			}

			cmd := exec.Command(FFprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", filePath)
			output, err := cmd.Output()
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "FFprobe 启动失败，请检查文件是否存在:", err)
				return
			}
			result := strings.Split(string(output), ",")
			if len(result) != 2 {
				LogPrintln(UUID, DeStr, ErStr, "无法读取视频宽高，请检查视频文件是否正确")
				return
			}
			videoWidth, err := strconv.Atoi(strings.TrimSpace(result[0]))
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法读取视频宽高，请检查视频文件是否正确:", err)
				return
			}
			videoHeight, err := strconv.Atoi(strings.TrimSpace(result[1]))
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法读取视频宽高，请检查视频文件是否正确:", err)
				return
			}
			cmd = exec.Command(FFprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=nb_frames", "-of", "default=nokey=1:noprint_wrappers=1", filePath)
			output, err = cmd.Output()
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "执行 FFprobe 命令时出错:", err)
				return
			}
			frameCount, err := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(string(output)))
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "解析视频帧数时出错:", err)
				return
			}

			// 设置输出路径
			outputFilePath := filepath.Join(videoFileOutputDir, filepath.Base(filePath)+".fec")

			LogPrintln(UUID, DeStr, "开始解码")
			LogPrintln(UUID, DeStr, "使用配置：")
			LogPrintln(UUID, DeStr, "  ---------------------------")
			LogPrintln(UUID, DeStr, "  视频宽度:", videoWidth)
			LogPrintln(UUID, DeStr, "  视频高度:", videoHeight)
			LogPrintln(UUID, DeStr, "  总帧数:", frameCount)
			LogPrintln(UUID, DeStr, "  输入视频路径:", filePath)
			LogPrintln(UUID, DeStr, "  输出文件路径:", outputFilePath)
			LogPrintln(UUID, DeStr, "  ---------------------------")

			// 打开输出文件
			LogPrintln(UUID, DeStr, "创建输出文件:", outputFilePath)
			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法创建输出文件:", err)
				return
			}

			// 检查是否有 FFmpeg 在程序目录下
			FFmpegPath := SearchFileNameInDir(EpPath, "ffmpeg")
			if FFmpegPath == "" || FFmpegPath != "" && !strings.Contains(filepath.Base(FFmpegPath), "ffmpeg") {
				LogPrintln(UUID, DeStr, "使用系统环境变量中的 FFmpeg")
				FFmpegPath = "ffmpeg"
			} else {
				LogPrintln(UUID, DeStr, "使用找到 FFmpeg 程序:", FFmpegPath)
			}

			FFmpegCmd := []string{
				FFmpegPath,
				"-i", filePath,
				"-f", "image2pipe",
				"-pix_fmt", "rgb24",
				"-vcodec", "rawvideo",
				"-",
			}
			FFmpegProcess := exec.Command(FFmpegCmd[0], FFmpegCmd[1:]...)
			FFmpegStdout, err := FFmpegProcess.StdoutPipe()
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法创建 FFmpeg 标准输出管道:", err)
				return
			}
			err = FFmpegProcess.Start()
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法启动 FFmpeg 进程:", err)
				return
			}

			// 记录数据
			isRecord := false
			var recordData [][]byte

			enc, err := reedsolomon.New(KGValue, MGValue-KGValue)
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法创建RS解码器:", err)
				return
			}

			bar := pb.StartNew(frameCount)
			i := 0
			allDataNum := 0
			for {
				// 检测是否暂停
				if UUID != "" {
					if GetTaskList[indexU].IsPaused {
						time.Sleep(time.Second)
						continue
					}
				} else {
					if isPaused {
						time.Sleep(time.Second)
						continue
					}
				}
				rawData := make([]byte, videoWidth*videoHeight*3)
				readBytes := 0
				exitFlag := false
				for readBytes < len(rawData) {
					n, err := FFmpegStdout.Read(rawData[readBytes:])
					if err != nil {
						exitFlag = true
						break
					}
					readBytes += n
				}
				if exitFlag {
					break
				}
				img := RawDataToImage(rawData, videoWidth, videoHeight)
				// 类型：
				// 0: 数据帧
				// 1: 空白帧
				// 2: 空白起始帧
				// 3: 空白终止帧
				data, t := Image2Data(img)
				if t == 1 {
					//LogPrintln(UUID, DeStr, "检测到空白帧，跳过")
					i++
					continue
				}

				// 检查是否是空白起始帧
				if t == 2 {
					//LogPrintln(UUID, DeStr, "检测到空白起始帧")
					// 检查是否没有找到终止帧
					if isRecord {
						for {
							isRecord = false
							//LogPrintln(UUID, DeStr, "本轮检测到", len(recordData), "帧数据")
							if len(recordData) == 0 {
								// 没有检查到数据，直接退出即可
								LogPrintln(UUID, DeStr, "检测到终止帧重复，跳过操作")
								break
							}
							// 对数据进行排序等操作
							sortShards := ProcessSlices(recordData, MGValue)
							// 删除记录数据
							recordData = make([][]byte, 0)
							var dataShards [][]byte
							// 检查整理后的长度是否为预期长度且nil元素数量小于等于MGValue-KGValue
							if len(sortShards) == MGValue && countNilElements(sortShards) <= MGValue-KGValue {
								// 修改 sortShards 的空白数据为 nil
								for oiu := range sortShards {
									if len(sortShards[oiu]) >= 4 {
										continue
									}
									sortShards[oiu] = nil
								}
								// 删除索引，复制数据到新切片
								dataShards = MakeMax2ByteSlice(ExtractForwardElements(sortShards, 4), videoHeight*videoWidth/8-4, MGValue)
								// 数据将开始重建
								ok, err := enc.Verify(dataShards)
								if !ok {
									//LogPrintln(UUID, DeStr, "检测到数据出现损坏，开始重建数据")
									//LogPrintln(UUID, "输出一些详细的信息供参考：")
									//LogPrintln(UUID, "数据帧数量:", len(sortShards))
									//LogPrintln(UUID, "数据帧长度:", len(sortShards[0]))
									//for oiu := range sortShards {
									//	if len(sortShards[oiu]) >= 4 {
									//		LogPrintln(UUID, "数据帧索引", oiu, ":", sortShards[oiu][:4])
									//		if oiu == 0 {
									//			LogPrintln(UUID, sortShards[oiu])
									//		}
									//		continue
									//	}
									//	sortShards[oiu] = nil
									//	LogPrintln(UUID, "数据帧索引(u)", oiu, ":", sortShards[oiu])
									//}
									for {
										err = enc.Reconstruct(dataShards)
										if err != nil {
											// 尝试进行二次重建
											isReconstructSuccess := false
											reconstructData := dataShards
											for ytp := range dataShards {
												copyDataShards := dataShards
												copyDataShards[ytp] = nil
												err = enc.Reconstruct(copyDataShards)
												if err != nil {
													continue
												}
												ok, err = enc.Verify(copyDataShards)
												if !ok {
													continue
												}
												if err != nil {
													continue
												}
												isReconstructSuccess = true
												reconstructData = copyDataShards
											}
											if !isReconstructSuccess {
												// 重建失败，数据出现无法修复的错误
												errorDataNum++
												LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
												LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：编码视频出现数据损坏且两次重建均失败，建议缩短分片编码视频的时长/增大文件冗余量，这样可以有效降低错误发生的概率)")
												LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
												LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
												LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
												LogPrintln("")
												bar.Finish()
												if errorDataNum > MGValue-KGValue {
													LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
													return
												}
												return
											}
											dataShards = reconstructData
										}
										ok, err = enc.Verify(dataShards)
										if !ok {
											// 尝试进行二次重建
											isReconstructSuccess := false
											reconstructData := dataShards
											for ytp := range dataShards {
												copyDataShards := dataShards
												copyDataShards[ytp] = nil
												err = enc.Reconstruct(copyDataShards)
												if err != nil {
													continue
												}
												ok, err = enc.Verify(copyDataShards)
												if !ok {
													continue
												}
												if err != nil {
													continue
												}
												isReconstructSuccess = true
												reconstructData = copyDataShards
											}
											if !isReconstructSuccess {
												// 重建失败，数据出现无法修复的错误
												errorDataNum++
												LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
												LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：编码视频出现数据损坏且两次重建均失败，建议缩短分片编码视频的时长/增大文件冗余量，这样可以有效降低错误发生的概率)")
												LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
												LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
												LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
												LogPrintln("")
												bar.Finish()
												if errorDataNum > MGValue-KGValue {
													LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
													return
												}
												return
											}
											dataShards = reconstructData
										}
										if err != nil {
											// 尝试进行二次重建
											isReconstructSuccess := false
											reconstructData := dataShards
											for ytp := range dataShards {
												copyDataShards := dataShards
												copyDataShards[ytp] = nil
												err = enc.Reconstruct(copyDataShards)
												if err != nil {
													continue
												}
												ok, err = enc.Verify(copyDataShards)
												if !ok {
													continue
												}
												if err != nil {
													continue
												}
												isReconstructSuccess = true
												reconstructData = copyDataShards
											}
											if !isReconstructSuccess {
												// 重建失败，数据出现无法修复的错误
												errorDataNum++
												LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
												LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：编码视频出现数据损坏且两次重建均失败，建议缩短分片编码视频的时长/增大文件冗余量，这样可以有效降低错误发生的概率)")
												LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
												LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
												LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
												LogPrintln("")
												bar.Finish()
												if errorDataNum > MGValue-KGValue {
													LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
													return
												}
												return
											}
											dataShards = reconstructData
										}
										//LogPrintln(UUID, DeStr, "数据重建成功")
										break
									}
								}
							} else {
								// 数据出现无法修复的错误
								errorDataNum++
								LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
								LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：数据丢失过多，出现了超出冗余数据长度的较多空白元素，适当增大 MG 和 KG 和缓解此问题)")
								LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
								LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
								LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
								LogPrintln("")
								bar.Finish()
								if errorDataNum > MGValue-KGValue {
									LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
									return
								}
								return
							}

							if errorDataNum > MGValue-KGValue {
								LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
								bar.Finish()
								return
							}

							dataOrigin := dataShards[:len(dataShards)-MGValue+KGValue]
							// 写入到文件
							for _, dataW := range dataOrigin {
								_, err := outputFile.Write(dataW)
								if err != nil {
									LogPrintln(UUID, DeStr, ErStr, "写入文件失败:", err)
									break
								}
							}
							break
						}
					}
					isRecord = true
					i++
					continue
				}

				// 检查是否是空白终止帧
				if t == 3 {
					//LogPrintln(UUID, DeStr, "检测到空白终止帧")

					// 检查是否没有找到起始帧
					if !isRecord {
						isRecord = true
					}

					for {
						isRecord = false
						//LogPrintln(UUID, DeStr, "本轮检测到", len(recordData), "帧数据")
						if len(recordData) == 0 {
							// 没有检查到数据，直接退出即可
							LogPrintln(UUID, DeStr, "检测到终止帧重复，跳过操作")
							break
						}
						// 对数据进行排序等操作
						sortShards := ProcessSlices(recordData, MGValue)
						// 删除记录数据
						recordData = make([][]byte, 0)
						var dataShards [][]byte
						// 检查整理后的长度是否为预期长度且nil元素数量小于等于MGValue-KGValue
						if len(sortShards) == MGValue && countNilElements(sortShards) <= MGValue-KGValue {
							// 修改 sortShards 的空白数据为 nil
							for oiu := range sortShards {
								if len(sortShards[oiu]) >= 4 {
									continue
								}
								sortShards[oiu] = nil
							}
							// 删除索引，复制数据到新切片
							dataShards = MakeMax2ByteSlice(ExtractForwardElements(sortShards, 4), videoHeight*videoWidth/8-4, MGValue)
							// 数据将开始重建
							ok, err := enc.Verify(dataShards)
							if !ok {
								//LogPrintln(UUID, DeStr, "检测到数据出现损坏，开始重建数据")
								//LogPrintln(UUID, "输出一些详细的信息供参考：")
								//LogPrintln(UUID, "数据帧数量:", len(sortShards))
								//LogPrintln(UUID, "数据帧长度:", len(sortShards[0]))
								//for oiu := range sortShards {
								//	if len(sortShards[oiu]) >= 4 {
								//		LogPrintln(UUID, "数据帧索引", oiu, ":", sortShards[oiu][:4])
								//		if oiu == 0 {
								//			LogPrintln(UUID, sortShards[oiu])
								//		}
								//		continue
								//	}
								//	sortShards[oiu] = nil
								//	LogPrintln(UUID, "数据帧索引(u)", oiu, ":", sortShards[oiu])
								//}
								for {
									err = enc.Reconstruct(dataShards)
									if err != nil {
										// 尝试进行二次重建
										isReconstructSuccess := false
										reconstructData := dataShards
										for ytp := range dataShards {
											copyDataShards := dataShards
											copyDataShards[ytp] = nil
											err = enc.Reconstruct(copyDataShards)
											if err != nil {
												continue
											}
											ok, err = enc.Verify(copyDataShards)
											if !ok {
												continue
											}
											if err != nil {
												continue
											}
											isReconstructSuccess = true
											reconstructData = copyDataShards
										}
										if !isReconstructSuccess {
											// 重建失败，数据出现无法修复的错误
											errorDataNum++
											LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
											LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：编码视频出现数据损坏且两次重建均失败，建议缩短分片编码视频的时长/增大文件冗余量，这样可以有效降低错误发生的概率)")
											LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
											LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
											LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
											LogPrintln("")
											bar.Finish()
											if errorDataNum > MGValue-KGValue {
												LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
												return
											}
											return
										}
										dataShards = reconstructData
									}
									ok, err = enc.Verify(dataShards)
									if !ok {
										// 尝试进行二次重建
										isReconstructSuccess := false
										reconstructData := dataShards
										for ytp := range dataShards {
											copyDataShards := dataShards
											copyDataShards[ytp] = nil
											err = enc.Reconstruct(copyDataShards)
											if err != nil {
												continue
											}
											ok, err = enc.Verify(copyDataShards)
											if !ok {
												continue
											}
											if err != nil {
												continue
											}
											isReconstructSuccess = true
											reconstructData = copyDataShards
										}
										if !isReconstructSuccess {
											// 重建失败，数据出现无法修复的错误
											errorDataNum++
											LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
											LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：编码视频出现数据损坏且两次重建均失败，建议缩短分片编码视频的时长/增大文件冗余量，这样可以有效降低错误发生的概率)")
											LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
											LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
											LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
											LogPrintln("")
											bar.Finish()
											if errorDataNum > MGValue-KGValue {
												LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
												return
											}
											return
										}
										dataShards = reconstructData
									}
									if err != nil {
										// 尝试进行二次重建
										isReconstructSuccess := false
										reconstructData := dataShards
										for ytp := range dataShards {
											copyDataShards := dataShards
											copyDataShards[ytp] = nil
											err = enc.Reconstruct(copyDataShards)
											if err != nil {
												continue
											}
											ok, err = enc.Verify(copyDataShards)
											if !ok {
												continue
											}
											if err != nil {
												continue
											}
											isReconstructSuccess = true
											reconstructData = copyDataShards
										}
										if !isReconstructSuccess {
											// 重建失败，数据出现无法修复的错误
											errorDataNum++
											LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
											LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：编码视频出现数据损坏且两次重建均失败，建议缩短分片编码视频的时长/增大文件冗余量，这样可以有效降低错误发生的概率)")
											LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
											LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
											LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
											LogPrintln("")
											bar.Finish()
											if errorDataNum > MGValue-KGValue {
												LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
												return
											}
											return
										}
										dataShards = reconstructData
									}
									//LogPrintln(UUID, DeStr, "数据重建成功")
									break
								}
							}
						} else {
							// 数据出现无法修复的错误
							errorDataNum++
							LogPrintln(UUID, DeStr, ErStr, "\n\n————————————————————————————————————————————")
							LogPrintln(UUID, DeStr, ErStr, "警告：数据出现无法修复的错误，停止输出数据到分片文件(原因：数据丢失过多，出现了超出冗余数据长度的较多空白元素，适当增大 MG 和 KG 和缓解此问题)")
							LogPrintln(UUID, DeStr, ErStr, "当前无法恢复的切片文件数量:", errorDataNum)
							LogPrintln(UUID, DeStr, ErStr, "最大可丢失的切片文件数量:", MGValue-KGValue)
							LogPrintln(UUID, DeStr, ErStr, "————————————————————————————————————————————")
							LogPrintln("")
							bar.Finish()
							if errorDataNum > MGValue-KGValue {
								LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
								return
							}
							return
						}

						if errorDataNum > MGValue-KGValue {
							LogPrintln(UUID, DeStr, ErStr, "无法修复的切片文件数量已经超过最大可丢失的切片文件数量，停止解码")
							bar.Finish()
							return
						}

						dataOrigin := dataShards[:len(dataShards)-MGValue+KGValue]
						// 写入到文件
						for _, dataW := range dataOrigin {
							_, err := outputFile.Write(dataW)
							if err != nil {
								LogPrintln(UUID, DeStr, ErStr, "写入文件失败:", err)
								break
							}
						}
						break
					}

					i++
					continue
				}

				// 直接向 recordData 添加数据帧
				recordData = append(recordData, data)
				allDataNum++
				i++

				bar.SetCurrent(int64(i + 1))
				if i%30000 == 0 {
					LogPrintf(UUID, "\nDecode: 写入帧 %d 总帧 %d\n", i, frameCount)
				}
			}
			bar.Finish()

			err = FFmpegStdout.Close()
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "无法关闭 FFmpeg 标准输出管道:", err)
				return
			}
			err = FFmpegProcess.Wait()
			if err != nil {
				LogPrintln(UUID, DeStr, ErStr, "FFmpeg 命令执行失败:", err)
				return
			}
			outputFile.Close()

			if segmentLength != 0 {
				err := TruncateFile(segmentLength, outputFilePath)
				if err != nil {
					LogPrintln(UUID, DeStr, ErStr, "截断解码文件失败:", err)
					return
				}
			} else {
				// 删除解码文件的末尾连续的零字节
				LogPrintln(UUID, DeStr, "未提供原始文件的长度参数，默认删除解码文件的末尾连续的零字节来还原原始文件(无法还原尾部带零字节的分段文件)")
				err = RemoveTrailingZerosFromFile(outputFilePath)
				if err != nil {
					LogPrintln(UUID, DeStr, ErStr, "删除解码文件的末尾连续的零字节失败:", err)
					return
				}
			}

			// 为全局 ProgressRate 变量赋值
			for kp, kq := range GetTaskList {
				if kq.UUID == UUID {
					GetTaskList[kp].ProgressRate++
					// 计算正确的 progressNum
					GetTaskList[kp].ProgressNum = float64(GetTaskList[kp].ProgressRate) / float64(len(filePathList)) * 100
					break
				}
			}

			LogPrintln(UUID, DeStr, "完成")
			LogPrintln(UUID, DeStr, "使用配置：")
			LogPrintln(UUID, DeStr, "  ---------------------------")
			LogPrintln(UUID, DeStr, "  视频宽度:", videoWidth)
			LogPrintln(UUID, DeStr, "  视频高度:", videoHeight)
			LogPrintln(UUID, DeStr, "  总帧数:", frameCount)
			LogPrintln(UUID, DeStr, "  输入视频路径:", filePath)
			LogPrintln(UUID, DeStr, "  输出文件路径:", outputFilePath)
			LogPrintln(UUID, DeStr, "  ---------------------------")
		}(filePathIndex, filePath)
	}
	wg.Wait()

	if errorDataNum > MGValue-KGValue {
		LogPrintln(UUID, DeStr, ErStr, "恢复失败")
		return
	}

	isRuntime = false
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	LogPrintln(UUID, DeStr, "全部完成")
	LogPrintf(UUID, DeStr+" 总共耗时%f秒\n", allDuration.Seconds())
}
