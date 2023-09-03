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
	"github.com/klauspost/reedsolomon"
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
	ar                    = "AutoRun:"
	addMLevel             = 100
	addKLevel             = 90
	encodeVideoSizeLevel  = 32
	encodeOutputFPSLevel  = 24
	encodeMaxSecondsLevel = 86400
	encodeFFmpegModeLevel = "medium"
	defaultHashLength     = 7
	defaultBlankSeconds   = 3
	defaultBlankByte      = 85
	defaultDeleteFecFiles = true
)

type FecFileConfig struct {
	Name          string   `json:"n"`
	Summary       string   `json:"s"`
	Hash          string   `json:"h"`
	M             int      `json:"m"`
	K             int      `json:"k"`
	Length        int64    `json:"l"`
	SegmentLength int64    `json:"sl"`
	FecHashList   []string `json:"fhl"`
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
		fmt.Println("clearScreen: 清屏失败:", err)
		return
	}
}

func DeleteFecFiles(fileDir string) {
	// 是否删除.fec临时文件
	if defaultDeleteFecFiles {
		fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			fmt.Println("DeleteFecFiles: 无法生成文件列表:", err)
			return
		}
		if len(fileDict) != 0 {
			fmt.Println("DeleteFecFiles: 删除临时文件")
			for _, filePath := range fileDict {
				err = os.Remove(filePath)
				if err != nil {
					fmt.Println("DeleteFecFiles: 删除文件失败:", err)
					return
				}
			}
		}
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
	// 检查是否为空白帧
	isBlank := true
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
		if b != defaultBlankByte {
			isBlank = false
		}
		data[i] = b
	}
	if isBlank {
		return nil
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
		fmt.Println("GetUserInput: 获取用户输入失败:", err)
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

func GetSubDirectories(path string) ([]string, error) {
	var subdirectories []string
	files, err := os.ReadDir(path)
	if err != nil {
		return subdirectories, err
	}
	for _, file := range files {
		if file.IsDir() {
			subdirectoryPath := filepath.Join(path, file.Name())
			subdirectories = append(subdirectories, subdirectoryPath)
		}
	}
	return subdirectories, nil
}

func IsFileExistsInDir(directory, filename string) bool {
	files, err := os.ReadDir(directory)
	if err != nil {
		fmt.Println("IsFileExistsInDir: 无法读取目录:", err)
		return false
	}
	for _, file := range files {
		if strings.Contains(file.Name(), filename) {
			return true
		}
	}
	return false
}

func SearchFileNameInDir(directory, filename string) string {
	files, err := os.ReadDir(directory)
	if err != nil {
		fmt.Println("SearchFileNameInDir: 无法读取目录:", err)
		return ""
	}
	for _, file := range files {
		if strings.Contains(file.Name(), filename) {
			return filepath.Join(directory, file.Name())
		}
	}
	return ""
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
			fmt.Println(en, strconv.Itoa(index)+":", fileDict[index])
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
	maxGoroutines := runtime.NumCPU() // 最大同时运行的协程数量
	semaphore := make(chan struct{}, maxGoroutines)
	allStartTime := time.Now()

	// 遍历需要处理的文件列表
	for fileIndexNum, filePath := range filePathList {
		fmt.Println(en, "开始编码第", fileIndexNum+1, "个文件，路径:", filePath)
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

			FFmpegProcess := exec.Command("ffmpeg", FFmpegCmd...)
			stdin, err := FFmpegProcess.StdinPipe()
			if err != nil {
				fmt.Println(en, "无法创建 FFmpeg 的标准输入管道:", err)
				return
			}
			err = FFmpegProcess.Start()
			if err != nil {
				fmt.Println(en, "无法启动 FFmpeg 子进程:", err)
				return
			}

			// 为规避某些编码器会自动在视频的前后删除某些帧，导致解码失败，这里在视频的前后各添加defaultBlankSeconds秒的空白帧
			// 由于视频的前后各添加了defaultBlankSeconds秒的空白帧，所以总时长需要加上4秒
			for i := 0; i < outputFPS*defaultBlankSeconds; i++ {
				data := make([]byte, dataSliceLen)
				for j := 0; j < dataSliceLen; j++ {
					data[j] = defaultBlankByte
				}
				// 生成带空白数据的图像
				img := Data2Image(data, videoSize)
				imageBuffer := new(bytes.Buffer)
				err = png.Encode(imageBuffer, img)
				if err != nil {
					return
				}
				imageData := imageBuffer.Bytes()
				_, err = stdin.Write(imageData)
				if err != nil {
					fmt.Println(en, "无法写入帧数据到 FFmpeg:", err)
					return
				}
				imageBuffer = nil
				imageData = nil
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
				if i%30000 == 0 {
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
					fmt.Println(en, "无法写入帧数据到 FFmpeg:", err)
					return
				}
				imageBuffer = nil
				imageData = nil
			}
			bar.Finish()

			// 为规避某些编码器会自动在视频的前后删除某些帧，导致解码失败，这里在视频的前后各添加defaultBlankSeconds秒的空白帧
			for i := 0; i < outputFPS*defaultBlankSeconds; i++ {
				data := make([]byte, dataSliceLen)
				for j := 0; j < dataSliceLen; j++ {
					data[j] = defaultBlankByte
				}
				// 生成带空白数据的图像
				img := Data2Image(data, videoSize)
				imageBuffer := new(bytes.Buffer)
				err = png.Encode(imageBuffer, img)
				if err != nil {
					return
				}
				imageData := imageBuffer.Bytes()
				_, err = stdin.Write(imageData)
				if err != nil {
					fmt.Println(en, "无法写入帧数据到 FFmpeg:", err)
					return
				}
				imageBuffer = nil
				imageData = nil
			}

			// 关闭 FFmpeg 的标准输入管道，等待子进程完成
			err = stdin.Close()
			if err != nil {
				fmt.Println(en, "无法关闭 FFmpeg 的标准输入管道:", err)
				return
			}
			if err := FFmpegProcess.Wait(); err != nil {
				fmt.Println(en, "FFmpeg 子进程执行失败:", err)
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
				fmt.Println(de, "当前目录下没有.mp4文件，请将需要解码的视频文件放到当前目录下")
				return
			}
			fmt.Println(de, "请选择需要编码的.mp4文件，输入索引并回车来选择")
			fmt.Println(de, "如果需要编码当前目录下的所有.mp4文件，请直接输入回车")
			for index := 0; index < len(fileDict); index++ {
				fmt.Println("Encode:", strconv.Itoa(index)+":", fileDict[index])
			}
			result := GetUserInput("")
			if result == "" {
				fmt.Println(de, "注意：开始编码当前目录下的所有.mp4文件")
				for _, filePath := range fileDict {
					filePathList = append(filePathList, filePath)
				}
				break
			} else {
				index, err := strconv.Atoi(result)
				if err != nil {
					fmt.Println(de, "输入索引不是数字，请重新输入")
					continue
				}
				if index < 0 || index >= len(fileDict) {
					fmt.Println(de, "输入索引超出范围，请重新输入")
					continue
				}
				filePathList = append(filePathList, fileDict[index])
				break
			}
		}
	}

	var wg sync.WaitGroup
	maxGoroutines := runtime.NumCPU() // 最大同时运行的协程数量
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

			FFmpegCmd := []string{
				"ffmpeg",
				"-i", filePath,
				"-f", "image2pipe",
				"-pix_fmt", "rgb24",
				"-vcodec", "rawvideo",
				"-",
			}
			FFmpegProcess := exec.Command(FFmpegCmd[0], FFmpegCmd[1:]...)
			FFmpegStdout, err := FFmpegProcess.StdoutPipe()
			if err != nil {
				fmt.Println(de, "无法创建 FFmpeg 标准输出管道:", err)
				return
			}
			err = FFmpegProcess.Start()
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
				bar.SetCurrent(int64(i + 1))
				if i%30000 == 0 {
					fmt.Printf("\nDecode: 写入帧 %d 总帧 %d\n", i, frameCount)
				}
				img := RawDataToImage(rawData, videoWidth, videoHeight)
				data := Image2Data(img)
				if data == nil {
					i++
					continue
				}
				_, err = outputFile.Write(data)
				if err != nil {
					fmt.Println(de, "写入文件失败:", err)
					break
				}
				i++
			}
			bar.Finish()
			err = FFmpegStdout.Close()
			if err != nil {
				fmt.Println(de, "无法关闭 FFmpeg 标准输出管道:", err)
				return
			}
			err = FFmpegProcess.Wait()
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
				fmt.Println(de, "未提供原始文件的长度参数，默认删除解码文件的末尾连续的零字节来还原原始文件(无法还原尾部带零字节的分段文件)")
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
		fmt.Println(add, "获取程序所在目录失败:", err)
		return
	}
	fileDir := filepath.Dir(fd)

	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		fmt.Println(add, "输入文件夹不存在:", err)
		return
	}

	fmt.Println(add, "当前目录:", fileDir)

	fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
	if err != nil {
		fmt.Println(add, "无法生成文件列表:", err)
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
			fmt.Println(add, "注意：开始编码当前目录下的所有文件")
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

		// 计算文件长度
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fileSize := fileInfo.Size()

		// 开始生成 .fec 文件
		fmt.Println(add, "开始生成 .fec 文件")
		zfecStartTime := time.Now()
		enc, err := reedsolomon.New(defaultK, defaultM-defaultK)
		if err != nil {
			fmt.Println(add, "创建 reedsolomon 对象失败:", err)
			return
		}
		b, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println(add, "读取文件失败:", err)
			return
		}
		shards, err := enc.Split(b)
		if err != nil {
			fmt.Println(add, "分割文件失败:", err)
			return
		}
		err = enc.Encode(shards)
		if err != nil {
			fmt.Println(add, "编码文件失败:", err)
			return
		}
		// 生成 fecHashList
		fecHashList := make([]string, len(shards))
		for i, shard := range shards {
			outfn := fmt.Sprintf("%s.%d_%d.fec", filepath.Base(filePath), i, len(shards))
			outfnPath := filepath.Join(defaultOutputDir, outfn)
			fmt.Println(add, "写入 .fec 文件:", outfn)
			err = os.WriteFile(outfnPath, shard, 0644)
			if err != nil {
				fmt.Println(add, ".fec 文件写入失败:", err)
				return
			}
			fileHash := CalculateFileHash(outfnPath, defaultHashLength)
			fecHashList[i] = fileHash
		}
		zfecEndTime := time.Now()
		zfecDuration := zfecEndTime.Sub(zfecStartTime)
		fmt.Println(add, ".fec 文件生成完成，耗时:", zfecDuration)

		fmt.Println(add, "开始进行编码")
		segmentLength := Encode(defaultOutputDir, encodeVideoSizeLevel, encodeOutputFPSLevel, encodeMaxSecondsLevel, encodeFFmpegModeLevel, true)

		fmt.Println(add, "编码完成，开始生成配置")
		fecFileConfig := FecFileConfig{
			Name:          defaultFileName,
			Summary:       defaultSummary,
			Hash:          CalculateFileHash(filePath, defaultHashLength),
			M:             defaultM,
			K:             defaultK,
			Length:        fileSize,
			SegmentLength: segmentLength,
			FecHashList:   fecHashList,
		}
		fecFileConfigJson, err := json.Marshal(fecFileConfig)
		if err != nil {
			fmt.Println(add, "生成 JSON 配置失败:", err)
			return
		}
		// 转换为 Base64
		fecFileConfigBase64 := base64.StdEncoding.EncodeToString(fecFileConfigJson)
		fecFileConfigFilePath := filepath.Join(fileDir, "lumika_config_"+strings.ReplaceAll(defaultFileName, ".", "_")+".txt")
		fmt.Println(add, "Base64 配置生成完成，开始写入文件:", fecFileConfigFilePath)
		err = os.WriteFile(fecFileConfigFilePath, []byte(fecFileConfigBase64), 0644)
		if err != nil {
			fmt.Println(add, "写入文件失败:", err)
			return
		}
		fmt.Println(add, "写入配置成功")
		DeleteFecFiles(fileDir)
		fmt.Println(add, "Base64 配置文件已生成，路径:", fecFileConfigFilePath)
		fmt.Println(add, "Base64:", fecFileConfigBase64)
		fmt.Println(add, "请将生成的 .mp4 fec 视频文件和 Base64 配置分享或发送给你的好友，对方可使用 \"get\" 子命令来获取文件")
		fmt.Println(add, "添加完成")
	}
}

