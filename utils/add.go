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
	"strings"
	"time"
)

func AddAddTask(fileNameList []string, defaultM int, defaultK int, MGValue int, KGValue int, videoSize int, outputFPS int, encodeMaxSeconds int, encodeThread int, encodeFFmpegMode string, defaultSummary string) {
	uuidd := uuid.New().String()
	dt := &common.AddTaskListData{
		UUID:      uuidd,
		TimeStamp: time.Now().Format("2006-01-02 15:04:05"),
		TaskInfo: &common.AddTaskInfo{
			FileNameList:     fileNameList,
			DefaultM:         defaultM,
			DefaultK:         defaultK,
			MGValue:          MGValue,
			KGValue:          KGValue,
			VideoSize:        videoSize,
			OutputFPS:        outputFPS,
			EncodeMaxSeconds: encodeMaxSeconds,
			EncodeThread:     encodeThread,
			EncodeFFmpegMode: encodeFFmpegMode,
			DefaultSummary:   defaultSummary,
		},
		ProgressRate: 0,
		Duration:     "",
	}
	common.AddTaskList[uuidd] = dt
	common.AddTaskQueue <- dt
}

func AddTaskWorker(id int) {
	for task := range common.AddTaskQueue {
		allStartTime := time.Now()
		common.LogPrintf(task.UUID, "AddTaskWorker %d 处理编码任务：%v\n", id, task.UUID)
		_, exist := common.AddTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "AddTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		common.AddTaskList[task.UUID].Status = "正在执行"
		common.AddTaskList[task.UUID].StatusMsg = "正在执行"
		err := AddExec(task.TaskInfo.FileNameList, task.TaskInfo.DefaultM, task.TaskInfo.DefaultK, task.TaskInfo.MGValue, task.TaskInfo.KGValue, task.TaskInfo.VideoSize, task.TaskInfo.OutputFPS, task.TaskInfo.EncodeMaxSeconds, task.TaskInfo.EncodeThread, task.TaskInfo.EncodeFFmpegMode, task.TaskInfo.DefaultSummary, task.UUID)
		_, exist = common.AddTaskList[task.UUID]
		if !exist {
			common.LogPrintf(task.UUID, "AddTaskWorker %d 编码任务被用户删除\n", id)
			continue
		}
		if err != nil {
			common.LogPrintf(task.UUID, "AddTaskWorker %d 编码任务执行失败\n", id)
			common.AddTaskList[task.UUID].Status = "执行失败"
			common.AddTaskList[task.UUID].StatusMsg = err.Error()
			continue
		}
		common.AddTaskList[task.UUID].Status = "已完成"
		common.AddTaskList[task.UUID].StatusMsg = "已完成"
		common.AddTaskList[task.UUID].ProgressNum = 100.0
		common.AddTaskList[task.UUID].Duration = fmt.Sprintf("%vs", int64(math.Floor(time.Now().Sub(allStartTime).Seconds())))
	}
}

func AddTaskWorkerInit() {
	common.AddTaskQueue = make(chan *common.AddTaskListData)
	common.AddTaskList = make(map[string]*common.AddTaskListData)
	if len(common.DatabaseVariable.AddTaskList) != 0 {
		common.AddTaskList = common.DatabaseVariable.AddTaskList
		for kp, kq := range common.AddTaskList {
			if kq.Status == "正在执行" {
				common.AddTaskList[kp].Status = "执行失败"
				common.AddTaskList[kp].StatusMsg = "任务执行时服务器后端被终止，无法继续执行任务"
				common.AddTaskList[kp].ProgressNum = 0.0
			}
		}
	}
	// 启动多个 AddTaskWorker 协程来处理任务
	for i := 0; i < common.VarSettingsVariable.DefaultTaskWorkerGoRoutines; i++ {
		go AddTaskWorker(i)
	}
}

