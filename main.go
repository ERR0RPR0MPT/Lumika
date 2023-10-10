package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/ERR0RPR0MPT/Lumika/utils"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed ui/*
var UIFiles embed.FS

func LumikaDataPathInit(p string) {
	common.LumikaWorkDirPath = filepath.Join(p, common.LumikaWorkDirName)
	common.LumikaEncodePath = filepath.Join(p, common.LumikaWorkDirName, "encode")
	common.LumikaDecodePath = filepath.Join(p, common.LumikaWorkDirName, "decode")
	common.LumikaEncodeOutputPath = filepath.Join(p, common.LumikaWorkDirName, "encodeOutput")
	common.LumikaDecodeOutputPath = filepath.Join(p, common.LumikaWorkDirName, "decodeOutput")
	// 创建 Lumika 工作目录
	if _, err := os.Stat(common.LumikaWorkDirPath); err == nil {
		common.LogPrintln("", common.InitStr, "Lumika 工作目录已存在，跳过创建 Lumika 工作目录")
		if _, err := os.Stat(common.LumikaEncodePath); err != nil {
			common.LogPrintln("", common.InitStr, "创建 encode 工作目录")
			err = os.Mkdir(common.LumikaEncodePath, 0755)
			if err != nil {
				common.LogPrintln("", common.InitStr, "创建 encode 目录失败:", err)
				return
			}
		}
		if _, err := os.Stat(common.LumikaDecodePath); err != nil {
			common.LogPrintln("", common.InitStr, "创建 decode 工作目录")
			err = os.Mkdir(common.LumikaDecodePath, 0755)
			if err != nil {
				common.LogPrintln("", common.InitStr, "创建 decode 目录失败:", err)
				return
			}
		}
		if _, err := os.Stat(common.LumikaEncodeOutputPath); err != nil {
			common.LogPrintln("", common.InitStr, "创建 encodeOutput 工作目录")
			err = os.Mkdir(common.LumikaEncodeOutputPath, 0755)
			if err != nil {
				common.LogPrintln("", common.InitStr, "创建 encodeOutput 目录失败:", err)
				return
			}
		}
		if _, err := os.Stat(common.LumikaDecodeOutputPath); err != nil {
			common.LogPrintln("", common.InitStr, "创建 decodeOutput 工作目录")
			err = os.Mkdir(common.LumikaDecodeOutputPath, 0755)
			if err != nil {
				common.LogPrintln("", common.InitStr, "创建 decodeOutput 目录失败:", err)
				return
			}
		}
	} else {
		common.LogPrintln("", common.InitStr, "Lumika 工作目录不存在，创建 Lumika 工作目录")
		err = os.Mkdir(common.LumikaWorkDirPath, 0755)
		if err != nil {
			common.LogPrintln("", common.InitStr, "创建 Lumika 工作目录失败:", err)
			return
		}
		common.LogPrintln("", common.InitStr, "创建 encode 工作目录")
		err = os.Mkdir(common.LumikaEncodePath, 0755)
		if err != nil {
			common.LogPrintln("", common.InitStr, "创建 encode 目录失败:", err)
			return
		}
		common.LogPrintln("", common.InitStr, "创建 decode 工作目录")
		err = os.Mkdir(common.LumikaDecodePath, 0755)
		if err != nil {
			common.LogPrintln("", common.InitStr, "创建 decode 目录失败:", err)
			return
		}
		common.LogPrintln("", common.InitStr, "创建 encodeOutput 工作目录")
		err = os.Mkdir(common.LumikaEncodeOutputPath, 0755)
		if err != nil {
			common.LogPrintln("", common.InitStr, "创建 encodeOutput 目录失败:", err)
			return
		}
		common.LogPrintln("", common.InitStr, "创建 decodeOutput 工作目录")
		err = os.Mkdir(common.LumikaDecodeOutputPath, 0755)
		if err != nil {
			common.LogPrintln("", common.InitStr, "创建 decodeOutput 目录失败:", err)
			return
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	est, err := os.Executable()
	if err != nil {
		common.LogPrintln("", common.InitStr, "工作目录获取失败")
		return
	}
	common.EpPath = filepath.Dir(est)
	UISubFiles, err := fs.Sub(UIFiles, "ui")
	if err != nil {
		fmt.Println("静态文件加载失败:", err)
		return
	}
	common.UISubFiles = UISubFiles
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [command] [options]\n", os.Args[0])
		fmt.Fprintln(os.Stdout, "\nLumika", common.LumikaVersionString)
		fmt.Fprintln(os.Stdout, "Double-click to run: Start via automatic mode")
		fmt.Fprintln(os.Stdout, "\nCommands:")
		fmt.Fprintln(os.Stdout, "version\tOutput Lumika version.")
		fmt.Fprintln(os.Stdout, "web\tStart Lumika Backend and Lumika Web Server, default listen on :7860.")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -h\tThe host to listen on(default=\"\")")
		fmt.Fprintln(os.Stdout, " -p\tThe port to listen on(default=7860)")
		fmt.Fprintln(os.Stdout, "add\tUsing FFmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.")
		fmt.Fprintln(os.Stdout, "get\tUsing FFmpeg to decode .mp4 FEC video files into the original files.")
		fmt.Fprintln(os.Stdout, "encode\tEncode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input fec file to encode")
		fmt.Fprintln(os.Stdout, " -s\tThe video size(default="+strconv.Itoa(common.EncodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
		fmt.Fprintln(os.Stdout, " -p\tThe output video fps setting(default="+strconv.Itoa(common.EncodeOutputFPSLevel)+"), 1-60")
		fmt.Fprintln(os.Stdout, " -l\tThe output video max segment length(seconds) setting(default="+strconv.Itoa(common.EncodeMaxSecondsLevel)+"), 1-10^9")
		fmt.Fprintln(os.Stdout, " -g\tThe output video frame all shards(default="+strconv.Itoa(common.AddMGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, " -k\tThe output video frame data shards(default="+strconv.Itoa(common.AddKGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, " -m\tFFmpeg mode(default="+common.EncodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
		fmt.Fprintln(os.Stdout, "decode\tDecode a file")
		fmt.Fprintln(os.Stdout, " Options:")
		fmt.Fprintln(os.Stdout, " -i\tThe input file to decode")
		fmt.Fprintln(os.Stdout, " -m\tThe output video frame all shards(default="+strconv.Itoa(common.AddMGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, " -k\tThe output video frame data shards(default="+strconv.Itoa(common.AddKGLevel)+"), 2-256")
		fmt.Fprintln(os.Stdout, "help\tShow this help")
		flag.PrintDefaults()
	}
	encodeFlag := flag.NewFlagSet("encode", flag.ExitOnError)
	encodeInput := encodeFlag.String("i", "", "The input fec file to encode")
	encodeQrcodeSize := encodeFlag.Int("s", common.EncodeVideoSizeLevel, "The video size(default="+strconv.Itoa(common.EncodeVideoSizeLevel)+"), 8-1024(must be a multiple of 8)")
	encodeOutputFPS := encodeFlag.Int("p", common.EncodeOutputFPSLevel, "The output video fps setting(default="+strconv.Itoa(common.EncodeOutputFPSLevel)+"), 1-60")
	encodeMaxSeconds := encodeFlag.Int("l", common.EncodeMaxSecondsLevel, "The output video max segment length(seconds) setting(default="+strconv.Itoa(common.EncodeMaxSecondsLevel)+"), 1-10^9")
	encodeMGValue := encodeFlag.Int("g", common.AddMGLevel, "The output video frame all shards(default="+strconv.Itoa(common.AddMGLevel)+"), 2-256")
	encodeKGValue := encodeFlag.Int("k", common.AddKGLevel, "The output video frame data shards(default="+strconv.Itoa(common.AddKGLevel)+"), 2-256")
	encodeThread := encodeFlag.Int("t", common.VarSettingsVariable.DefaultMaxThreads, "Set Runtime Go routines number to process the task(default="+strconv.Itoa(runtime.NumCPU())+"), 1-128")
	encodeFFmpegMode := encodeFlag.String("m", common.EncodeFFmpegModeLevel, "FFmpeg mode(default="+common.EncodeFFmpegModeLevel+"): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo")
	encodePath := encodeFlag.String("d", common.EpPath, "The dir path to save lumika data files")

	decodeFlag := flag.NewFlagSet("decode", flag.ExitOnError)
	decodeInputDir := decodeFlag.String("i", "", "The input dir include video segments to decode")
	decodeMGValue := decodeFlag.Int("m", common.AddMGLevel, "The output video frame all shards(default="+strconv.Itoa(common.AddMGLevel)+"), 2-256")
	decodeKGValue := decodeFlag.Int("k", common.AddKGLevel, "The output video frame data shards(default="+strconv.Itoa(common.AddKGLevel)+"), 2-256")
	decodeThread := decodeFlag.Int("t", common.VarSettingsVariable.DefaultMaxThreads, "Set Runtime Go routines number to process the task(default="+strconv.Itoa(runtime.NumCPU())+"), 1-128")
	decodePath := decodeFlag.String("d", common.EpPath, "The dir path to save lumika data files")

	addFlag := flag.NewFlagSet("add", flag.ExitOnError)
	addPath := addFlag.String("d", common.EpPath, "The dir path to save lumika data files")

	getFlag := flag.NewFlagSet("get", flag.ExitOnError)
	getPath := getFlag.String("d", common.EpPath, "The dir path to save lumika data files")

	dlFlag := flag.NewFlagSet("dl", flag.ExitOnError)
	dlPath := dlFlag.String("d", common.EpPath, "The dir path to save lumika data files")

	webFlag := flag.NewFlagSet("web", flag.ExitOnError)
	webHost := webFlag.String("h", common.DefaultWebServerHost, "The host to listen on")
	webPort := webFlag.Int("p", common.DefaultWebServerPort, "The port to listen on")
	webPath := webFlag.String("d", common.EpPath, "The dir path to save lumika data files")

	if len(os.Args) < 2 {
		LumikaDataPathInit(common.EpPath)
		utils.WebServer(common.DefaultWebServerHost, common.DefaultWebServerPort)
		return
	}
	switch os.Args[1] {
	case "web":
		err := webFlag.Parse(os.Args[2:])
		if err != nil {
			common.LogPrintln("", common.WebStr, common.ErStr, "参数解析错误")
			return
		}
		p := common.EpPath
		if *webPath != "" {
			p = *webPath
		}
		LumikaDataPathInit(p)
		utils.WebServer(*webHost, *webPort)
		return
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
			common.LogPrintln("", common.AddStr, common.ErStr, "参数解析错误")
			return
		}
		p := common.EpPath
		if *addPath != "" {
			p = *addPath
		}
		LumikaDataPathInit(p)
		utils.AddInput()
		return
	case "get":
		err := getFlag.Parse(os.Args[2:])
		if err != nil {
			common.LogPrintln("", common.GetStr, common.ErStr, "参数解析错误")
			return
		}
		p := common.EpPath
		if *getPath != "" {
			p = *getPath
		}
		LumikaDataPathInit(p)
		utils.GetInput()
		return
	case "dl":
		err := dlFlag.Parse(os.Args[2:])
		if err != nil || len(os.Args) < 3 {
			common.LogPrintln("", common.BDlStr, common.ErStr, "参数解析错误，请正确填写 av/BV 号，例如：", os.Args[0], "dl", "av2")
			return
		}
		if os.Args[2] == "" || (!strings.Contains(os.Args[2], "BV") && !strings.Contains(os.Args[2], "av")) {
			common.LogPrintln("", common.BDlStr, common.ErStr, "参数解析错误，请输入正确的av/BV号")
			return
		}
		p := common.EpPath
		if *dlPath != "" {
			p = *dlPath
		}
		LumikaDataPathInit(p)
		err = utils.BDl(os.Args[2], "encode", "")
		if err != nil {
			common.LogPrintln("", common.BDlStr, common.ErStr, "从哔哩源下载失败:", err)
			return
		}
		return
	case "encode":
		err := encodeFlag.Parse(os.Args[2:])
		if err != nil {
			common.LogPrintln("", common.EnStr, common.ErStr, "参数解析错误")
			return
		}
		p := common.EpPath
		if *encodePath != "" {
			p = *encodePath
		}
		LumikaDataPathInit(p)
		_, err = utils.Encode(*encodeInput, *encodeQrcodeSize, *encodeOutputFPS, *encodeMaxSeconds, *encodeMGValue, *encodeKGValue, *encodeThread, *encodeFFmpegMode, false, "")
		if err != nil {
			common.LogPrintln("", common.EnStr, common.ErStr, "编码失败:", err)
			return
		}
		return
	case "decode":
		err := decodeFlag.Parse(os.Args[2:])
		if err != nil {
			common.LogPrintln("", common.DeStr, common.ErStr, "参数解析错误")
			return
		}
		p := common.EpPath
		if *decodePath != "" {
			p = *decodePath
		}
		LumikaDataPathInit(p)
		err = utils.Decode(*decodeInputDir, 0, nil, *decodeMGValue, *decodeKGValue, *decodeThread, "")
		if err != nil {
			common.LogPrintln("", common.DeStr, common.ErStr, "解码失败:", err)
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
		common.LogPrintln("", common.LumikaVersionString)
		return
	case "-v":
		common.LogPrintln("", common.LumikaVersionString)
		return
	case "--version":
		common.LogPrintln("", common.LumikaVersionString)
		return
	default:
		common.LogPrintln("", "Unknown command:", os.Args[1])
		flag.Usage()
	}
}
