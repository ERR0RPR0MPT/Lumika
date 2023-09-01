package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	en                    = "Encode:"
	de                    = "Decode:"
	add                   = "Add:"
	get                   = "Get:"
	addMLevel             = 25
	addKLevel             = 19
	encodeVideoSizeLevel  = 32
	encodeOutputFPSLevel  = 24
	encodeMaxSecondsLevel = 35990
	encodeFFmpegModeLevel = "medium"
	defaultHashLength     = 10
)

type FecFileConfig struct {
	Name          string `json:"n"`
	Summary       string `json:"s"`
	Hash          string `json:"h"`
	SegmentLength int64  `json:"sl"`
	SegmentNumber int64  `json:"sn"`
}

func PressEnterToContinue() {
	fmt.Print("请按回车键继续...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Println("清屏失败:", err)
		return
	}
}

func CalculateFileHash(filePath string, cut int) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}
	hashValue := hash.Sum(nil)
	hashString := hex.EncodeToString(hashValue)
	return hashString[:cut]
}

func GetFileSize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

// RemoveTrailingZerosFromFile 从文件中删除末尾的连续零字节
func RemoveTrailingZerosFromFile(filename string) error {
	// 打开文件进行读取和写入
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := fileInfo.Size()
	// 从文件末尾开始向前搜索，找到第一个非零字节的位置
	position := fileSize - 1
	for position >= 0 {
		_, err = file.Seek(position, io.SeekStart)
		if err != nil {
			return err
		}
		var b [1]byte
		_, err = file.Read(b[:])
		if err != nil {
			return err
		}
		if b[0] != 0 {
			break
		}
		position--
	}
	// 如果找到了非零字节，则将文件截断到该位置
	if position >= 0 {
		err = file.Truncate(position + 1)
		if err != nil {
			return err
		}
	} else {
		// 如果文件中所有字节都是零，则将文件截断为长度为0
		err = file.Truncate(0)
		if err != nil {
			return err
		}
	}
	return nil
}

func TruncateFile(dataLength int64, filePath string) error {
	// 打开文件以读写模式
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	// 移动文件指针到指定位置
	_, err = file.Seek(dataLength, 0)
	if err != nil {
		return err
	}
	// 截断文件
	err = file.Truncate(dataLength)
	if err != nil {
		return err
	}
	return nil
}

func Data2Image(data []byte, size int) image.Image {
	// 计算最大可表示的数据长度
	maxDataLength := size * size / 8
	// 检查数据长度是否匹配，如果不匹配则进行填充
	if len(data) < maxDataLength {
		paddingLength := maxDataLength - len(data)
		padding := make([]byte, paddingLength)
		data = append(data, padding...)
	} else if len(data) > maxDataLength {
		fmt.Println("Data2Image: 警告: 数据过长，将进行截断")
		data = data[:maxDataLength]
	}

	// 创建新的RGBA图像对象
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// 遍历数据并设置像素颜色
	for i := 0; i < maxDataLength; i++ {
		b := data[i]
		for j := 0; j < 8; j++ {
			bit := (b >> uint(7-j)) & 1
			var c color.RGBA
			if bit == 0 {
				c = color.RGBA{A: 255} // 黑色
			} else {
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255} // 白色
			}
			x := (i*8 + j) % size
			y := (i*8 + j) / size
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// Image2Data 从图像中恢复数据
func Image2Data(img image.Image) []byte {
	bounds := img.Bounds()
	size := bounds.Size().X
	dataLength := size * size / 8
	data := make([]byte, dataLength)

	// 遍历图像像素并提取数据
	for i := 0; i < dataLength; i++ {
		b := byte(0)
		for j := 0; j < 8; j++ {
			x := (i*8 + j) % size
			y := (i*8 + j) / size
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0x7FFF {
				b |= 1 << uint(7-j)
			}
		}
		data[i] = b
	}
	return data
}

func RawDataToImage(rawData []byte, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := ((y * width) + x) * 3
			img.Set(x, y, color.RGBA{R: rawData[offset], G: rawData[offset+1], B: rawData[offset+2], A: 255})
		}
	}
	return img
}

func GetUserInput(s string) string {
	if s == "" {
		s = "请输入内容: "
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(s)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("获取用户输入失败:", err)
		return ""
	}
	return strings.TrimSpace(input)
}

func AddOutputToFileName(path string, ext string) string {
	filename := filepath.Base(path)
	extension := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, extension)
	newName := "output_" + strings.ReplaceAll(name, ".", "_") + ext
	newPath := filepath.Join(filepath.Dir(path), newName)
	return newPath
}

func GenerateFileDxDictionary(root string, ex string) (map[int]string, error) {
	fileDict := make(map[int]string)
	index := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ex {
			if strings.Contains(filepath.Base(path), "lumika") {
				return nil
			}
			fileDict[index] = path
			index++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]int, 0, len(fileDict))
	for key := range fileDict {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	sortedFileDict := make(map[int]string)
	for _, key := range keys {
		sortedFileDict[key] = fileDict[key]
	}
	return sortedFileDict, nil
}