func AddInput() {
	common.LogPrintln("", common.AddStr, "使用 \""+os.Args[0]+" help\" 查看帮助")

	fileDir := common.LumikaWorkDirPath
	fileEncodeDir := common.LumikaEncodePath

	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		common.LogPrintln("", common.AddStr, common.ErStr, "输入文件夹不存在:", err)
		return
	}

	common.LogPrintln("", common.AddStr, "当前编码目录:", fileDir)

	fileDict, err := GenerateFileDxDictionary(fileEncodeDir, ".fec")
	if err != nil {
		common.LogPrintln("", common.AddStr, common.ErStr, "无法生成文件列表:", err)
		return
	}

	if len(fileDict) != 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误：检测到目录下存在 .fec 文件，请先删除 .fec 文件再进行添加")
		return
	}

	// 设置默认的文件名
	fileDict, err = GenerateFileDictionary(fileDir)
	if err != nil {
		common.LogPrintln("", common.AddStr, common.ErStr, "无法生成文件列表:", err)
		return
	}
	fileNameList := make([]string, 0)
	for {
		if len(fileDict) == 0 {
			common.LogPrintln("", common.AddStr, common.ErStr, "当前目录下没有文件，请将需要编码的文件放到当前目录下")
			return
		}
		common.LogPrintln("", common.AddStr, "请选择需要编码的文件，输入索引并回车来选择")
		common.LogPrintln("", common.AddStr, "如果需要编码当前目录下的所有文件，请直接输入回车")
		for index := 0; index < len(fileDict); index++ {
			common.LogPrintln("", "Encode:", strconv.Itoa(index)+":", fileDict[index])
		}
		result := GetUserInput("")
		if result == "" {
			common.LogPrintln("", common.AddStr, "注意：开始编码当前目录下的所有文件")
			for _, filePath := range fileDict {
				fileNameList = append(fileNameList, filePath)
			}
			break
		} else {
			index, err := strconv.Atoi(result)
			if err != nil {
				common.LogPrintln("", common.AddStr, common.ErStr, "输入索引不是数字，请重新输入")
				continue
			}
			if index < 0 || index >= len(fileDict) {
				common.LogPrintln("", common.AddStr, common.ErStr, "输入索引超出范围，请重新输入")
				continue
			}
			fileNameList = append(fileNameList, fileDict[index])
			break
		}
	}

	// 设置M的值
	common.LogPrintln("", common.AddStr, "请输入 M 的值(0<=M<=256)，M 为最终生成的总切片数量。默认：\""+strconv.Itoa(common.AddMLevel)+"\"")
	defaultM, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置 M = "+strconv.Itoa(common.AddMLevel))
		defaultM = common.AddMLevel
	}
	if defaultM == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: M 的值不能为 0，自动设置 M = "+strconv.Itoa(common.AddMLevel))
		defaultM = common.AddMLevel
	}

	// 设置K的值
	common.LogPrintln("", common.AddStr, "请输入 K 的值(0<=K<=256)，K 为恢复原始文件所需的最少完整切片数量。默认：\""+strconv.Itoa(common.AddKLevel)+"\"")
	defaultK, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置 K = "+strconv.Itoa(common.AddKLevel))
		defaultK = common.AddKLevel
	}
	if defaultK == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: K 的值不能为 0，自动设置 K = "+strconv.Itoa(common.AddKLevel))
		defaultK = common.AddKLevel
	}

	if defaultK > defaultM {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: K 的值不能大于 M 的值，自动设置 K = M = "+strconv.Itoa(defaultM))
		defaultK = defaultM
	}

	// 设置MG的值
	common.LogPrintln("", common.AddStr, "请输入 MG 的值(0<=MG<=256)，MG 为帧数据的总切片数量。默认：\""+strconv.Itoa(common.AddMGLevel)+"\"")
	MGValue, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置 G = "+strconv.Itoa(common.AddMGLevel))
		MGValue = common.AddMGLevel
	}
	if MGValue == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: G 的值不能为 0，自动设置 G = "+strconv.Itoa(common.AddMGLevel))
		MGValue = common.AddMGLevel
	}

	// 设置KG的值
	common.LogPrintln("", common.AddStr, "请输入 KG 的值(0<=KG<=256)，KG 为恢复帧数据所需的最少完整切片数量。默认：\""+strconv.Itoa(common.AddKGLevel)+"\"")
	KGValue, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置 G = "+strconv.Itoa(common.AddKGLevel))
		KGValue = common.AddKGLevel
	}
	if KGValue == 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: G 的值不能为 0，自动设置 G = "+strconv.Itoa(common.AddKGLevel))
		KGValue = common.AddKGLevel
	}

	if KGValue > MGValue {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: KG 的值不能大于 MG 的值，自动设置 KG = MG = "+strconv.Itoa(MGValue))
		KGValue = MGValue
	}

	// 设置默认的分辨率大小
	common.LogPrintln("", common.AddStr, "请输入分辨率大小，例如输入32则分辨率为32x32。默认：\""+strconv.Itoa(common.EncodeVideoSizeLevel)+"\"")
	videoSize, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置分辨率大小为", common.EncodeVideoSizeLevel)
		videoSize = common.EncodeVideoSizeLevel
	}
	if videoSize <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 分辨率大小不能小于等于 0，自动设置分辨率大小为", common.EncodeVideoSizeLevel)
		videoSize = common.EncodeVideoSizeLevel
	}

	// 设置默认的帧率大小
	common.LogPrintln("", common.AddStr, "请输入帧率大小。默认：\""+strconv.Itoa(common.EncodeOutputFPSLevel)+"\"")
	outputFPS, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置帧率大小为", common.EncodeOutputFPSLevel)
		outputFPS = common.EncodeOutputFPSLevel
	}
	if outputFPS <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 帧率大小不能小于等于 0，自动设置帧率大小为", common.EncodeOutputFPSLevel)
		outputFPS = common.EncodeOutputFPSLevel
	}

	// 设置默认最大生成的视频长度限制
	common.LogPrintln("", common.AddStr, "请输入最大生成的视频长度限制(单位:秒)，默认：\""+strconv.Itoa(common.EncodeMaxSecondsLevel)+"\"")
	encodeMaxSeconds, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置最大生成的视频长度限制为", common.EncodeMaxSecondsLevel, "秒")
		encodeMaxSeconds = common.EncodeMaxSecondsLevel
	}
	if encodeMaxSeconds <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 最大生成的视频长度限制不能小于等于 0，自动设置最大生成的视频长度限制为", common.EncodeMaxSecondsLevel)
		encodeMaxSeconds = common.EncodeMaxSecondsLevel
	}

	// 设置默认使用的 FFmpeg 预设模式
	common.LogPrintln("", common.AddStr, "请输入使用的 FFmpeg 预设模式，例如：\"ultrafast\"。默认：\""+common.EncodeFFmpegModeLevel+"\"")
	encodeFFmpegMode := GetUserInput("")
	if encodeFFmpegMode == "" {
		common.LogPrintln("", common.AddStr, "自动设置使用的 FFmpeg 预设模式为", common.EncodeFFmpegModeLevel)
		encodeFFmpegMode = common.EncodeFFmpegModeLevel
	}

	// 设置处理使用的线程数量
	common.LogPrintln("", common.AddStr, "请输入处理使用的线程数量。默认(CPU核心数量)：\""+strconv.Itoa(common.VarSettingsVariable.DefaultMaxThreads)+"\"")
	encodeThread, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		common.LogPrintln("", common.AddStr, "自动设置处理使用的线程数量为", common.VarSettingsVariable.DefaultMaxThreads)
		encodeThread = common.VarSettingsVariable.DefaultMaxThreads
	}
	if encodeThread <= 0 {
		common.LogPrintln("", common.AddStr, common.ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", common.VarSettingsVariable.DefaultMaxThreads)
		encodeThread = common.VarSettingsVariable.DefaultMaxThreads
	}

	// 设置默认的摘要
	common.LogPrintln("", common.AddStr, "请输入摘要，可以作为文件内容的简介。例如：\"这是一个相册的压缩包\"")
	defaultSummary := GetUserInput("")

	err = AddExec(fileNameList, defaultM, defaultK, MGValue, KGValue, videoSize, outputFPS, encodeMaxSeconds, encodeThread, encodeFFmpegMode, defaultSummary, "")
	if err != nil {
		common.LogPrintln("", common.AddStr, common.ErStr, "添加任务失败:", err)
		return
	}
}

