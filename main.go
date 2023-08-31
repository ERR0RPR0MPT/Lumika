package main

import (
	"bufio"
	"bytes"
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
	addMLevel             = 10
	addKLevel             = 7
	encodeVideoSizeLevel  = 32
	encodeOutputFPSLevel  = 24
	encodeFFmpegModeLevel = "medium"
	decodeFileLengthLevel = 0
)

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

//// TrimTrailingZeros 删除切片末尾的连续零字节
//func TrimTrailingZeros(data []byte) []byte {
//	// 从切片末尾开始向前遍历，找到第一个非零字节的索引
//	index := len(data) - 1
//	for index >= 0 && data[index] == 0 {
//		index--
//	}
//
//	// 如果找到了非零字节，则返回删除末尾连续零字节后的切片
//	if index >= 0 {
//		return data[:index+1]
//	}
//
//	// 如果切片中所有字节都是零，则返回空切片
//	return []byte{}
//}

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

//func CalculateFileHash(filePath string) (string, error) {
//	file, err := os.Open(filePath)
//	if err != nil {
//		return "", err
//	}
//	defer file.Close()
//	hash := sha256.New()
//	if _, err := io.Copy(hash, file); err != nil {
//		return "", err
//	}
//	hashValue := hash.Sum(nil)
//	hashString := hex.EncodeToString(hashValue)
//	return hashString, nil
//}

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

func GetUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入内容: ")
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

func Encode(fileDir string, videoSize int, outputFPS int, encodeFFmpegMode string) {
	if videoSize%8 != 0 {
		fmt.Println(en, "视频大小必须是8的倍数")
		return
	}

	// 当没有检测到videoFileDir时，自动匹配
	if fileDir == "" {
		fmt.Println(en, "自动使用程序所在目录作为输入目录")
		fd, err := os.Executable()
		if err != nil {
			fmt.Println(en, "获取程序所在目录失败:", err)
			return
		}
		fileDir = filepath.Dir(fd)
	}

	// 检查输入文件夹是否存在
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		fmt.Println(en, "输入文件夹不存在:", err)
		return
	}

	fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
	if err != nil {
		fmt.Println(en, "无法生成文件列表:", err)
		return
	}
	filePathList := make([]string, 0)
	for {
		if len(fileDict) == 0 {
			fmt.Println(en, "当前目录下没有.fec文件，请将需要编码的文件放到当前目录下")
			return
		}
		fmt.Println(en, "请选择需要编码的.fec文件，输入索引并回车来选择")
		fmt.Println(en, "如果需要编码当前目录下的所有.fec文件，请直接输入回车")
		for index := 0; index < len(fileDict); index++ {
			fmt.Println("Encode:", strconv.Itoa(index)+":", fileDict[index])
		}
		result := GetUserInput()
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
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Println(en, "无法打开文件:", err)
				return
			}

			outputFilePath := AddOutputToFileName(filePath, ".mp4")                    // 输出文件路径
			fileLength := len(fileData)                                                // 输入文件长度
			dataSliceLen := videoSize * videoSize / 8                                  // 每帧存储的有效数据
			allFrameNum := int(math.Ceil(float64(fileLength) / float64(dataSliceLen))) // 生成总帧数
			allSeconds := int(math.Ceil(float64(allFrameNum) / float64(outputFPS)))    // 总时长(秒)

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
			bar := pb.StartNew(fileLength)

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
					fmt.Printf("\nEncode: 构建帧 %d, 已构建数据 %d, 总数据 %d bghjmntyvf\n", i, fileNowLength, fileLength)
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
	fmt.Println(en, "所有选择的.fec文件已编码完成，程序结束")
}