func GenerateFileDictionary(root string) (map[int]string, error) {
	fileDict := make(map[int]string)
	index := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.Contains(filepath.Base(path), "lumika") {
				return nil
			}
			fileDict[index] = path
			index++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]int, 0, len(fileDict))
	for key := range fileDict {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	sortedFileDict := make(map[int]string)
	for _, key := range keys {
		sortedFileDict[key] = fileDict[key]
	}
	return sortedFileDict, nil
}

func Encode(fileDir string, videoSize int, outputFPS int, maxSeconds int, encodeFFmpegMode string, auto bool) (segmentLength int64) {
	if videoSize%8 != 0 {
		fmt.Println(en, "视频大小必须是8的倍数")
		return 0
	}

	// 当没有检测到videoFileDir时，自动匹配
	if fileDir == "" {
		fmt.Println(en, "自动使用程序所在目录作为输入目录")
		fd, err := os.Executable()
		if err != nil {
			fmt.Println(en, "获取程序所在目录失败:", err)
			return 0
		}
		fileDir = filepath.Dir(fd)
	}

	// 检查输入文件夹是否存在
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		fmt.Println(en, "输入文件夹不存在:", err)
		return 0
	}

	fmt.Println(en, "当前目录:", fileDir)

	fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
	if err != nil {
		fmt.Println(en, "无法生成文件列表:", err)
		return 0
	}
	filePathList := make([]string, 0)
	for {
		if len(fileDict) == 0 {
			fmt.Println(en, "当前目录下没有.fec文件，请将需要编码的文件放到当前目录下")
			return 0
		}
		fmt.Println(en, "请选择需要编码的.fec文件，输入索引并回车来选择")
		fmt.Println(en, "如果需要编码当前目录下的所有.fec文件，请直接输入回车")
		for index := 0; index < len(fileDict); index++ {
			fmt.Println("Encode:", strconv.Itoa(index)+":", fileDict[index])
		}
		var result string
		if auto {
			result = ""
		} else {
			result = GetUserInput("")
		}
		if result == "" {
			fmt.Println(en, "注意：开始编码当前目录下的所有.fec文件")
			for _, filePath := range fileDict {
				filePathList = append(filePathList, filePath)
			}
			break
		} else {
			index, err := strconv.Atoi(result)
			if err != nil {
				fmt.Println(en, "输入索引不是数字，请重新输入")
				continue
			}
			if index < 0 || index >= len(fileDict) {
				fmt.Println(en, "输入索引超出范围，请重新输入")
				continue
			}
			filePathList = append(filePathList, fileDict[index])
			break
		}
	}

	// 启动多个goroutine
	var wg sync.WaitGroup
	allStartTime := time.Now()

	// 遍历需要处理的文件列表
	for fileIndexNum, filePath := range filePathList {
		fmt.Println(en, "开始编码第", fileIndexNum+1, "个文件，路径:", filePath)
		wg.Add(1) // 增加计数器
		go func(fileIndexNum int, filePath string) {
			defer wg.Done() // 减少计数器

			// 读取文件
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Println(en, "无法打开文件:", err)
				return
			}

			outputFilePath := AddOutputToFileName(filePath, ".mp4")                    // 输出文件路径
			fileLength := GetFileSize(filePath)                                        // 输入文件长度
			dataSliceLen := videoSize * videoSize / 8                                  // 每帧存储的有效数据
			allFrameNum := int(math.Ceil(float64(fileLength) / float64(dataSliceLen))) // 生成总帧数
			allSeconds := int(math.Ceil(float64(allFrameNum) / float64(outputFPS)))    // 总时长(秒)

			// 检查时长是否超过限制
			if allSeconds > maxSeconds {
				fmt.Println(en, "警告：生成的段视频时长超过限制("+strconv.Itoa(allSeconds)+"s>"+strconv.Itoa(maxSeconds)+"s)")
				fmt.Println(en, "请调整M值、K值、输出帧率、最大生成时长来满足要求")
				GetUserInput("请按回车键继续...")
				os.Exit(0)
			}

			segmentLength = fileLength

			fmt.Println(en, "开始运行")
			fmt.Println(en, "使用配置：")
			fmt.Println(en, "  ---------------------------")
			fmt.Println(en, "  输入文件:", filePath)
			fmt.Println(en, "  输出文件:", outputFilePath)
			fmt.Println(en, "  输入文件长度:", fileLength)
			fmt.Println(en, "  每帧数据长度:", dataSliceLen)
			fmt.Println(en, "  帧大小:", videoSize)
			fmt.Println(en, "  输出帧率:", outputFPS)
			fmt.Println(en, "  生成总帧数:", allFrameNum)
			fmt.Println(en, "  总时长: ", strconv.Itoa(allSeconds)+"s")
			fmt.Println(en, "  FFmpeg 预设:", encodeFFmpegMode)
			fmt.Println(en, "  ---------------------------")

			ffmpegCmd := []string{
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

			ffmpegProcess := exec.Command("ffmpeg", ffmpegCmd...)
			stdin, err := ffmpegProcess.StdinPipe()
			if err != nil {
				fmt.Println(en, "无法创建 ffmpeg 的标准输入管道:", err)
				return
			}
			err = ffmpegProcess.Start()
			if err != nil {
				fmt.Println(en, "无法启动 ffmpeg 子进程:", err)
				return
			}

			i := 0
			// 启动进度条
			bar := pb.StartNew(int(fileLength))

			fileNowLength := 0
			for {
				if len(fileData) == 0 {
					break
				}
				data := make([]byte, dataSliceLen)
				if len(fileData) >= dataSliceLen {
					data = fileData[:dataSliceLen]
					fileData = fileData[dataSliceLen:]
				} else {
					data = fileData
					fileData = nil
				}

				i++
				fileNowLength += len(data)

				bar.SetCurrent(int64(fileNowLength))
				if i%1000 == 0 {
					fmt.Printf("\nEncode: 构建帧 %d, 已构建数据 %d, 总数据 %d\n", i, fileNowLength, fileLength)
				}

				// 生成带数据的图像
				img := Data2Image(data, videoSize)

				imageBuffer := new(bytes.Buffer)
				err = png.Encode(imageBuffer, img)
				if err != nil {
					return
				}
				imageData := imageBuffer.Bytes()

				_, err = stdin.Write(imageData)
				if err != nil {
					fmt.Println(en, "无法写入帧数据到 ffmpeg:", err)
					return
				}
				imageBuffer = nil
				imageData = nil
			}
			bar.Finish()

			// 关闭 ffmpeg 的标准输入管道，等待子进程完成
			err = stdin.Close()
			if err != nil {
				fmt.Println(en, "无法关闭 ffmpeg 的标准输入管道:", err)
				return
			}
			if err := ffmpegProcess.Wait(); err != nil {
				fmt.Println(en, "ffmpeg 子进程执行失败:", err)
				return
			}

			fmt.Println(en, "完成")
			fmt.Println(en, "使用配置：")
			fmt.Println(en, "  ---------------------------")
			fmt.Println(en, "  输入文件:", filePath)
			fmt.Println(en, "  输出文件:", outputFilePath)
			fmt.Println(en, "  输入文件长度:", fileLength)
			fmt.Println(en, "  每帧数据长度:", dataSliceLen)
			fmt.Println(en, "  帧大小:", videoSize)
			fmt.Println(en, "  输出帧率:", outputFPS)
			fmt.Println(en, "  生成总帧数:", allFrameNum)
			fmt.Println(en, "  总时长: ", strconv.Itoa(allSeconds)+"s")
			fmt.Println(en, "  FFmpeg 预设:", encodeFFmpegMode)
			fmt.Println(en, "  ---------------------------")
		}(fileIndexNum, filePath)
	}
	wg.Wait()
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	fmt.Printf(en+" 总共耗时%f秒\n", allDuration.Seconds())
	fmt.Println(en, "所有选择的.fec文件已编码完成，编码结束")
	return segmentLength
}

