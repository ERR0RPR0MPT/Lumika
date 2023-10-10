package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/google/uuid"
	"github.com/klauspost/reedsolomon"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func AddGetTask(dirName string, decodeThread int, basestr string) {
	uuidd := uuid.New().String()
	dt := &common.GetTaskListData{
		UUID:      uuidd,
		TimeStamp: time.Now().Format("2006-01-02 15:04:05"),
		TaskInfo: &common.GetTaskInfo{
			DirName:      dirName,
			DecodeThread: decodeThread,
			BaseStr:      basestr,
		},
		Filename:     "",
		ProgressRate: 0,
		ProgressNum:  0,
		Duration:     "",
	}
	common.GetTaskList[uuidd] = dt
	common.GetTaskQueue <- dt
}

func GetTaskWorker(id int) {
	for task := range common.GetTaskQueue {
		allStartTime := time.Now()
		common.LogPrintf(task.UUID, "GetTaskWorker %d 处理编码任务：%v\n", id, task.UUID)
		_, exist := common.GetTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "GetTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		common.GetTaskList[task.UUID].Status = "正在执行"
		common.GetTaskList[task.UUID].StatusMsg = "正在执行"
		outputFileName, err := GetExec(filepath.Join(common.LumikaDecodePath, task.TaskInfo.DirName), task.TaskInfo.BaseStr, task.TaskInfo.DecodeThread, task.UUID)
		_, exist = common.GetTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "GetTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		if err != nil {
			common.LogPrintf(task.UUID, "GetTaskWorker %d 编码任务执行失败\n", id)
			common.GetTaskList[task.UUID].Status = "执行失败"
			common.GetTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		common.GetTaskList[task.UUID].Status = "已完成"
		common.GetTaskList[task.UUID].StatusMsg = "已完成"
		common.GetTaskList[task.UUID].Filename = outputFileName
		common.GetTaskList[task.UUID].ProgressNum = 100.0
		common.GetTaskList[task.UUID].Duration = fmt.Sprintf("%vs", int64(math.Floor(time.Now().Sub(allStartTime).Seconds())))
	}
}