func Decode(videoFileDir string, dataLength int) {
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

	fileDict, err := GenerateFileDxDictionary(videoFileDir, ".mp4")
	if err != nil {
		fmt.Println(de, "无法生成视频列表:", err)
		return
	}

	filePathList := make([]string, 0)
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
		result := GetUserInput()
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

			// 删除解码文件的末尾连续的零字节
			fmt.Println(de, "删除解码文件的末尾连续的零字节")
			err = RemoveTrailingZerosFromFile(outputFilePath)
			if err != nil {
				fmt.Println(de, "删除解码文件的末尾连续的零字节失败:", err)
				return
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

}

func Get() {

}

func AutoRun() {
	fmt.Println("AutoRun: 使用 \"" + os.Args[0] + " help\" 查看帮助")
	fmt.Println("AutoRun: 请选择你要执行的操作:")
	fmt.Println("AutoRun:   1. 编码")
	fmt.Println("AutoRun:   2. 解码")
	fmt.Println("AutoRun:   3. 退出")
	for {
		fmt.Print("AutoRun: 请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("AutoRun: 错误: 读取输入失败:", err)
			return
		}
		if input == "1" {
			clearScreen()
			Add()
			break
		} else if input == "2" {
			clearScreen()
			Get()
			break
		} else if input == "3" {
			clearScreen()
			Encode("", encodeVideoSizeLevel, encodeOutputFPSLevel, encodeFFmpegModeLevel)
			break
		} else if input == "4" {
			clearScreen()
			Decode("", 0)
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
		fmt.Fprintln(os.Stdout, "add\t使用 ffmpeg 将 zfec 冗余文件编码为看起来不那么有害的 .mp4 fec 视频文件")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input fec file to encode")
		fmt.Fprintln(os.Stdout, " -m\tthe total number of share files created (default "+strconv.Itoa(addMLevel)+")")
		fmt.Fprintln(os.Stdout, " -k\tthe number of share files required to reconstruct (default "+strconv.Itoa(addKLevel)+")")
		fmt.Fprintln(os.Stdout, " -s\tThe video size(default="+strconv.Itoa(encodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
		fmt.Fprintln(os.Stdout, " -p\tThe output video fps setting(default="+strconv.Itoa(encodeOutputFPSLevel)+"), 1-60")
		fmt.Fprintln(os.Stdout, " -m\tFFmpeg mode(default="+encodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
		fmt.Fprintln(os.Stdout, "get\t使用 ffmpeg 将 .mp4 fec 视频文件解码为原始文件")
		fmt.Fprintln(os.Stdout, "encode\tEncode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input fec file to encode")
		fmt.Fprintln(os.Stdout, " -s\tThe video size(default="+strconv.Itoa(encodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
		fmt.Fprintln(os.Stdout, " -p\tThe output video fps setting(default="+strconv.Itoa(encodeOutputFPSLevel)+"), 1-60")
		fmt.Fprintln(os.Stdout, " -m\tFFmpeg mode(default="+encodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
		fmt.Fprintln(os.Stdout, "decode\tDecode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input file to decode")
		fmt.Fprintln(os.Stdout, " -l\tThe source file length(default="+strconv.Itoa(decodeFileLengthLevel)+")")
		fmt.Fprintln(os.Stdout, "help\tShow this help")
		flag.PrintDefaults()
	}
	encodeFlag := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeInput := encodeFlag.String("i", "", "The input fec file to encode")
	encodeQrcodeSize := encodeFlag.Int("s", encodeVideoSizeLevel, "The video size(default="+strconv.Itoa(encodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
	encodeOutputFPS := encodeFlag.Int("p", encodeOutputFPSLevel, "The output video fps setting(default="+strconv.Itoa(encodeOutputFPSLevel)+"), 1-60")
	encodeFFmpegMode := encodeFlag.String("m", encodeFFmpegModeLevel, "FFmpeg mode(default="+encodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")

	decodeFlag := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeInputDir := decodeFlag.String("i", "", "The input dir include video segments to decode")
	decodeFileLength := decodeFlag.Int("l", 0, "The source file length(default="+strconv.Itoa(decodeFileLengthLevel)+")")

	if len(os.Args) < 2 {
		AutoRun()
		PressEnterToContinue()
		return
	}
	switch os.Args[1] {
	case "encode":
		err := encodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(en, "参数解析错误")
			return
		}
		Encode(*encodeInput, *encodeQrcodeSize, *encodeOutputFPS, *encodeFFmpegMode)
	case "decode":
		err := decodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(de, "参数解析错误")
			return
		}
		Decode(*decodeInputDir, *decodeFileLength)
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