func Decode(videoFileDir string, segmentLength int64, filePathList []string) {
	// 当没有检测到videoFileDir时，自动匹配
	if videoFileDir == "" {
		fmt.Println(de, "自动使用程序所在目录作为输入目录")
		fd, err := os.Executable()
		if err != nil {
			fmt.Println(de, "获取程序所在目录失败:", err)
			return
		}
		videoFileDir = filepath.Dir(fd)
	}

	// 检查输入文件夹是否存在
	if _, err := os.Stat(videoFileDir); os.IsNotExist(err) {
		fmt.Println(de, "输入文件夹不存在:", err)
		return
	}

	fmt.Println(de, "当前目录:", videoFileDir)

	fileDict, err := GenerateFileDxDictionary(videoFileDir, ".mp4")
	if err != nil {
		fmt.Println(de, "无法生成视频列表:", err)
		return
	}

	if filePathList == nil {
		filePathList = make([]string, 0)
		for {
			if len(fileDict) == 0 {
				fmt.Println(en, "当前目录下没有.mp4文件，请将需要解码的视频文件放到当前目录下")
				return
			}
			fmt.Println(en, "请选择需要编码的.mp4文件，输入索引并回车来选择")
			fmt.Println(en, "如果需要编码当前目录下的所有.mp4文件，请直接输入回车")
			for index := 0; index < len(fileDict); index++ {
				fmt.Println("Encode:", strconv.Itoa(index)+":", fileDict[index])
			}
			result := GetUserInput("")
			if result == "" {
				fmt.Println(en, "注意：开始编码当前目录下的所有.mp4文件")
				for _, filePath := range fileDict {
					filePathList = append(filePathList, filePath)
				}
				break
			} else {
				index, err := strconv.Atoi(result)
				if err != nil {
					fmt.Println(en, "输入索引不是数字，请重新输入")
					continue
				}
				if index < 0 || index >= len(fileDict) {
					fmt.Println(en, "输入索引超出范围，请重新输入")
					continue
				}
				filePathList = append(filePathList, fileDict[index])
				break
			}
		}
	}

	var wg sync.WaitGroup

	// 遍历解码所有文件
	allStartTime := time.Now()
	for filePathIndex, filePath := range filePathList {
		wg.Add(1) // 增加计数器
		go func(filePathIndex int, filePath string) {
			defer wg.Done() // 减少计数器
			fmt.Println(de, "开始解码第", filePathIndex+1, "个编码文件:", filePath)

			cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", filePath)
			output, err := cmd.Output()
			if err != nil {
				fmt.Println(de, "FFprobe 启动失败，请检查文件是否存在:", err)
				return
			}
			result := strings.Split(string(output), ",")
			if len(result) != 2 {
				fmt.Println(de, "无法读取视频宽高，请检查视频文件是否正确")
				return
			}
			videoWidth, err := strconv.Atoi(strings.TrimSpace(result[0]))
			if err != nil {
				fmt.Println(de, "无法读取视频宽高，请检查视频文件是否正确:", err)
				return
			}
			videoHeight, err := strconv.Atoi(strings.TrimSpace(result[1]))
			if err != nil {
				fmt.Println(de, "无法读取视频宽高，请检查视频文件是否正确:", err)
				return
			}
			cmd = exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=nb_frames", "-of", "default=nokey=1:noprint_wrappers=1", filePath)
			output, err = cmd.Output()
			if err != nil {
				fmt.Println(de, "执行 ffprobe 命令时出错:", err)
				return
			}
			frameCount, err := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(string(output)))
			if err != nil {
				fmt.Println(de, "解析视频帧数时出错:", err)
				return
			}

			// 设置输出路径
			outputFilePath := filepath.Join(videoFileDir, filepath.Base(filePath)+".fec")

			fmt.Println(de, "开始解码")
			fmt.Println(de, "使用配置：")
			fmt.Println(de, "  ---------------------------")
			fmt.Println(de, "  视频宽度:", videoWidth)
			fmt.Println(de, "  视频高度:", videoHeight)
			fmt.Println(de, "  总帧数:", frameCount)
			fmt.Println(de, "  输入视频路径:", filePath)
			fmt.Println(de, "  输出文件路径:", outputFilePath)
			fmt.Println(de, "  ---------------------------")

			// 打开输出文件
			fmt.Println(de, "创建输出文件:", outputFilePath)
			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				fmt.Println(de, "无法创建输出文件:", err)
				return
			}

			ffmpegCmd := []string{
				"ffmpeg",
				"-i", filePath,
				"-f", "image2pipe",
				"-pix_fmt", "rgb24",
				"-vcodec", "rawvideo",
				"-",
			}
			ffmpegProcess := exec.Command(ffmpegCmd[0], ffmpegCmd[1:]...)
			ffmpegStdout, err := ffmpegProcess.StdoutPipe()
			if err != nil {
				fmt.Println("无法创建 FFmpeg 标准输出管道:", err)
				return
			}
			err = ffmpegProcess.Start()
			if err != nil {
				fmt.Println(de, "无法启动 FFmpeg 进程:", err)
				return
			}

			bar := pb.StartNew(frameCount)
			i := 0
			for {
				rawData := make([]byte, videoWidth*videoHeight*3)
				readBytes := 0
				exitFlag := false
				for readBytes < len(rawData) {
					n, err := ffmpegStdout.Read(rawData[readBytes:])
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
				data := Image2Data(img)

				if data == nil {
					fmt.Println(de, "还原原始数据失败")
					return
				}
				bar.SetCurrent(int64(i + 1))
				if i%1000 == 0 {
					fmt.Printf("\nDecode: 写入帧 %d 总帧 %d\n", i, frameCount)
				}
				_, err = outputFile.Write(data)
				if err != nil {
					fmt.Println(de, "写入文件失败:", err)
					break
				}
				i++
			}
			bar.Finish()
			err = ffmpegStdout.Close()
			if err != nil {
				fmt.Println(de, "无法关闭 FFmpeg 标准输出管道:", err)
				return
			}
			err = ffmpegProcess.Wait()
			if err != nil {
				fmt.Println(de, "FFmpeg 命令执行失败:", err)
				return
			}
			outputFile.Close()

			if segmentLength != 0 {
				err := TruncateFile(segmentLength, outputFilePath)
				if err != nil {
					fmt.Println(de, "截断解码文件失败:", err)
					return
				}
			} else {
				// 删除解码文件的末尾连续的零字节
				fmt.Println(de, "未提供原始文件的长度参数，默认删除解码文件的末尾连续的零字节来还原原始文件(无法还原尾部带零字节)")
				err = RemoveTrailingZerosFromFile(outputFilePath)
				if err != nil {
					fmt.Println(de, "删除解码文件的末尾连续的零字节失败:", err)
					return
				}
			}

			fmt.Println(de, "完成")
			fmt.Println(de, "使用配置：")
			fmt.Println(de, "  ---------------------------")
			fmt.Println(de, "  视频宽度:", videoWidth)
			fmt.Println(de, "  视频高度:", videoHeight)
			fmt.Println(de, "  总帧数:", frameCount)
			fmt.Println(de, "  输入视频路径:", filePath)
			fmt.Println(de, "  输出文件路径:", outputFilePath)
			fmt.Println(de, "  ---------------------------")
		}(filePathIndex, filePath)
	}
	wg.Wait()

	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	fmt.Println(de, "全部完成")
	fmt.Printf(de+" 总共耗时%f秒\n", allDuration.Seconds())
}