func AddExec(fileNameList []string, defaultM int, defaultK int, MGValue int, KGValue int, videoSize int, outputFPS int, encodeMaxSeconds int, encodeThread int, encodeFFmpegMode string, defaultSummary string, UUID string) error {
	fileDir := common.LumikaWorkDirPath
	fileEncodeDir := common.LumikaEncodePath
	fileEncodeOutputDir := common.LumikaEncodeOutputPath

	for ai, fileName := range fileNameList {
		common.LogPrintln(UUID, common.AddStr, "开始编码第"+strconv.Itoa(ai+1)+"个文件:", fileName)
		if fileName == "" {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "错误: 文件名不能为空，跳过该文件")
			continue
		}
		// 设置默认文件名
		filePath := filepath.Join(fileEncodeDir, fileName)
		defaultOutputDirName := "output_" + strings.ReplaceAll(fileName, ".", "_")
		defaultOutputDir := filepath.Join(fileEncodeOutputDir, defaultOutputDirName)
		// 创建输出目录
		err := os.Mkdir(defaultOutputDir, 0755)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "创建输出目录失败:", err)
			return &common.CommonError{Msg: "创建输出目录失败:" + err.Error()}
		}
		common.LogPrintln(UUID, common.AddStr, "使用默认文件名:", fileName)
		common.LogPrintln(UUID, common.AddStr, "使用默认输出目录:", defaultOutputDir)

		// 计算文件长度
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "文件长度计算失败:", err)
			return &common.CommonError{Msg: "文件长度计算失败:" + err.Error()}
		}
		fileSize := fileInfo.Size()

		// 开始生成 .fec 文件
		common.LogPrintln(UUID, common.AddStr, "开始生成 .fec 文件")
		zfecStartTime := time.Now()
		enc, err := reedsolomon.New(defaultK, defaultM-defaultK)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "创建RS编码器失败:", err)
			return &common.CommonError{Msg: "创建RS编码器失败:" + err.Error()}
		}
		b, err := os.ReadFile(filePath)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "读取文件失败:", err)
			return &common.CommonError{Msg: "读取文件失败:" + err.Error()}
		}
		shards, err := enc.Split(b)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "分割文件失败:", err)
			return &common.CommonError{Msg: "分割文件失败:" + err.Error()}
		}
		err = enc.Encode(shards)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "编码文件失败:", err)
			return &common.CommonError{Msg: "编码文件失败:" + err.Error()}
		}
		// 生成 fecHashList
		fecHashList := make([]string, len(shards))
		for i, shard := range shards {
			outfn := fmt.Sprintf("%s.%d_%d.fec", filepath.Base(filePath), i, len(shards))
			outfnPath := filepath.Join(defaultOutputDir, outfn)
			common.LogPrintln(UUID, common.AddStr, "写入 .fec 文件:", outfn)
			err = os.WriteFile(outfnPath, shard, 0755)
			if err != nil {
				common.LogPrintln(UUID, common.AddStr, common.ErStr, ".fec 文件写入失败:", err)
				return &common.CommonError{Msg: ".fec 文件写入失败:" + err.Error()}
			}
			fileHash := CalculateFileHash(outfnPath, common.DefaultHashLength)
			fecHashList[i] = fileHash
		}
		zfecEndTime := time.Now()
		zfecDuration := zfecEndTime.Sub(zfecStartTime)
		common.LogPrintln(UUID, common.AddStr, ".fec 文件生成完成，耗时:", zfecDuration)

		common.LogPrintln(UUID, common.AddStr, "开始进行编码")
		segmentLength, err := Encode(defaultOutputDir, videoSize, outputFPS, encodeMaxSeconds, MGValue, KGValue, encodeThread, encodeFFmpegMode, true, UUID)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "编码失败:", err)
			return &common.CommonError{Msg: "编码失败:" + err.Error()}
		}

		common.LogPrintln(UUID, common.AddStr, "编码完成，开始生成配置")
		fecFileConfig := common.FecFileConfig{
			Version:       common.LumikaVersionNum,
			Name:          fileName,
			Summary:       defaultSummary,
			Hash:          CalculateFileHash(filePath, common.DefaultHashLength),
			M:             defaultM,
			K:             defaultK,
			MG:            MGValue,
			KG:            KGValue,
			Length:        fileSize,
			SegmentLength: segmentLength,
			FecHashList:   fecHashList,
		}
		fecFileConfigJson, err := json.Marshal(fecFileConfig)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "生成 JSON 配置失败:", err)
			return &common.CommonError{Msg: "生成 JSON 配置失败:" + err.Error()}
		}
		// 转换为 Base64
		fecFileConfigBase64 := base64.StdEncoding.EncodeToString(fecFileConfigJson)
		fecFileConfigFilePath := filepath.Join(common.LumikaEncodeOutputPath, "lumika_config_"+strings.ReplaceAll(fileName, ".", "_")+".txt")
		common.LogPrintln(UUID, common.AddStr, "Base64 配置生成完成，开始写入文件:", fecFileConfigFilePath)
		err = os.WriteFile(fecFileConfigFilePath, []byte(fecFileConfigBase64), 0755)
		if err != nil {
			common.LogPrintln(UUID, common.AddStr, common.ErStr, "写入文件失败:", err)
			return &common.CommonError{Msg: "写入文件失败:" + err.Error()}
		}
		common.LogPrintln(UUID, common.AddStr, "写入配置成功")
		DeleteFecFiles(fileDir)

		// 将 Base64 配置对接到 Web API
		if UUID != "" {
			_, exist := common.AddTaskList[UUID]
			if exist {
				common.AddTaskList[UUID].BaseStr = fecFileConfigBase64
			} else {
				common.LogPrintln(UUID, common.AddStr, common.ErStr, "当前任务被用户删除")
				return &common.CommonError{Msg: "当前任务被用户删除"}
			}
		}

		common.LogPrintln(UUID, common.AddStr, "Base64 配置文件已生成，路径:", fecFileConfigFilePath)
		common.LogPrintln(UUID, common.AddStr, "Base64:", fecFileConfigBase64)
		common.LogPrintln(UUID, common.AddStr, "请将生成的 .mp4 fec 视频文件和 Base64 配置分享或发送给你的好友，对方可使用 \"GetInput\" 子命令来获取文件")
		common.LogPrintln(UUID, common.AddStr, "添加完成")
	}
	return nil
}