func GetTaskWorkerInit() {
	common.GetTaskQueue = make(chan *common.GetTaskListData)
	common.GetTaskList = make(map[string]*common.GetTaskListData)
	if len(common.DatabaseVariable.GetTaskList) != 0 {
		common.GetTaskList = common.DatabaseVariable.GetTaskList
		for kp, kq := range common.GetTaskList {
			if kq.Status == "正在执行" {
				common.GetTaskList[kp].Status = "执行失败"
				common.GetTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				common.GetTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 GetTaskWorker 协程来处理任务
	for i := 0; i < common.VarSettingsVariable.DefaultTaskWorkerGoRoutines; i++ {
		go GetTaskWorker(i)
	}
}

func GetInput() {
	base64Config := ""
	var fecDirList []string

	// 读取 LumikaDecodePath 目录下所有的子目录
	dirList, err := GetSubDirectories(common.LumikaDecodePath)
	if err != nil {
		common.LogPrintln("", common.GetStr, common.ErStr, "无法获取子目录:", err)
		return
	}
	if len(dirList) == 0 {
		common.LogPrintln("", common.GetStr, common.ErStr, "没有找到子目录，请添加存放编码文件的目录")
		return
	}
	// 从子目录读取 Base64 配置文件，有配置文件的目录就放入 fecDirList
	for _, d := range dirList {
		if IsFileExistsInDir(d, common.LumikaConfigFileName) {
			fecDirList = append(fecDirList, d)
		}
	}
	if len(fecDirList) == 0 {
		common.LogPrintln("", common.GetStr, common.ErStr, "没有找到子目录下的索引配置，请添加索引来解码")
		return
	}
	common.LogPrintln("", common.GetStr, "找到存有索引配置的目录:")
	for i, d := range fecDirList {
		common.LogPrintln("", common.GetStr, strconv.Itoa(i+1)+":", d)
	}

	// 设置处理使用的线程数量
	common.LogPrintln("", common.GetStr, "请输入处理使用的线程数量。默认(CPU核心数量)：\""+strconv.Itoa(common.VarSettingsVariable.DefaultMaxThreads)+"\"")
	decodeThread, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.GetStr, "自动设置处理使用的线程数量为", common.VarSettingsVariable.DefaultMaxThreads)
		decodeThread = common.VarSettingsVariable.DefaultMaxThreads
	}
	if decodeThread <= 0 {
		common.LogPrintln("", common.GetStr, common.ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", common.VarSettingsVariable.DefaultMaxThreads)
		decodeThread = common.VarSettingsVariable.DefaultMaxThreads
	}

	// 遍历每一个子目录并运行
	for _, fileDir := range fecDirList {
		// 搜索子目录的 Base64 配置文件
		configBase64FilePath := SearchFileNameInDir(fileDir, common.LumikaConfigFileName)
		common.LogPrintln("", common.GetStr, "读取配置文件")
		// 读取文件
		configBase64Bytes, err := os.ReadFile(configBase64FilePath)
		if err != nil {
			common.LogPrintln("", common.GetStr, common.ErStr, "读取文件失败:", err)
			return
		}
		base64Config = string(configBase64Bytes)
		_, err = GetExec(fileDir, base64Config, decodeThread, "")
		if err != nil {
			common.LogPrintln("", common.GetStr, common.ErStr, "解码失败:", err)
			return
		}
	}
}

func GetExec(fileDir string, base64Config string, decodeThread int, UUID string) (string, error) {
	// 创建输出目录
	fileOutputDir := filepath.Join(common.LumikaDecodeOutputPath, filepath.Base(fileDir))
	if _, err := os.Stat(fileOutputDir); os.IsNotExist(err) {
		common.LogPrintln(UUID, common.DeStr, "创建输出目录:", fileOutputDir)
		err = os.Mkdir(fileOutputDir, 0755)
		if err != nil {
			common.LogPrintln(UUID, common.DeStr, common.ErStr, "创建输出目录失败:", err)
			return "", &common.CommonError{Msg: "创建输出目录失败:" + err.Error()}
		}
	}

	var fecFileConfig common.FecFileConfig
	fecFileConfigJson, err := base64.StdEncoding.DecodeString(base64Config)
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "解析 Base64 配置失败:", err)
		return "", &common.CommonError{Msg: "解析 Base64 配置失败:" + err.Error()}
	}
	err = json.Unmarshal(fecFileConfigJson, &fecFileConfig)
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "解析 JSON 配置失败:", err)
		return "", &common.CommonError{Msg: "解析 JSON 配置失败:" + err.Error()}
	}

	// 检测是否与当前版本匹配
	if fecFileConfig.Version != common.LumikaVersionNum {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "错误: 版本不匹配，无法进行解码。编码文件版本:", fecFileConfig.Version, "当前版本:", common.LumikaVersionNum)
		return "", &common.CommonError{Msg: "版本不匹配，无法进行解码。"}
	}

	// 查找 .mp4 文件
	fileDict, err := GenerateFileDxDictionary(fileDir, ".mp4")
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "无法生成文件列表:", err)
		return "", &common.CommonError{Msg: "无法生成文件列表:" + err.Error()}
	}

	common.LogPrintln(UUID, common.GetStr, "文件名:", fecFileConfig.Name)
	common.LogPrintln(UUID, common.GetStr, "摘要:", fecFileConfig.Summary)
	common.LogPrintln(UUID, common.GetStr, "分段长度:", fecFileConfig.SegmentLength)
	common.LogPrintln(UUID, common.GetStr, "分段数量:", fecFileConfig.M)
	common.LogPrintln(UUID, common.GetStr, "Hash:", fecFileConfig.Hash)
	common.LogPrintln(UUID, common.GetStr, "在目录下找到以下匹配的 .mp4 文件:")
	for h, v := range fileDict {
		common.LogPrintln(UUID, common.GetStr, strconv.Itoa(h)+":", "文件路径:", v)
	}

	// 转换map[int]string 到 []string
	var fileDictList []string
	for _, v := range fileDict {
		fileDictList = append(fileDictList, v)
	}

	common.LogPrintln(UUID, common.GetStr, "开始解码")
	err = Decode(fileDir, fecFileConfig.SegmentLength, fileDictList, fecFileConfig.MG, fecFileConfig.KG, decodeThread, UUID)
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "解码失败:", err)
		return "", err
	}
	common.LogPrintln(UUID, common.GetStr, "解码完成")

	// 查找生成的 .fec 文件
	fileDict, err = GenerateFileDxDictionary(fileOutputDir, ".fec")
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "无法生成文件列表:", err)
		return "", &common.CommonError{Msg: "无法生成文件列表:" + err.Error()}
	}

	// 遍历索引的 FecHashList
	findNum := 0
	fecFindFileList := make([]string, fecFileConfig.M)
	for fecIndex, fecHash := range fecFileConfig.FecHashList {
		// 遍历生成的 .fec 文件
		isFind := false
		for _, fecFilePath := range fileDict {
			// 检查hash是否在配置中
			if fecHash == CalculateFileHash(fecFilePath, common.DefaultHashLength) {
				fecFindFileList[fecIndex] = fecFilePath
				isFind = true
				break
			}
		}
		if !isFind {
			common.LogPrintln(UUID, common.GetStr, "警告：未找到匹配的 .fec 文件，Hash:", fecHash)
		} else {
			common.LogPrintln(UUID, common.GetStr, "找到匹配的 .fec 文件，Hash:", fecHash)
			findNum++
		}
	}
	common.LogPrintln(UUID, common.GetStr, "找到完整的 .fec 文件数量:", findNum)
	common.LogPrintln(UUID, common.GetStr, "未找到的文件数量:", fecFileConfig.M-findNum)
	common.LogPrintln(UUID, common.GetStr, "编码时生成的 .fec 文件数量(M):", fecFileConfig.M)
	common.LogPrintln(UUID, common.GetStr, "恢复所需最少的 .fec 文件数量(K):", fecFileConfig.K)
	if findNum >= fecFileConfig.K {
		common.LogPrintln(UUID, common.GetStr, "提示：可以成功恢复数据")
	} else {
		common.LogPrintln(UUID, common.GetStr, "警告：无法成功恢复数据，请按下回车键来确定")
		GetUserInput("请按回车键继续...")
	}

	// 生成原始文件
	common.LogPrintln(UUID, common.GetStr, "开始生成原始文件")
	zunfecStartTime := time.Now()
	enc, err := reedsolomon.New(fecFileConfig.K, fecFileConfig.M-fecFileConfig.K)
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "无法构建RS解码器:", err)
		return "", &common.CommonError{Msg: "无法构建RS解码器:" + err.Error()}
	}
	shards := make([][]byte, fecFileConfig.M)
	for i := range shards {
		if fecFindFileList[i] == "" {
			common.LogPrintln(UUID, common.GetStr, "Index:", i, ", 警告：未找到匹配的 .fec 文件")
			continue
		}
		common.LogPrintln(UUID, common.GetStr, "Index:", i, ", 读取文件:", fecFindFileList[i])
		shards[i], err = os.ReadFile(fecFindFileList[i])
		if err != nil {
			common.LogPrintln(UUID, common.GetStr, common.ErStr, "读取 .fec 文件时出错", err)
			shards[i] = nil
		}
	}
	// 校验数据
	ok, err := enc.Verify(shards)
	if ok {
		common.LogPrintln(UUID, common.GetStr, "数据完整，不需要恢复")
	} else {
		common.LogPrintln(UUID, common.GetStr, "数据不完整，准备恢复数据")
		err = enc.Reconstruct(shards)
		if err != nil {
			common.LogPrintln(UUID, common.GetStr, common.ErStr, "恢复失败 -", err)
			DeleteFecFiles(fileOutputDir)
			if UUID == "" {
				GetUserInput("请按回车键继续...")
			}
			return "", &common.CommonError{Msg: "恢复失败:" + err.Error()}
		}
		ok, err = enc.Verify(shards)
		if !ok {
			common.LogPrintln(UUID, common.GetStr, common.ErStr, "恢复失败，数据可能已损坏")
			DeleteFecFiles(fileOutputDir)
			if UUID == "" {
				GetUserInput("请按回车键继续...")
			}
			return "", &common.CommonError{Msg: "恢复失败，数据可能已损坏"}
		}
		if err != nil {
			common.LogPrintln(UUID, common.GetStr, common.ErStr, "恢复失败 -", err)
			DeleteFecFiles(fileOutputDir)
			if UUID == "" {
				GetUserInput("请按回车键继续...")
			}
			return "", &common.CommonError{Msg: "恢复失败:" + err.Error()}
		}
		common.LogPrintln(UUID, common.GetStr, "恢复成功")
	}
	common.LogPrintln(UUID, common.GetStr, "写入文件到:", fecFileConfig.Name)
	f, err := os.Create(filepath.Join(common.LumikaDecodeOutputPath, fecFileConfig.Name))
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "创建文件失败:", err)
		return "", &common.CommonError{Msg: "创建文件失败:" + err.Error()}
	}
	err = enc.Join(f, shards, len(shards[0])*fecFileConfig.K)
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "写入文件失败:", err)
		return "", &common.CommonError{Msg: "写入文件失败:" + err.Error()}
	}
	f.Close()
	err = TruncateFile(fecFileConfig.Length, filepath.Join(common.LumikaDecodeOutputPath, fecFileConfig.Name))
	if err != nil {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "截断解码文件失败:", err)
		return "", &common.CommonError{Msg: "截断解码文件失败:" + err.Error()}
	}
	zunfecEndTime := time.Now()
	zunfecDuration := zunfecEndTime.Sub(zunfecStartTime)
	common.LogPrintln(UUID, common.GetStr, "生成原始文件成功，耗时:", zunfecDuration)
	DeleteFecFiles(fileOutputDir)
	// 删除临时输出目录
	err = os.RemoveAll(fileOutputDir)
	if err != nil {
		common.LogPrintln(UUID, common.DeStr, common.ErStr, "删除临时输出目录失败:", err)
	}
	// 检查最终生成的文件是否与原始文件一致
	common.LogPrintln(UUID, common.GetStr, "检查生成的文件是否与源文件一致")
	targetHash := CalculateFileHash(filepath.Join(common.LumikaDecodeOutputPath, fecFileConfig.Name), common.DefaultHashLength)
	if targetHash != fecFileConfig.Hash {
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "警告: 生成的文件与源文件不一致")
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "源文件 Hash:", fecFileConfig.Hash)
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "生成文件 Hash:", targetHash)
		common.LogPrintln(UUID, common.GetStr, common.ErStr, "文件解码失败")
	} else {
		common.LogPrintln(UUID, common.GetStr, "生成的文件与源文件一致")
		common.LogPrintln(UUID, common.GetStr, "源文件 Hash:", fecFileConfig.Hash)
		common.LogPrintln(UUID, common.GetStr, "生成文件 Hash:", targetHash)
		common.LogPrintln(UUID, common.GetStr, "文件成功解码")
	}
	common.LogPrintln(UUID, common.GetStr, "获取完成")
	return fecFileConfig.Name, nil
}