func Add() {
	fmt.Println(add, "使用 \""+os.Args[0]+" help\" 查看帮助")

	fd, err := os.Executable()
	if err != nil {
		fmt.Println(en, "获取程序所在目录失败:", err)
		return
	}
	fileDir := filepath.Dir(fd)

	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		fmt.Println(en, "输入文件夹不存在:", err)
		return
	}

	fmt.Println(add, "当前目录:", fileDir)

	fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
	if err != nil {
		fmt.Println(en, "无法生成文件列表:", err)
		return
	}

	if len(fileDict) != 0 {
		fmt.Println(add, "错误：检测到目录下存在 .fec 文件，请先删除 .fec 文件再进行添加")
		return
	}

	// 设置默认的文件名
	fileDict, err = GenerateFileDictionary(fileDir)
	if err != nil {
		fmt.Println(add, "无法生成文件列表:", err)
		return
	}
	filePathList := make([]string, 0)
	for {
		if len(fileDict) == 0 {
			fmt.Println(add, "当前目录下没有文件，请将需要编码的文件放到当前目录下")
			return
		}
		fmt.Println(add, "请选择需要编码的文件，输入索引并回车来选择")
		fmt.Println(add, "如果需要编码当前目录下的所有文件，请直接输入回车")
		for index := 0; index < len(fileDict); index++ {
			fmt.Println("Encode:", strconv.Itoa(index)+":", fileDict[index])
		}
		result := GetUserInput("")
		if result == "" {
			fmt.Println(en, "注意：开始编码当前目录下的所有文件")
			for _, filePath := range fileDict {
				filePathList = append(filePathList, filePath)
			}
			break
		} else {
			index, err := strconv.Atoi(result)
			if err != nil {
				fmt.Println(add, "输入索引不是数字，请重新输入")
				continue
			}
			if index < 0 || index >= len(fileDict) {
				fmt.Println(add, "输入索引超出范围，请重新输入")
				continue
			}
			filePathList = append(filePathList, fileDict[index])
			break
		}
	}

	// 设置默认的摘要
	fmt.Println(add, "请输入摘要，可以作为文件内容的简介。例如：\"这是一个相册的压缩包\"")
	defaultSummary := GetUserInput("")

	// 设置M的值
	fmt.Println(add, "请输入 M 的值(0<=M<=256)，M 为最终生成的切片文件数量。默认：\""+strconv.Itoa(addMLevel)+"\"")
	defaultM, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		fmt.Println(add, "自动设置 M = "+strconv.Itoa(addMLevel))
		defaultM = addMLevel
	}
	if defaultM == 0 {
		fmt.Println(add, "错误: M 的值不能为 0，自动设置 M = "+strconv.Itoa(addMLevel))
		defaultM = addMLevel
	}

	// 设置K的值
	fmt.Println(add, "请输入 K 的值(0<=K<=256)，K 为恢复原始文件所需的最少完整切片数量。默认：\""+strconv.Itoa(addKLevel)+"\"")
	defaultK, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		fmt.Println(add, "自动设置 K = "+strconv.Itoa(addKLevel))
		defaultK = addKLevel
	}
	if defaultK == 0 {
		fmt.Println(add, "错误: K 的值不能为 0，自动设置 K = "+strconv.Itoa(addKLevel))
		defaultK = addKLevel
	}

	for ai, filePath := range filePathList {
		fmt.Println(add, "开始编码第"+strconv.Itoa(ai)+"个文件:", filePath)
		// 设置默认文件名
		defaultFileName := filepath.Base(filePath)
		defaultOutputDirName := "output_" + strings.ReplaceAll(defaultFileName, ".", "_")
		defaultOutputDir := filepath.Join(fileDir, defaultOutputDirName)
		// 创建输出目录
		err = os.Mkdir(defaultOutputDir, os.ModePerm)
		if err != nil {
			fmt.Println(add, "创建输出目录失败:", err)
			return
		}
		fmt.Println(add, "使用默认文件名:", defaultFileName)
		fmt.Println(add, "使用默认输出目录:", defaultOutputDir)

		// 调用 zfec
		fmt.Println(add, "开始调用 zfec")
		zfecStartTime := time.Now()
		// zfec 的一个奇怪的Bug: 传入的文件必须是相对路径，否则 -d 指定的输出目录会无效
		relPath, err := filepath.Rel(fileDir, filePath)
		if err != nil {
			fmt.Println("Failed to calculate relative path:", err)
			return
		}
		zfecCmd := exec.Command("zfec", "-m", strconv.Itoa(defaultM), "-k", strconv.Itoa(defaultK), "-f", "-q", "-d", defaultOutputDir, relPath)
		err = zfecCmd.Run()
		if err != nil {
			fmt.Println("zfecCmd 命令执行出错:", err)
			return
		}
		zfecEndTime := time.Now()
		zfecDuration := zfecEndTime.Sub(zfecStartTime)
		fmt.Println(add, "zfec 调用完成，耗时:", zfecDuration)

		// 查找生成的 .fec 文件
		fileDict, err = GenerateFileDxDictionary(defaultOutputDir, ".fec")
		if err != nil {
			fmt.Println(en, "无法生成文件列表:", err)
			return
		}

		fmt.Println(add, "开始进行编码")
		segmentLength := Encode(defaultOutputDir, encodeVideoSizeLevel, encodeOutputFPSLevel, encodeMaxSecondsLevel, encodeFFmpegModeLevel, true)

		fmt.Println(add, "编码完成，开始生成配置")
		fecFileConfig := FecFileConfig{
			Name:          defaultFileName,
			Summary:       defaultSummary,
			Hash:          CalculateFileHash(filePath, defaultHashLength),
			SegmentLength: segmentLength,
		}
		fecFileConfigJson, err := json.Marshal(fecFileConfig)
		if err != nil {
			fmt.Println(add, "生成 JSON 配置失败:", err)
			return
		}
		// 转换为 Base64
		fecFileConfigBase64 := base64.StdEncoding.EncodeToString(fecFileConfigJson)
		fecFileConfigFilePath := filepath.Join(defaultOutputDir, "config_base64.txt")
		fmt.Println(add, "Base64 配置生成完成，开始写入文件:", fecFileConfigFilePath)
		err = os.WriteFile(fecFileConfigFilePath, []byte(fecFileConfigBase64), 0644)
		if err != nil {
			fmt.Println(add, "写入文件失败:", err)
			return
		}
		fmt.Println(add, "写入配置成功")

		fileDict, err = GenerateFileDxDictionary(defaultOutputDir, ".fec")
		if err != nil {
			fmt.Println(en, "无法生成文件列表:", err)
			return
		}
		if len(fileDict) != 0 {
			fmt.Println(add, "删除临时文件")
			for _, filePath := range fileDict {
				err = os.Remove(filePath)
				if err != nil {
					fmt.Println(add, "删除文件失败:", err)
					return
				}
			}
		}

		fmt.Println(add, "Base64 配置文件已生成，路径:", fecFileConfigFilePath)
		fmt.Println(add, "Base64:", fecFileConfigBase64)
		fmt.Println(add, "请将生成的 .mp4 fec 视频文件和 Base64 配置分享或发送给你的好友，对方可使用 \"get\" 子命令来获取文件")
		fmt.Println(add, "添加完成")
	}
}

