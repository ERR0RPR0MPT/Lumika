package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/utils"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed ui/*
var uiFiles embed.FS

func MainInit() {
	UIFiles, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		fmt.Println("静态文件加载失败:", err)
		return
	}
	utils.UIFiles = UIFiles
	est, err := os.Executable()
	if err != nil {
		utils.LogPrintln("", utils.InitStr, "工作目录获取失败")
		return
	}
	utils.EpPath = filepath.Dir(est)
	utils.LumikaWorkDirPath = filepath.Join(utils.EpPath, utils.LumikaWorkDirName)
	utils.LumikaEncodePath = filepath.Join(utils.EpPath, utils.LumikaWorkDirName, "encode")
	utils.LumikaDecodePath = filepath.Join(utils.EpPath, utils.LumikaWorkDirName, "decode")
	utils.LumikaEncodeOutputPath = filepath.Join(utils.EpPath, utils.LumikaWorkDirName, "encodeOutput")
	utils.LumikaDecodeOutputPath = filepath.Join(utils.EpPath, utils.LumikaWorkDirName, "decodeOutput")
	// 创建 Lumika 工作目录
	if _, err := os.Stat(utils.LumikaWorkDirPath); err == nil {
		utils.LogPrintln("", utils.InitStr, "Lumika 工作目录已存在，跳过创建 Lumika 工作目录")
		if _, err := os.Stat(utils.LumikaEncodePath); err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 encode 工作目录")
			err = os.Mkdir(utils.LumikaEncodePath, 0755)
			if err != nil {
				utils.LogPrintln("", utils.InitStr, "创建 encode 目录失败:", err)
				return
			}
		}
		if _, err := os.Stat(utils.LumikaDecodePath); err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 decode 工作目录")
			err = os.Mkdir(utils.LumikaDecodePath, 0755)
			if err != nil {
				utils.LogPrintln("", utils.InitStr, "创建 decode 目录失败:", err)
				return
			}
		}
		if _, err := os.Stat(utils.LumikaEncodeOutputPath); err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 encodeOutput 工作目录")
			err = os.Mkdir(utils.LumikaEncodeOutputPath, 0755)
			if err != nil {
				utils.LogPrintln("", utils.InitStr, "创建 encodeOutput 目录失败:", err)
				return
			}
		}
		if _, err := os.Stat(utils.LumikaDecodeOutputPath); err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 decodeOutput 工作目录")
			err = os.Mkdir(utils.LumikaDecodeOutputPath, 0755)
			if err != nil {
				utils.LogPrintln("", utils.InitStr, "创建 decodeOutput 目录失败:", err)
				return
			}
		}
	} else {
		utils.LogPrintln("", utils.InitStr, "Lumika 工作目录不存在，创建 Lumika 工作目录")
		err = os.Mkdir(utils.LumikaWorkDirPath, 0755)
		if err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 Lumika 工作目录失败:", err)
			return
		}
		utils.LogPrintln("", utils.InitStr, "创建 encode 工作目录")
		err = os.Mkdir(utils.LumikaEncodePath, 0755)
		if err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 encode 目录失败:", err)
			return
		}
		utils.LogPrintln("", utils.InitStr, "创建 decode 工作目录")
		err = os.Mkdir(utils.LumikaDecodePath, 0755)
		if err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 decode 目录失败:", err)
			return
		}
		utils.LogPrintln("", utils.InitStr, "创建 encodeOutput 工作目录")
		err = os.Mkdir(utils.LumikaEncodeOutputPath, 0755)
		if err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 encodeOutput 目录失败:", err)
			return
		}
		utils.LogPrintln("", utils.InitStr, "创建 decodeOutput 工作目录")
		err = os.Mkdir(utils.LumikaDecodeOutputPath, 0755)
		if err != nil {
			utils.LogPrintln("", utils.InitStr, "创建 decodeOutput 目录失败:", err)
			return
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * utils.DefaultMaxConcurrencyTimes)
	MainInit()
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [command] [options]\n", os.Args[0])
		fmt.Fprintln(os.Stdout, "\nLumika", utils.LumikaVersionString)
		fmt.Fprintln(os.Stdout, "Double-click to run: Start via automatic mode")
		fmt.Fprintln(os.Stdout, "\nCommands:")
		fmt.Fprintln(os.Stdout, "version\tLumika version.")
		fmt.Fprintln(os.Stdout, "add\tUsing FFmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.")
		fmt.Fprintln(os.Stdout, "get\tUsing FFmpeg to decode .mp4 FEC video files into the original files.")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -b\tThe Base64 encoded JSON included message to provide decode")
		fmt.Fprintln(os.Stdout, "encode\tEncode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input fec file to encode")
		fmt.Fprintln(os.Stdout, " -s\tThe video size(default="+strconv.Itoa(utils.EncodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
		fmt.Fprintln(os.Stdout, " -p\tThe output video fps setting(default="+strconv.Itoa(utils.EncodeOutputFPSLevel)+"), 1-60")
		fmt.Fprintln(os.Stdout, " -l\tThe output video max segment length(seconds) setting(default="+strconv.Itoa(utils.EncodeMaxSecondsLevel)+"), 1-10^9")
		fmt.Fprintln(os.Stdout, " -g\tThe output video frame all shards(default="+strconv.Itoa(utils.AddMGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, " -k\tThe output video frame data shards(default="+strconv.Itoa(utils.AddKGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, " -m\tFFmpeg mode(default="+utils.EncodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
		fmt.Fprintln(os.Stdout, "decode\tDecode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input file to decode")
		fmt.Fprintln(os.Stdout, " -m\tThe output video frame all shards(default="+strconv.Itoa(utils.AddMGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, " -k\tThe output video frame data shards(default="+strconv.Itoa(utils.AddKGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, "help\tShow this help")
		flag.PrintDefaults()
	}
	encodeFlag := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeInput := encodeFlag.String("i", "", "The input fec file to encode")
	encodeQrcodeSize := encodeFlag.Int("s", utils.EncodeVideoSizeLevel, "The video size(default="+strconv.Itoa(utils.EncodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
	encodeOutputFPS := encodeFlag.Int("p", utils.EncodeOutputFPSLevel, "The output video fps setting(default="+strconv.Itoa(utils.EncodeOutputFPSLevel)+"), 1-60")
	encodeMaxSeconds := encodeFlag.Int("l", utils.EncodeMaxSecondsLevel, "The output video max segment length(seconds) setting(default="+strconv.Itoa(utils.EncodeMaxSecondsLevel)+"), 1-10^9")
	encodeMGValue := encodeFlag.Int("g", utils.AddMGLevel, "The output video frame all shards(default="+strconv.Itoa(utils.AddMGLevel)+"), 2-256")
	encodeKGValue := encodeFlag.Int("k", utils.AddKGLevel, "The output video frame data shards(default="+strconv.Itoa(utils.AddKGLevel)+"), 2-256")
	encodeThread := encodeFlag.Int("t", runtime.NumCPU(), "Set Runtime Go routines number to process the task(default="+strconv.Itoa(runtime.NumCPU())+"), 1-128")
	encodeFFmpegMode := encodeFlag.String("m", utils.EncodeFFmpegModeLevel, "FFmpeg mode(default="+utils.EncodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")

	decodeFlag := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeInputDir := decodeFlag.String("i", "", "The input dir include video segments to decode")
	decodeMGValue := decodeFlag.Int("m", utils.AddMGLevel, "The output video frame all shards(default="+strconv.Itoa(utils.AddMGLevel)+"), 2-256")
	decodeKGValue := decodeFlag.Int("k", utils.AddKGLevel, "The output video frame data shards(default="+strconv.Itoa(utils.AddKGLevel)+"), 2-256")
	decodeThread := decodeFlag.Int("t", runtime.NumCPU(), "Set Runtime Go routines number to process the task(default="+strconv.Itoa(runtime.NumCPU())+"), 1-128")

	addFlag := flag.NewFlagSet("add", flag.ExitOnError)

	getFlag := flag.NewFlagSet("get", flag.ExitOnError)

	dlFlag := flag.NewFlagSet("dl", flag.ExitOnError)

	if len(os.Args) < 2 {
		utils.WebServer()
		return
	}
	switch os.Args[1] {
	case "a":
		utils.AutoRun()
		utils.PressEnterToContinue()
		return
	case "autorun":
		utils.AutoRun()
		utils.PressEnterToContinue()
		return
	case "add":
		err := addFlag.Parse(os.Args[2:])
		if err != nil {
			utils.LogPrintln("", utils.AddStr, utils.ErStr, "参数解析错误")
			return
		}
		utils.AddInput()
		return
	case "get":
		err := getFlag.Parse(os.Args[2:])
		if err != nil {
			utils.LogPrintln("", utils.GetStr, utils.ErStr, "参数解析错误")
			return
		}
		utils.GetInput()
		return
	case "dl":
		err := dlFlag.Parse(os.Args[2:])
		if err != nil || len(os.Args) < 3 {
			utils.LogPrintln("", utils.BDlStr, utils.ErStr, "参数解析错误，请正确填写 av/BV 号，例如：", os.Args[0], "dl", "av2")
			return
		}
		if os.Args[2] == "" || (!strings.Contains(os.Args[2], "BV") && !strings.Contains(os.Args[2], "av")) {
			utils.LogPrintln("", utils.BDlStr, utils.ErStr, "参数解析错误，请输入正确的av/BV号")
			return
		}
		err = utils.BDl(os.Args[2], "encode", "")
		if err != nil {
			utils.LogPrintln("", utils.BDlStr, utils.ErStr, "从哔哩源下载失败:", err)
			return
		}
		return
	case "encode":
		err := encodeFlag.Parse(os.Args[2:])
		if err != nil {
			utils.LogPrintln("", utils.EnStr, utils.ErStr, "参数解析错误")
			return
		}
		_, err = utils.Encode(*encodeInput, *encodeQrcodeSize, *encodeOutputFPS, *encodeMaxSeconds, *encodeMGValue, *encodeKGValue, *encodeThread, *encodeFFmpegMode, false, "")
		if err != nil {
			utils.LogPrintln("", utils.EnStr, utils.ErStr, "编码失败:", err)
			return
		}
		return
	case "decode":
		err := decodeFlag.Parse(os.Args[2:])
		if err != nil {
			utils.LogPrintln("", utils.DeStr, utils.ErStr, "参数解析错误")
			return
		}
		err = utils.Decode(*decodeInputDir, 0, nil, *decodeMGValue, *decodeKGValue, *decodeThread, "")
		if err != nil {
			utils.LogPrintln("", utils.DeStr, utils.ErStr, "解码失败:", err)
			return
		}
		return
	case "help":
		flag.Usage()
		return
	case "-h":
		flag.Usage()
		return
	case "--help":
		flag.Usage()
		return
	case "version":
		utils.LogPrintln("", utils.LumikaVersionString)
		return
	case "-v":
		utils.LogPrintln("", utils.LumikaVersionString)
		return
	case "--version":
		utils.LogPrintln("", utils.LumikaVersionString)
		return
	default:
		utils.LogPrintln("", "Unknown command:", os.Args[1])
		flag.Usage()
	}
}
