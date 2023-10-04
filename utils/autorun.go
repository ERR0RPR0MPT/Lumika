package utils

import (
	"fmt"
	"os"
	"runtime"
)

func AutoRun() {
	LogPrintln("", ArStr, "使用 \""+os.Args[0]+" help\" 查看帮助")
	LogPrintln("", ArStr, "请选择你要执行的操作:")
	LogPrintln("", ArStr, "  1. 添加")
	LogPrintln("", ArStr, "  2. 获取")
	LogPrintln("", ArStr, "  3. 编码")
	LogPrintln("", ArStr, "  4. 解码")
	LogPrintln("", ArStr, "  5. 退出")
	for {
		fmt.Print(ArStr, "请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			LogPrintln("", ArStr, ErStr, "错误: 请重新输入")
			continue
		}
		if input == "1" {
			clearScreen()
			AddInput()
			break
		} else if input == "2" {
			clearScreen()
			GetInput()
			break
		} else if input == "3" {
			clearScreen()
			_, err := Encode("", EncodeVideoSizeLevel, EncodeOutputFPSLevel, EncodeMaxSecondsLevel, AddMGLevel, AddKGLevel, runtime.NumCPU(), EncodeFFmpegModeLevel, false, "")
			if err != nil {
				LogPrintln("", ArStr, ErStr, "错误: 编码失败:", err)
				break
			}
			break
		} else if input == "4" {
			clearScreen()
			Decode("", 0, nil, AddMGLevel, AddKGLevel, runtime.NumCPU(), "")
			break
		} else if input == "5" {
			os.Exit(0)
		} else {
			LogPrintln("", ArStr, ErStr, "错误: 无效的操作编号")
			continue
		}
	}
}