func Get() {
	base64Config := ""
	ep, err := os.Executable()
	if err != nil {
		fmt.Println(get, "无法获取运行目录:", err)
		return
	}
	epPath := filepath.Dir(ep)

	// 选择执行模式
	var fecDirList []string
	for {
		fmt.Println(get, "请选择执行模式(默认为1):")
		fmt.Println(get, "1. 读取本目录下所有的子目录并从子目录读取 Base64 配置文件(适用于解码多个文件)")
		fmt.Println(get, "2. 从本目录下读取 Base64 配置文件(适用于解码单个文件)")
		result := GetUserInput("")
		if result == "1" || result == "" {
			// 读取本目录下所有的子目录
			dirList, err := GetSubDirectories(epPath)
			if err != nil {
				fmt.Println(get, "无法获取子目录:", err)
				return
			}
			if len(dirList) == 0 {
				fmt.Println(get, "没有找到子目录，请添加存放编码文件的目录")
				return
			}
			// 从子目录读取 Base64 配置文件，有配置文件的目录就放入 fecDirList
			for _, d := range dirList {
				if IsFileExistsInDir(d, "lumika_config") {
					fecDirList = append(fecDirList, d)
				}
			}
			if len(fecDirList) == 0 {
				fmt.Println(get, "没有找到子目录下的索引配置，请添加索引来解码")
				return
			}
			fmt.Println(get, "找到存有索引配置的目录:")
			for i, d := range fecDirList {
				fmt.Println(get, strconv.Itoa(i+1)+":", d)
			}
			break
		} else if result == "2" {
			// 从本目录读取 Base64 配置文件
			if IsFileExistsInDir(epPath, "lumika_config") {
				fecDirList = append(fecDirList, epPath)
			} else {
				fmt.Println(get, "没有找到本目录下的索引配置，请添加索引来解码")
				return
			}
			break
		} else {
			fmt.Println(get, "无效输入，请重新输入")
			continue
		}
	}

	// 遍历每一个子目录并运行
	for _, fileDir := range fecDirList {
		// 搜索子目录的 Base64 配置文件
		configBase64FilePath := SearchFileNameInDir(fileDir, "lumika_config")
		fmt.Println(get, "读取配置文件")
		// 读取文件
		configBase64Bytes, err := os.ReadFile(configBase64FilePath)
		if err != nil {
			fmt.Println(get, "读取文件失败:", err)
			return
		}
		base64Config = string(configBase64Bytes)

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
			fmt.Println(get, "无法生成文件列表:", err)
			return
		}

		// 修改文件名加上output前缀
		fecFileConfig.Name = "output_" + fecFileConfig.Name

		fmt.Println(get, "文件名:", fecFileConfig.Name)
		fmt.Println(get, "摘要:", fecFileConfig.Summary)
		fmt.Println(get, "分段长度:", fecFileConfig.SegmentLength)
		fmt.Println(get, "分段数量:", fecFileConfig.M)
		fmt.Println(get, "Hash:", fecFileConfig.Hash)
		fmt.Println(get, "在目录下找到以下匹配的 .mp4 文件:")
		for h, v := range fileDict {
			fmt.Println(get, strconv.Itoa(h)+":", "文件路径:", v)
		}

		// 转换map[int]string 到 []string
		var fileDictList []string
		for _, v := range fileDict {
			fileDictList = append(fileDictList, v)
		}

		fmt.Println(get, "开始解码")
		Decode(fileDir, fecFileConfig.SegmentLength, fileDictList)
		fmt.Println(get, "解码完成")

		// 查找生成的 .fec 文件
		fileDict, err = GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			fmt.Println(get, "无法生成文件列表:", err)
			return
		}

		// 遍历索引的 FecHashList
		findNum := 0
		fecFindFileList := make([]string, fecFileConfig.M)
		for fecIndex, fecHash := range fecFileConfig.FecHashList {
			// 遍历生成的 .fec 文件
			isFind := false
			for _, fecFilePath := range fileDict {
				// 检查hash是否在配置中
				if fecHash == CalculateFileHash(fecFilePath, defaultHashLength) {
					fecFindFileList[fecIndex] = fecFilePath
					isFind = true
					break
				}
			}
			if !isFind {
				fmt.Println(get, "警告：未找到匹配的 .fec 文件，Hash:", fecHash)
			} else {
				fmt.Println(get, "找到匹配的 .fec 文件，Hash:", fecHash)
				findNum++
			}
		}
		fmt.Println(get, "找到完整的 .fec 文件数量:", findNum)
		fmt.Println(get, "未找到的文件数量:", fecFileConfig.M-findNum)
		fmt.Println(get, "编码时生成的 .fec 文件数量(M):", fecFileConfig.M)
		fmt.Println(get, "恢复所需最少的 .fec 文件数量(K):", fecFileConfig.K)
		if findNum >= fecFileConfig.K {
			fmt.Println(get, "提示：可以成功恢复数据")
		} else {
			fmt.Println(get, "警告：无法成功恢复数据，请按下回车键来确定")
			GetUserInput("请按回车键继续...")
		}

		// 生成原始文件
		fmt.Println(get, "开始生成原始文件")
		zunfecStartTime := time.Now()
		enc, err := reedsolomon.New(fecFileConfig.K, fecFileConfig.M-fecFileConfig.K)
		if err != nil {
			fmt.Println(get, "无法构建 reedsolomon 解码器:", err)
			return
		}
		shards := make([][]byte, fecFileConfig.M)
		for i := range shards {
			if fecFindFileList[i] == "" {
				fmt.Println(get, "Index:", i, ", 警告：未找到匹配的 .fec 文件")
				continue
			}
			fmt.Println(get, "Index:", i, ", 读取文件:", fecFindFileList[i])
			shards[i], err = os.ReadFile(fecFindFileList[i])
			if err != nil {
				fmt.Println(get, "读取 .fec 文件时出错", err)
				shards[i] = nil
			}
		}
		// 校验数据
		ok, err := enc.Verify(shards)
		if ok {
			fmt.Println(get, "数据完整，不需要恢复")
		} else {
			fmt.Println(get, "数据不完整，准备恢复数据")
			err = enc.Reconstruct(shards)
			if err != nil {
				fmt.Println(get, "恢复失败 -", err)
				DeleteFecFiles(fileDir)
				GetUserInput("请按回车键继续...")
				return
			}
			ok, err = enc.Verify(shards)
			if !ok {
				fmt.Println(get, "恢复失败，数据可能已损坏")
				DeleteFecFiles(fileDir)
				GetUserInput("请按回车键继续...")
				return
			}
			if err != nil {
				fmt.Println(get, "恢复失败 -", err)
				DeleteFecFiles(fileDir)
				GetUserInput("请按回车键继续...")
				return
			}
			fmt.Println(get, "恢复成功")
		}
		fmt.Println(get, "写入文件到:", fecFileConfig.Name)
		f, err := os.Create(fecFileConfig.Name)
		if err != nil {
			fmt.Println(get, "创建文件失败:", err)
			return
		}
		err = enc.Join(f, shards, len(shards[0])*fecFileConfig.K)
		if err != nil {
			fmt.Println(get, "写入文件失败:", err)
			return
		}
		f.Close()
		err = TruncateFile(fecFileConfig.Length, filepath.Join(epPath, fecFileConfig.Name))
		if err != nil {
			fmt.Println(get, "截断解码文件失败:", err)
			return
		}
		zunfecEndTime := time.Now()
		zunfecDuration := zunfecEndTime.Sub(zunfecStartTime)
		fmt.Println(get, "生成原始文件成功，耗时:", zunfecDuration)
		DeleteFecFiles(fileDir)
		// 检查最终生成的文件是否与原始文件一致
		fmt.Println(get, "检查生成的文件是否与源文件一致")
		targetHash := CalculateFileHash(filepath.Join(epPath, fecFileConfig.Name), defaultHashLength)
		if targetHash != fecFileConfig.Hash {
			fmt.Println(get, "警告: 生成的文件与源文件不一致")
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
	}
}

func AutoRun() {
	fmt.Println(ar, "使用 \""+os.Args[0]+" help\" 查看帮助")
	fmt.Println(ar, "请选择你要执行的操作:")
	fmt.Println(ar, "  1. 添加")
	fmt.Println(ar, "  2. 获取")
	fmt.Println(ar, "  3. 编码")
	fmt.Println(ar, "  4. 解码")
	fmt.Println(ar, "  5. 退出")
	for {
		fmt.Print(ar, "请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println(ar, "错误: 请重新输入")
			continue
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
			Encode("", encodeVideoSizeLevel, encodeOutputFPSLevel, encodeMaxSecondsLevel, encodeFFmpegModeLevel, false)
			break
		} else if input == "4" {
			clearScreen()
			Decode("", 0, nil)
			break
		} else if input == "5" {
			os.Exit(0)
		} else {
			fmt.Println(ar, "错误: 无效的操作编号")
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
		fmt.Fprintln(os.Stdout, "add\tUsing FFmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.")
		fmt.Fprintln(os.Stdout, "get\tUsing FFmpeg to decode .mp4 FEC video files into the original files.")
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
		Get()
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
