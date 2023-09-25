package main

import (
	"flag"
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/utils"
	"os"
	"runtime"
	"strconv"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Usage: %s [command] [options]\n", os.Args[0])
		fmt.Fprintln(os.Stdout, "\nLumika", utils.LumikaVersionString)
		fmt.Fprintln(os.Stdout, "Double-click to run: Start via automatic mode")
		fmt.Fprintln(os.Stdout, "\nCommands:")
		fmt.Fprintln(os.Stdout, "version\tLumika version.")
		fmt.Fprintln(os.Stdout, "AddStr\tUsing FFmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.")
		fmt.Fprintln(os.Stdout, "GetStr\tUsing FFmpeg to decode .mp4 FEC video files into the original files.")
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

	addFlag := flag.NewFlagSet("AddStr", flag.ExitOnError)

	getFlag := flag.NewFlagSet("GetStr", flag.ExitOnError)

	if len(os.Args) < 2 {
		utils.AutoRun()
		utils.PressEnterToContinue()
		return
	}
	switch os.Args[1] {
	case "AddStr":
		err := addFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(utils.AddStr, utils.ErStr, "参数解析错误")
			return
		}
		utils.Add()
	case "GetStr":
		err := getFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(utils.GetStr, utils.ErStr, "参数解析错误")
			return
		}
		utils.Get()
	case "encode":
		err := encodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(utils.EnStr, utils.ErStr, "参数解析错误")
			return
		}
		utils.Encode(*encodeInput, *encodeQrcodeSize, *encodeOutputFPS, *encodeMaxSeconds, *encodeMGValue, *encodeKGValue, *encodeThread, *encodeFFmpegMode, false)
	case "decode":
		err := decodeFlag.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(utils.DeStr, utils.ErStr, "参数解析错误")
			return
		}
		utils.Decode(*decodeInputDir, 0, nil, *decodeMGValue, *decodeKGValue, *decodeThread)
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
		fmt.Println(utils.LumikaVersionString)
		return
	case "-v":
		fmt.Println(utils.LumikaVersionString)
		return
	case "--version":
		fmt.Println(utils.LumikaVersionString)
		return
	default:
		fmt.Println("Unknown command:", os.Args[1])
		flag.Usage()
	}
}
