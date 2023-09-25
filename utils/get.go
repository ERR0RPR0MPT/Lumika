package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/klauspost/reedsolomon"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

func Get() {
	base64Config := ""
	ep, err := os.Executable()
	if err != nil {
		fmt.Println(GetStr, ErStr, "无法获取运行目录:", err)
		return
	}
	epPath := filepath.Dir(ep)

	// 选择执行模式
	var fecDirList []string
	for {
		fmt.Println(GetStr, "请选择执行模式(默认为1):")
		fmt.Println(GetStr, "1. 读取本目录下所有的子目录并从子目录读取 Base64 配置文件(适用于解码多个文件)")
		fmt.Println(GetStr, "2. 从本目录下读取 Base64 配置文件(适用于解码单个文件)")
		result := GetUserInput("")
		if result == "1" || result == "" {
			// 读取本目录下所有的子目录
			dirList, err := GetSubDirectories(epPath)
			if err != nil {
				fmt.Println(GetStr, ErStr, "无法获取子目录:", err)
				return
			}
			if len(dirList) == 0 {
				fmt.Println(GetStr, ErStr, "没有找到子目录，请添加存放编码文件的目录")
				return
			}
			// 从子目录读取 Base64 配置文件，有配置文件的目录就放入 fecDirList
			for _, d := range dirList {
				if IsFileExistsInDir(d, "lumika_config") {
					fecDirList = append(fecDirList, d)
				}
			}
			if len(fecDirList) == 0 {
				fmt.Println(GetStr, ErStr, "没有找到子目录下的索引配置，请添加索引来解码")
				return
			}
			fmt.Println(GetStr, "找到存有索引配置的目录:")
			for i, d := range fecDirList {
				fmt.Println(GetStr, strconv.Itoa(i+1)+":", d)
			}
			break
		} else if result == "2" {
			// 从本目录读取 Base64 配置文件
			if IsFileExistsInDir(epPath, "lumika_config") {
				fecDirList = append(fecDirList, epPath)
			} else {
				fmt.Println(GetStr, ErStr, "没有找到本目录下的索引配置，请添加索引来解码")
				return
			}
			break
		} else {
			fmt.Println(GetStr, ErStr, "无效输入，请重新输入")
			continue
		}
	}

	// 设置处理使用的线程数量
	fmt.Println(GetStr, "请输入处理使用的线程数量。默认(CPU核心数量)：\""+strconv.Itoa(runtime.NumCPU())+"\"")
	decodeThread, err := strconv.Atoi(GetUserInput(""))
	if err != nil {
		fmt.Println(GetStr, "自动设置处理使用的线程数量为", runtime.NumCPU())
		decodeThread = runtime.NumCPU()
	}
	if decodeThread <= 0 {
		fmt.Println(GetStr, ErStr, "错误: 处理使用的线程数量不能小于等于 0，自动设置处理使用的线程数量为", runtime.NumCPU())
		decodeThread = runtime.NumCPU()
	}

	// 遍历每一个子目录并运行
	for _, fileDir := range fecDirList {
		// 搜索子目录的 Base64 配置文件
		configBase64FilePath := SearchFileNameInDir(fileDir, "lumika_config")
		fmt.Println(GetStr, "读取配置文件")
		// 读取文件
		configBase64Bytes, err := os.ReadFile(configBase64FilePath)
		if err != nil {
			fmt.Println(GetStr, ErStr, "读取文件失败:", err)
			continue
		}
		base64Config = string(configBase64Bytes)

		var fecFileConfig FecFileConfig
		fecFileConfigJson, err := base64.StdEncoding.DecodeString(base64Config)
		if err != nil {
			fmt.Println(GetStr, ErStr, "解析 Base64 配置失败:", err)
			continue
		}
		err = json.Unmarshal(fecFileConfigJson, &fecFileConfig)
		if err != nil {
			fmt.Println(GetStr, ErStr, "解析 JSON 配置失败:", err)
			continue
		}

		// 检测是否与当前版本匹配
		if fecFileConfig.Version != LumikaVersionNum {
			fmt.Println(GetStr, ErStr, "错误: 版本不匹配，无法进行解码。编码文件版本:", fecFileConfig.Version, "当前版本:", LumikaVersionNum)
			continue
		}

		// 查找 .mp4 文件
		fileDict, err := GenerateFileDxDictionary(fileDir, ".mp4")
		if err != nil {
			fmt.Println(GetStr, ErStr, "无法生成文件列表:", err)
			continue
		}

		// 修改文件名加上output前缀
		fecFileConfig.Name = "output_" + fecFileConfig.Name

		fmt.Println(GetStr, "文件名:", fecFileConfig.Name)
		fmt.Println(GetStr, "摘要:", fecFileConfig.Summary)
		fmt.Println(GetStr, "分段长度:", fecFileConfig.SegmentLength)
		fmt.Println(GetStr, "分段数量:", fecFileConfig.M)
		fmt.Println(GetStr, "Hash:", fecFileConfig.Hash)
		fmt.Println(GetStr, "在目录下找到以下匹配的 .mp4 文件:")
		for h, v := range fileDict {
			fmt.Println(GetStr, strconv.Itoa(h)+":", "文件路径:", v)
		}

		// 转换map[int]string 到 []string
		var fileDictList []string
		for _, v := range fileDict {
			fileDictList = append(fileDictList, v)
		}

		fmt.Println(GetStr, "开始解码")
		Decode(fileDir, fecFileConfig.SegmentLength, fileDictList, fecFileConfig.MG, fecFileConfig.KG, decodeThread)
		fmt.Println(GetStr, "解码完成")

		// 查找生成的 .fec 文件
		fileDict, err = GenerateFileDxDictionary(fileDir, ".fec")
		if err != nil {
			fmt.Println(GetStr, ErStr, "无法生成文件列表:", err)
			continue
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
				fmt.Println(GetStr, "警告：未找到匹配的 .fec 文件，Hash:", fecHash)
			} else {
				fmt.Println(GetStr, "找到匹配的 .fec 文件，Hash:", fecHash)
				findNum++
			}
		}
		fmt.Println(GetStr, "找到完整的 .fec 文件数量:", findNum)
		fmt.Println(GetStr, "未找到的文件数量:", fecFileConfig.M-findNum)
		fmt.Println(GetStr, "编码时生成的 .fec 文件数量(M):", fecFileConfig.M)
		fmt.Println(GetStr, "恢复所需最少的 .fec 文件数量(K):", fecFileConfig.K)
		if findNum >= fecFileConfig.K {
			fmt.Println(GetStr, "提示：可以成功恢复数据")
		} else {
			fmt.Println(GetStr, "警告：无法成功恢复数据，请按下回车键来确定")
			GetUserInput("请按回车键继续...")
		}

		// 生成原始文件
		fmt.Println(GetStr, "开始生成原始文件")
		zunfecStartTime := time.Now()
		enc, err := reedsolomon.New(fecFileConfig.K, fecFileConfig.M-fecFileConfig.K)
		if err != nil {
			fmt.Println(GetStr, ErStr, "无法构建RS解码器:", err)
			continue
		}
		shards := make([][]byte, fecFileConfig.M)
		for i := range shards {
			if fecFindFileList[i] == "" {
				fmt.Println(GetStr, "Index:", i, ", 警告：未找到匹配的 .fec 文件")
				continue
			}
			fmt.Println(GetStr, "Index:", i, ", 读取文件:", fecFindFileList[i])
			shards[i], err = os.ReadFile(fecFindFileList[i])
			if err != nil {
				fmt.Println(GetStr, ErStr, "读取 .fec 文件时出错", err)
				shards[i] = nil
			}
		}
		// 校验数据
		ok, err := enc.Verify(shards)
		if ok {
			fmt.Println(GetStr, "数据完整，不需要恢复")
		} else {
			fmt.Println(GetStr, "数据不完整，准备恢复数据")
			err = enc.Reconstruct(shards)
			if err != nil {
				fmt.Println(GetStr, ErStr, "恢复失败 -", err)
				DeleteFecFiles(fileDir)
				GetUserInput("请按回车键继续...")
				continue
			}
			ok, err = enc.Verify(shards)
			if !ok {
				fmt.Println(GetStr, ErStr, "恢复失败，数据可能已损坏")
				DeleteFecFiles(fileDir)
				GetUserInput("请按回车键继续...")
				continue
			}
			if err != nil {
				fmt.Println(GetStr, ErStr, "恢复失败 -", err)
				DeleteFecFiles(fileDir)
				GetUserInput("请按回车键继续...")
				continue
			}
			fmt.Println(GetStr, "恢复成功")
		}
		fmt.Println(GetStr, "写入文件到:", fecFileConfig.Name)
		f, err := os.Create(fecFileConfig.Name)
		if err != nil {
			fmt.Println(GetStr, ErStr, "创建文件失败:", err)
			continue
		}
		err = enc.Join(f, shards, len(shards[0])*fecFileConfig.K)
		if err != nil {
			fmt.Println(GetStr, ErStr, "写入文件失败:", err)
			continue
		}
		f.Close()
		err = TruncateFile(fecFileConfig.Length, filepath.Join(epPath, fecFileConfig.Name))
		if err != nil {
			fmt.Println(GetStr, ErStr, "截断解码文件失败:", err)
			continue
		}
		zunfecEndTime := time.Now()
		zunfecDuration := zunfecEndTime.Sub(zunfecStartTime)
		fmt.Println(GetStr, "生成原始文件成功，耗时:", zunfecDuration)
		DeleteFecFiles(fileDir)
		// 检查最终生成的文件是否与原始文件一致
		fmt.Println(GetStr, "检查生成的文件是否与源文件一致")
		targetHash := CalculateFileHash(filepath.Join(epPath, fecFileConfig.Name), DefaultHashLength)
		if targetHash != fecFileConfig.Hash {
			fmt.Println(GetStr, ErStr, "警告: 生成的文件与源文件不一致")
			fmt.Println(GetStr, ErStr, "源文件 Hash:", fecFileConfig.Hash)
			fmt.Println(GetStr, ErStr, "生成文件 Hash:", targetHash)
			fmt.Println(GetStr, ErStr, "文件解码失败")
		} else {
			fmt.Println(GetStr, "生成的文件与源文件一致")
			fmt.Println(GetStr, "源文件 Hash:", fecFileConfig.Hash)
			fmt.Println(GetStr, "生成文件 Hash:", targetHash)
			fmt.Println(GetStr, "文件成功解码")
		}
		fmt.Println(GetStr, "获取完成")
	}
}