func Get(base64Config string) {
	ep, _ := os.Executable()
	fileDir := filepath.Dir(ep)
	// 获取配置
	// 从运行目录检测是否存在配置文件
	configBase64FilePath := filepath.Join(fileDir, "config_base64.txt")
	if base64Config == "" {
		if _, err := os.Stat(configBase64FilePath); err == nil {
			fmt.Println(get, "检测到配置文件，是否使用该配置？ [Y/n]")
			result := GetUserInput("")
			if result == "y" || result == "Y" || result == "" {
				fmt.Println(get, "读取配置文件")
				// 读取文件
				configBase64Bytes, err := os.ReadFile(configBase64FilePath)
				if err != nil {
					fmt.Println(get, "读取文件失败:", err)
					return
				}
				base64Config = string(configBase64Bytes)
			}
		}
		if base64Config == "" {
			fmt.Println(get, "请输入 Base64 配置")
			base64Config = GetUserInput("")
			if base64Config == "" {
				fmt.Println(get, "错误: 未输入 Base64 配置")
			}
		}
	}

	if base64Config == "" {
		fmt.Println(get, "警告：将使用无配置模式进行解析，由于不知道原始文件的长度，这可能会导致解析失败")
		fmt.Println(get, "请输入要生成的文件名")
		fileName := GetUserInput("")
		fmt.Println(get, "即将进入解码程序，请在目录下放置要解码的 .mp4 fec 文件，然后回车确定")
		GetUserInput("请按回车键继续...")
		Decode("", 0, nil)
		fmt.Println(get, "解码完成")
		// 查找生成的 .fec 文件
		fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			fmt.Println(en, "无法生成文件列表:", err)
			return
		}
		var cmdElement []string
		cmdElement = append(cmdElement, "-o")
		cmdElement = append(cmdElement, fileName)
		cmdElement = append(cmdElement, "-f")
		for _, fp := range fileDict {
			cmdElement = append(cmdElement, fp)
		}
		fmt.Println(get, "开始调用 zunfec")
		zunfecStartTime := time.Now()
		zunfecCmd := exec.Command("zunfec", cmdElement...)
		zunfecCmd.Dir = fileDir
		err = zunfecCmd.Run()
		if err != nil {
			fmt.Println("zunfecCmd 命令执行出错:", err)
			return
		}
		zunfecEndTime := time.Now()
		zunfecDuration := zunfecEndTime.Sub(zunfecStartTime)
		fmt.Println(get, "zunfec 调用完成，耗时:", zunfecDuration)
		fmt.Println(get, "获取完成")
		return
	} else {
		var fecFileConfig FecFileConfig
		fecFileConfigJson, err := base64.StdEncoding.DecodeString(base64Config)
		if err != nil {
			fmt.Println(get, "解析 Base64 配置失败:", err)
			return
		}
		err = json.Unmarshal(fecFileConfigJson, &fecFileConfig)
		if err != nil {
			fmt.Println(get, "解析 JSON 配置失败:", err)
			return
		}

		// 查找 .mp4 文件
		fileDict, err := GenerateFileDxDictionary(fileDir, ".mp4")
		if err != nil {
			fmt.Println(en, "无法生成文件列表:", err)
			return
		}

		//type VideoHashListMap struct {
		//	FilePath string            `json:"filePath"`
		//	HashList FecFileListConfig `json:"hashList"`
		//}
		//
		//videoHashMap := make(map[string]VideoHashListMap, 0)
		//for _, filePathF := range fileDict {
		//	hash := CalculateFileHash(filePathF)
		//	// 检查hash是否在配置中
		//	for _, fecFile := range fecFileConfig.FecFileList {
		//		if fecFile.VideoHash == hash {
		//			videoHashMap[hash] = VideoHashListMap{
		//				FilePath: filePathF,
		//				HashList: fecFile,
		//			}
		//			break
		//		}
		//	}
		//}

		// 修改文件名加上output前缀
		fecFileConfig.Name = "output_" + fecFileConfig.Name

		fmt.Println(get, "文件名:", fecFileConfig.Name)
		fmt.Println(get, "摘要:", fecFileConfig.Summary)
		fmt.Println(get, "分段长度:", fecFileConfig.SegmentLength)
		fmt.Println(get, "分段数量:", fecFileConfig.SegmentNumber)
		fmt.Println(get, "Hash:", fecFileConfig.Hash)
		fmt.Println(get, "在目录下找到以下匹配的 .mp4 文件:")
		for h, v := range fileDict {
			fmt.Println(get, strconv.Itoa(h)+":", "文件路径:", v)
		}

		fmt.Println(get, "是否使用配置默认的文件名:", fecFileConfig.Name, "？ [Y/n]")
		fileName := GetUserInput("")
		if fileName == "N" || fileName == "n" {
			fmt.Println(get, "请输入要生成的文件名")
			fileName = GetUserInput("")
			if fileName == "" {
				fmt.Println(get, "警告：您未输入任何内容，将使用默认文件名:", fecFileConfig.Name)
			} else {
				fecFileConfig.Name = fileName
				fmt.Println(get, "输出文件名修改为:", fileName)
			}
		}

		// 转换map[int]string 到 []string
		var fileDictList []string
		for _, v := range fileDict {
			fileDictList = append(fileDictList, v)
		}

		Decode("", fecFileConfig.SegmentLength, fileDictList)
		fmt.Println(get, "解码完成")
		// 查找生成的 .fec 文件
		fileDict, err = GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			fmt.Println(en, "无法生成文件列表:", err)
			return
		}
		var cmdElement []string
		cmdElement = append(cmdElement, "-o")
		cmdElement = append(cmdElement, fecFileConfig.Name)
		cmdElement = append(cmdElement, "-f")
		for _, fp := range fileDict {
			cmdElement = append(cmdElement, fp)
		}
		fmt.Println(get, "开始调用 zunfec")
		zunfecStartTime := time.Now()
		zunfecCmd := exec.Command("zunfec", cmdElement...)
		zunfecCmd.Dir = fileDir
		err = zunfecCmd.Run()
		if err != nil {
			fmt.Println("zunfecCmd 命令执行出错:", err)
			return
		}
		zunfecEndTime := time.Now()
		zunfecDuration := zunfecEndTime.Sub(zunfecStartTime)
		fmt.Println(get, "zunfec 调用完成，耗时:", zunfecDuration)

		fileDict, err = GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			fmt.Println(en, "无法生成文件列表:", err)
			return
		}
		if len(fileDict) != 0 {
			fmt.Println(add, "删除临时文件")
			for _, filePath := range fileDict {
				err = os.Remove(filePath)
				if err != nil {
					fmt.Println(add, "删除文件失败:", err)
					return
				}
			}
		}

		// 检查最终生成的文件是否与原始文件一致
		fmt.Println(get, "检查生成的文件是否与源文件一致")
		targetHash := CalculateFileHash(filepath.Join(fileDir, fecFileConfig.Name), defaultHashLength)
		if targetHash != fecFileConfig.Hash {
			fmt.Println(get, "警告: 生成的文件与源文件不一致:")
			fmt.Println(get, "源文件 Hash:", fecFileConfig.Hash)
			fmt.Println(get, "生成文件 Hash:", targetHash)
			fmt.Println(get, "文件解码失败")
		} else {
			fmt.Println(get, "生成的文件与源文件一致")
			fmt.Println(get, "源文件 Hash:", fecFileConfig.Hash)
			fmt.Println(get, "生成文件 Hash:", targetHash)
			fmt.Println(get, "文件成功解码")
		}
		fmt.Println(get, "获取完成")
		return
	}
}

