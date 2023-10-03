package utils

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
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
		LogPrint("", "clearScreen: 清屏失败:", err)
		return
	}
}

func ExtractForwardElements(slice [][]byte, t int) [][]byte {
	var result [][]byte
	for _, row := range slice {
		var newRow []byte
		for i, element := range row {
			if i >= t {
				newRow = append(newRow, element)
			}
		}
		result = append(result, newRow)
	}
	return result
}

func countNilElements(slice [][]byte) int {
	count := 0
	for _, row := range slice {
		if row == nil {
			count++
		}
	}
	return count
}

func MakeMaxByteSlice(data []byte) []byte {
	newSlice := make([]byte, len(data))
	copy(newSlice, data)
	return newSlice
}

func MakeMax2ByteSlice(data [][]byte, dataLength, MGValue int) [][]byte {
	// 创建新切片并设置最大长度
	newSlice := make([][]byte, MGValue)
	for i := range newSlice {
		newSlice[i] = make([]byte, dataLength)
	}
	// 将数据遍历赋值给新切片
	for i, row := range data {
		if row == nil {
			newSlice[i] = nil
			continue
		}
		for j, element := range row {
			newSlice[i][j] = element
		}
	}
	return newSlice
}

func IntToByteArray(n uint32) []byte {
	result := make([]byte, 4)
	result[0] = byte(n >> 24 & 0xFF)
	result[1] = byte(n >> 16 & 0xFF)
	result[2] = byte(n >> 8 & 0xFF)
	result[3] = byte(n & 0xFF)
	return result
}

func ByteArrayToInt(data []byte) uint32 {
	if len(data) > 4 {
		data = data[:4]
	}
	result := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
	return result
}

func RemoveDuplicates(slices [][]byte) [][]byte {
	seen := make(map[string]struct{}) // 用于记录已经出现过的元素
	// 遍历输入切片
	for _, slice := range slices {
		str := string(slice) // 将切片转换为字符串作为键

		// 如果元素不重复，则将其记录到 seen 中，并添加到结果切片中
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
		}
	}
	// 构建结果切片
	result := make([][]byte, 0, len(seen))
	for str := range seen {
		result = append(result, []byte(str))
	}
	return result
}

func ProcessSlices(slices [][]byte, MGValue int) [][]byte {
	slices = RemoveDuplicates(slices) // 去重
	// 自定义排序函数
	sort.Slice(slices, func(i, j int) bool {
		num1 := ByteArrayToInt(slices[i][:4]) // 获取第一个切片的前四个字节的 256 进制数
		num2 := ByteArrayToInt(slices[j][:4]) // 获取第二个切片的前四个字节的 256 进制数
		return num1 < num2
	})
	result := make([][]byte, MGValue) // 创建一个新切片来存储结果
	// 遍历输入切片
	for i := 0; i < len(slices); i++ {
		if len(slices[i]) < 4 {
			// 没有读取到索引数据，跳过
			continue
		}
		// 获取真实数据的索引
		dataIndex := ByteArrayToInt(slices[i][:4])
		result[dataIndex] = slices[i]
	}
	return result
}

// IsConsecutive 检查两个切片是否连续
func IsConsecutive(slice1, slice2 []byte) bool {
	if len(slice1) < 4 || len(slice2) < 4 {
		return false
	}
	num1 := ByteArrayToInt(slice1[:4]) // 获取第一个切片的前四个字节的 256 进制数
	num2 := ByteArrayToInt(slice2[:4]) // 获取第二个切片的前四个字节的 256 进制数
	return num2 == num1+1
}

func DeleteFecFiles(fileDir string) {
	// 是否删除.fec临时文件
	if DefaultDeleteFecFiles {
		fileDict, err := GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			LogPrint("", "DeleteFecFiles:", ErStr, "无法生成文件列表:", err)
			return
		}
		if len(fileDict) != 0 {
			LogPrint("", "DeleteFecFiles:", "删除临时文件")
			for _, filePath := range fileDict {
				err = os.Remove(filePath)
				if err != nil {
					LogPrint("", "DeleteFecFiles:", ErStr, "删除文件失败:", err)
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
		LogPrint("", "Data2Image:", ErStr, "警告: 数据过长，将进行截断")
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
// 类型：
// 0: 数据帧
// 1: 空白帧
// 2: 空白起始帧
// 3: 空白终止帧
func Image2Data(img image.Image) (dataR []byte, t int) {
	bounds := img.Bounds()
	size := bounds.Size().X
	dataLength := size * size / 8
	data := make([]byte, dataLength)
	// 遍历图像像素并提取数据
	// 检查是否为空白帧
	isBlank := true
	// 检查是否为空白起始帧
	isBlankStart := true
	// 检查是否为空白终止帧
	isBlankEnd := true
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
		if b != DefaultBlankByte {
			isBlank = false
		}
		if b != DefaultBlankStartByte {
			isBlankStart = false
		}
		if b != DefaultBlankEndByte {
			isBlankEnd = false
		}
		data[i] = b
	}
	if isBlank {
		return nil, 1
	}
	if isBlankStart {
		return nil, 2
	}
	if isBlankEnd {
		return nil, 3
	}
	return data, 0
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
		LogPrint("", "GetUserInput:", ErStr, "获取用户输入失败:", err)
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
			if strings.Contains(filepath.Base(path), "lumika") || strings.Contains(filepath.Base(path), "ffmpeg") || strings.Contains(filepath.Base(path), "ffprobe") {
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
			if strings.Contains(filepath.Base(path), "lumika") || strings.Contains(filepath.Base(path), "ffmpeg") || strings.Contains(filepath.Base(path), "ffprobe") {
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
		LogPrint("", "IsFileExistsInDir:", ErStr, "无法读取目录:", err)
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
		LogPrint("", "SearchFileNameInDir:", ErStr, "无法读取目录:", err)
		return ""
	}
	for _, file := range files {
		if strings.Contains(file.Name(), filename) {
			return filepath.Join(directory, file.Name())
		}
	}
	return ""
}

func ReplaceInvalidCharacters(input string, replacement rune) string {
	invalidChars := []rune{'\\', '/', ':', '*', '?', '"', '<', '>', '|'}
	validChars := make(map[rune]bool)

	// 替换非法字符为有效字符
	for _, char := range input {
		if unicode.IsControl(char) || char == replacement {
			continue
		}
		if ReplaceInvalidCharactersContains(invalidChars, char) {
			validChars[char] = true
		}
	}

	// 构建替换后的字符串
	var result strings.Builder
	for _, char := range input {
		if unicode.IsControl(char) || char == replacement {
			continue
		}
		if validChars[char] {
			result.WriteRune(replacement)
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

func ReplaceInvalidCharactersContains(slice []rune, char rune) bool {
	for _, c := range slice {
		if c == char {
			return true
		}
	}
	return false
}

func GetFileNameFromURL(urlString string) string {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return ""
	}

	fileName := path.Base(parsedURL.Path)
	return fileName
}

func GetDirectoryJSON(directoryPath string) ([]FileInfo, error) {
	fileList := make([]FileInfo, 0)
	files, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fileType := "file"
		if file.IsDir() {
			fileType = "dir"
		}
		fileList = append(fileList, FileInfo{
			Filename: file.Name(),
			Type:     fileType,
		})
	}
	return fileList, nil
}
