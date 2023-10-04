package utils

import (
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/klauspost/reedsolomon"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

func AddGetTask(dirName string, decodeThread int, basestr string) {
	uuidd := uuid.New().String()
	dt := &GetTaskListData{
		UUID:      uuidd,
		TimeStamp: time.Now().Format("2006-01-02 15:04:05"),
		TaskInfo: &GetTaskInfo{
			DirName:      dirName,
			DecodeThread: decodeThread,
			BaseStr:      basestr,
		},
		ProgressRate: 0,
		ProgressNum:  0,
	}
	GetTaskList = append(GetTaskList, dt)
	GetTaskQueue <- dt
}

func GetTaskWorker(id int) {
	for task := range GetTaskQueue {
		// 处理任务
		LogPrintf(task.UUID, "GetTaskWorker %d 处理编码任务：%v\n", id, task.UUID)
		i := 0
		for kp, kq := range GetTaskList {
			if kq.UUID == task.UUID {
				i = kp
				break
			}
		}
		GetTaskList[i].Status = "正在执行"
		GetTaskList[i].StatusMsg = "正在执行"
		err := GetExec(filepath.Join(LumikaDecodePath, task.TaskInfo.DirName), task.TaskInfo.BaseStr, task.TaskInfo.DecodeThread, task.UUID)
		if err != nil {
			LogPrintf(task.UUID, "GetTaskWorker %d 编码任务执行失败\n", id)
			GetTaskList[i].Status = "执行失败"
			GetTaskList[i].StatusMsg = err.Error()
			return
		}
		GetTaskList[i].Status = "已完成"
		GetTaskList[i].StatusMsg = "已完成"
		GetTaskList[i].ProgressNum = 100.0
	}
}

func GetTaskWorkerInit() {
	GetTaskQueue = make(chan *GetTaskListData)
	GetTaskList = make([]*GetTaskListData, 0)
	// 启动多个 GetTaskWorker 协程来处理任务
	for i := 0; i < DefaultTaskWorkerGoRoutines; i++ {
		go GetTaskWorker(i)
	}
}

func GetInput() {
	base64Config := ""
	var fecDirList []string

	// 读取 LumikaDecodePath 目录下所有的子目录
	dirList, err := GetSubDirectories(LumikaDecodePath)
	if err != nil {
		LogPrint("", GetStr, ErStr, "无法获取子目录:", err)
		return
	}
	if len(dirList) == 0 {
		LogPrint("", GetStr, ErStr, "没有找到子目录，请添加存放编码文件的目录")
		return
	}
	// 从子目录读取 Base64 配置文件，有配置文件的目录就放入 fecDirList
	for _, d := range dirList {
		if IsFileExistsInDir(d, LumikaConfigFileName) {
			fecDirList = append(fecDirList, d)
		}
	}
	if len(fecDirList) == 0 {
		LogPrint("", GetStr, ErStr, "没有找到子目录下的索引配置，请添加索引来解码")
		return
	}
	LogPrint("", GetStr, "找到存有索引配置的目录:")
	for i, d := range fecDirList {
		LogPrint("", GetStr, strconv.Itoa(i+1)+":", d)
	}

	// 设置处理使用的线程数量
	LogPrint("", GetStr, "请输入处理使用的线程数量。默认(CPU核心数量)：\""+strconv.Itoa(runtime.NumCPU())+"\"")
	decodeThread, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		LogPrint("", GetStr, "自动设置处理使用的线程数量为", runtime.NumCPU())
		decodeThread = runtime.NumCPU()
	}
	if decodeThread <= 0 {
		LogPrint("", GetStr, ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", runtime.NumCPU())
		decodeThread = runtime.NumCPU()
	}

	// 遍历每一个子目录并运行
	for _, fileDir := range fecDirList {
		// 搜索子目录的 Base64 配置文件
		configBase64FilePath := SearchFileNameInDir(fileDir, LumikaConfigFileName)
		LogPrint("", GetStr, "读取配置文件")
		// 读取文件
		configBase64Bytes, err := os.ReadFile(configBase64FilePath)
		if err != nil {
			LogPrint("", GetStr, ErStr, "读取文件失败:", err)
			return
		}
		base64Config = string(configBase64Bytes)
		err = GetExec(fileDir, base64Config, decodeThread, "")
		if err != nil {
			LogPrint("", GetStr, ErStr, "解码失败:", err)
			return
		}
	}
}

func GetExec(fileDir string, base64Config string, decodeThread int, UUID string) error {
	// 创建输出目录
	fileOutputDir := filepath.Join(LumikaDecodeOutputPath, filepath.Base(fileDir))
	if _, err := os.Stat(fileOutputDir); os.IsNotExist(err) {
		LogPrint(UUID, DeStr, "创建输出目录:", fileOutputDir)
		err = os.Mkdir(fileOutputDir, 0644)
		if err != nil {
			LogPrint(UUID, DeStr, ErStr, "创建输出目录失败:", err)
			return &CommonError{Msg: "创建输出目录失败:" + err.Error()}
		}
	}

	var fecFileConfig FecFileConfig
	fecFileConfigJson, err := base64.StdEncoding.DecodeString(base64Config)
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "解析 Base64 配置失败:", err)
		return &CommonError{Msg: "解析 Base64 配置失败:" + err.Error()}
	}
	err = json.Unmarshal(fecFileConfigJson, &fecFileConfig)
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "解析 JSON 配置失败:", err)
		return &CommonError{Msg: "解析 JSON 配置失败:" + err.Error()}
	}

	// 检测是否与当前版本匹配
	if fecFileConfig.Version != LumikaVersionNum {
		LogPrint(UUID, GetStr, ErStr, "错误: 版本不匹配，无法进行解码。编码文件版本:", fecFileConfig.Version, "当前版本:", LumikaVersionNum)
		return &CommonError{Msg: "版本不匹配，无法进行解码。"}
	}

	// 查找 .mp4 文件
	fileDict, err := GenerateFileDxDictionary(fileDir, ".mp4")
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "无法生成文件列表:", err)
		return &CommonError{Msg: "无法生成文件列表:" + err.Error()}
	}

	LogPrint(UUID, GetStr, "文件名:", fecFileConfig.Name)
	LogPrint(UUID, GetStr, "摘要:", fecFileConfig.Summary)
	LogPrint(UUID, GetStr, "分段长度:", fecFileConfig.SegmentLength)
	LogPrint(UUID, GetStr, "分段数量:", fecFileConfig.M)
	LogPrint(UUID, GetStr, "Hash:", fecFileConfig.Hash)
	LogPrint(UUID, GetStr, "在目录下找到以下匹配的 .mp4 文件:")
	for h, v := range fileDict {
		LogPrint(UUID, GetStr, strconv.Itoa(h)+":", "文件路径:", v)
	}

	// 转换map[int]string 到 []string
	var fileDictList []string
	for _, v := range fileDict {
		fileDictList = append(fileDictList, v)
	}

	LogPrint(UUID, GetStr, "开始解码")
	Decode(fileDir, fecFileConfig.SegmentLength, fileDictList, fecFileConfig.MG, fecFileConfig.KG, decodeThread, UUID)
	LogPrint(UUID, GetStr, "解码完成")

	// 查找生成的 .fec 文件
	fileDict, err = GenerateFileDxDictionary(fileOutputDir, ".fec")
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "无法生成文件列表:", err)
		return &CommonError{Msg: "无法生成文件列表:" + err.Error()}
	}

	// 遍历索引的 FecHashList
	findNum := 0
	fecFindFileList := make([]string, fecFileConfig.M)
	for fecIndex, fecHash := range fecFileConfig.FecHashList {
		// 遍历生成的 .fec 文件
		isFind := false
		for _, fecFilePath := range fileDict {
			// 检查hash是否在配置中
			if fecHash == CalculateFileHash(fecFilePath, DefaultHashLength) {
				fecFindFileList[fecIndex] = fecFilePath
				isFind = true
				break
			}
		}
		if !isFind {
			LogPrint(UUID, GetStr, "警告：未找到匹配的 .fec 文件，Hash:", fecHash)
		} else {
			LogPrint(UUID, GetStr, "找到匹配的 .fec 文件，Hash:", fecHash)
			findNum++
		}
	}
	LogPrint(UUID, GetStr, "找到完整的 .fec 文件数量:", findNum)
	LogPrint(UUID, GetStr, "未找到的文件数量:", fecFileConfig.M-findNum)
	LogPrint(UUID, GetStr, "编码时生成的 .fec 文件数量(M):", fecFileConfig.M)
	LogPrint(UUID, GetStr, "恢复所需最少的 .fec 文件数量(K):", fecFileConfig.K)
	if findNum >= fecFileConfig.K {
		LogPrint(UUID, GetStr, "提示：可以成功恢复数据")
	} else {
		LogPrint(UUID, GetStr, "警告：无法成功恢复数据，请按下回车键来确定")
		GetUserInput("请按回车键继续...")
	}

	// 生成原始文件
	LogPrint(UUID, GetStr, "开始生成原始文件")
	zunfecStartTime := time.Now()
	enc, err := reedsolomon.New(fecFileConfig.K, fecFileConfig.M-fecFileConfig.K)
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "无法构建RS解码器:", err)
		return &CommonError{Msg: "无法构建RS解码器:" + err.Error()}
	}
	shards := make([][]byte, fecFileConfig.M)
	for i := range shards {
		if fecFindFileList[i] == "" {
			LogPrint(UUID, GetStr, "Index:", i, ", 警告：未找到匹配的 .fec 文件")
			continue
		}
		LogPrint(UUID, GetStr, "Index:", i, ", 读取文件:", fecFindFileList[i])
		shards[i], err = os.ReadFile(fecFindFileList[i])
		if err != nil {
			LogPrint(UUID, GetStr, ErStr, "读取 .fec 文件时出错", err)
			shards[i] = nil
		}
	}
	// 校验数据
	ok, err := enc.Verify(shards)
	if ok {
		LogPrint(UUID, GetStr, "数据完整，不需要恢复")
	} else {
		LogPrint(UUID, GetStr, "数据不完整，准备恢复数据")
		err = enc.Reconstruct(shards)
		if err != nil {
			LogPrint(UUID, GetStr, ErStr, "恢复失败 -", err)
			DeleteFecFiles(fileOutputDir)
			if UUID == "" {
				GetUserInput("请按回车键继续...")
			}
			return &CommonError{Msg: "恢复失败:" + err.Error()}
		}
		ok, err = enc.Verify(shards)
		if !ok {
			LogPrint(UUID, GetStr, ErStr, "恢复失败，数据可能已损坏")
			DeleteFecFiles(fileOutputDir)
			if UUID == "" {
				GetUserInput("请按回车键继续...")
			}
			return &CommonError{Msg: "恢复失败，数据可能已损坏"}
		}
		if err != nil {
			LogPrint(UUID, GetStr, ErStr, "恢复失败 -", err)
			DeleteFecFiles(fileOutputDir)
			if UUID == "" {
				GetUserInput("请按回车键继续...")
			}
			return &CommonError{Msg: "恢复失败:" + err.Error()}
		}
		LogPrint(UUID, GetStr, "恢复成功")
	}
	LogPrint(UUID, GetStr, "写入文件到:", fecFileConfig.Name)
	f, err := os.Create(filepath.Join(LumikaDecodeOutputPath, fecFileConfig.Name))
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "创建文件失败:", err)
		return &CommonError{Msg: "创建文件失败:" + err.Error()}
	}
	err = enc.Join(f, shards, len(shards[0])*fecFileConfig.K)
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "写入文件失败:", err)
		return &CommonError{Msg: "写入文件失败:" + err.Error()}
	}
	f.Close()
	err = TruncateFile(fecFileConfig.Length, filepath.Join(LumikaDecodeOutputPath, fecFileConfig.Name))
	if err != nil {
		LogPrint(UUID, GetStr, ErStr, "截断解码文件失败:", err)
		return &CommonError{Msg: "截断解码文件失败:" + err.Error()}
	}
	zunfecEndTime := time.Now()
	zunfecDuration := zunfecEndTime.Sub(zunfecStartTime)
	LogPrint(UUID, GetStr, "生成原始文件成功，耗时:", zunfecDuration)
	DeleteFecFiles(fileOutputDir)
	// 检查最终生成的文件是否与原始文件一致
	LogPrint(UUID, GetStr, "检查生成的文件是否与源文件一致")
	targetHash := CalculateFileHash(filepath.Join(LumikaDecodeOutputPath, fecFileConfig.Name), DefaultHashLength)
	if targetHash != fecFileConfig.Hash {
		LogPrint(UUID, GetStr, ErStr, "警告: 生成的文件与源文件不一致")
		LogPrint(UUID, GetStr, ErStr, "源文件 Hash:", fecFileConfig.Hash)
		LogPrint(UUID, GetStr, ErStr, "生成文件 Hash:", targetHash)
		LogPrint(UUID, GetStr, ErStr, "文件解码失败")
	} else {
		LogPrint(UUID, GetStr, "生成的文件与源文件一致")
		LogPrint(UUID, GetStr, "源文件 Hash:", fecFileConfig.Hash)
		LogPrint(UUID, GetStr, "生成文件 Hash:", targetHash)
		LogPrint(UUID, GetStr, "文件成功解码")
	}
	LogPrint(UUID, GetStr, "获取完成")
	return nil
}