func AutoRun() {
	fmt.Println("AutoRun: 使用 \"" + os.Args[0] + " help\" 查看帮助")
	fmt.Println("AutoRun: 请选择你要执行的操作:")
	fmt.Println("AutoRun:   1. 添加")
	fmt.Println("AutoRun:   2. 获取")
	fmt.Println("AutoRun:   3. 编码")
	fmt.Println("AutoRun:   4. 解码")
	fmt.Println("AutoRun:   5. 退出")
	for {
		fmt.Print("AutoRun: 请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("AutoRun: 错误: 请重新输入")
			continue
		}
		if input == "1" {
			clearScreen()
			Add()
			break
		} else if input == "2" {
			clearScreen()
			Get("")
			break
		} else if input == "3" {
			clearScreen()
			Encode("", encodeVideoSizeLevel, encodeOutputFPSLevel, encodeMaxSecondsLevel, encodeFFmpegModeLevel, false)
			break
		} else if input == "4" {
			clearScreen()
			Decode("", 0, nil)
			break
		} else if input == "5" {
			os.Exit(0)
		} else {
			fmt.Println("AutoRun: 错误: 无效的操作编号")
			continue
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [command] [options]\n", os.Args[0])
		fmt.Fprintln(os.Stdout, "Double-click to run: Start via automatic mode")
		fmt.Fprintln(os.Stdout, "\nCommands:")
		fmt.Fprintln(os.Stdout, "add\tUsing ffmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.")
		fmt.Fprintln(os.Stdout, "get\tUsing ffmpeg to decode .mp4 FEC video files into the original files.")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -b\tThe Base64 encoded JSON included message to provide decode")
		fmt.Fprintln(os.Stdout, "encode\tEncode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input fec file to encode")
		fmt.Fprintln(os.Stdout, " -s\tThe video size(default="+strconv.Itoa(encodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
		fmt.Fprintln(os.Stdout, " -p\tThe output video fps setting(default="+strconv.Itoa(encodeOutputFPSLevel)+"), 1-60")
		fmt.Fprintln(os.Stdout, " -l\tThe output video max segment length(seconds) setting(default="+strconv.Itoa(encodeMaxSecondsLevel)+"), 1-10^9")
		fmt.Fprintln(os.Stdout, " -m\tFFmpeg mode(default="+encodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
		fmt.Fprintln(os.Stdout, "decode\tDecode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input file to decode")
		fmt.Fprintln(os.Stdout, "help\tShow this help")
		flag.PrintDefaults()
	}
	encodeFlag := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeInput := encodeFlag.String("i", "", "The input fec file to encode")
	encodeQrcodeSize := encodeFlag.Int("s", encodeVideoSizeLevel, "The video size(default="+strconv.Itoa(encodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
	encodeOutputFPS := encodeFlag.Int("p", encodeOutputFPSLevel, "The output video fps setting(default="+strconv.Itoa(encodeOutputFPSLevel)+"), 1-60")
	encodeMaxSeconds := encodeFlag.Int("l", encodeMaxSecondsLevel, "The output video max segment length(seconds) setting(default="+strconv.Itoa(encodeMaxSecondsLevel)+"), 1-10^9")
	encodeFFmpegMode := encodeFlag.String("m", encodeFFmpegModeLevel, "FFmpeg mode(default="+encodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")

	decodeFlag := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeInputDir := decodeFlag.String("i", "", "The input dir include video segments to decode")

	addFlag := flag.NewFlagSet("add", flag.ExitOnError)

	getFlag := flag.NewFlagSet("get", flag.ExitOnError)
	getBase64Config := getFlag.String("b", "", "The Base64 encoded JSON included message to provide decode")

	if len(os.Args) < 2 {
		AutoRun()
		PressEnterToContinue()
		return
	}
	switch os.Args[1] {
	case "add":
		err := addFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(add, "参数解析错误")
			return
		}
		Add()
	case "get":
		err := getFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(get, "参数解析错误")
			return
		}
		Get(*getBase64Config)
	case "encode":
		err := encodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(en, "参数解析错误")
			return
		}
		Encode(*encodeInput, *encodeQrcodeSize, *encodeOutputFPS, *encodeMaxSeconds, *encodeFFmpegMode, false)
	case "decode":
		err := decodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(de, "参数解析错误")
			return
		}
		Decode(*decodeInputDir, 0, nil)
	case "help":
		flag.Usage()
		return
	case "-h":
		flag.Usage()
		return
	case "--help":
		flag.Usage()
		return
	default:
		fmt.Println("Unknown command:", os.Args[1])
		flag.Usage()
	}
}
