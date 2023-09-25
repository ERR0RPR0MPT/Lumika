package utils

import (
	"fmt"
	"os"
	"runtime"
)

func AutoRun() {
	fmt.Println(ArStr, "使用 \""+os.Args[0]+" help\" 查看帮助")
	fmt.Println(ArStr, "请选择你要执行的操作:")
	fmt.Println(ArStr, "  1. 添加")
	fmt.Println(ArStr, "  2. 获取")
	fmt.Println(ArStr, "  3. 编码")
	fmt.Println(ArStr, "  4. 解码")
	fmt.Println(ArStr, "  5. 退出")
	for {
		fmt.Print(ArStr, "请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println(ArStr, ErStr, "错误: 请重新输入")
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
			Encode("", EncodeVideoSizeLevel, EncodeOutputFPSLevel, EncodeMaxSecondsLevel, AddMGLevel, AddKGLevel, runtime.NumCPU(), EncodeFFmpegModeLevel, false)
			break
		} else if input == "4" {
			clearScreen()
			Decode("", 0, nil, AddMGLevel, AddKGLevel, runtime.NumCPU())
			break
		} else if input == "5" {
			os.Exit(0)
		} else {
			fmt.Println(ArStr, ErStr, "错误: 无效的操作编号")
			continue
		}
	}
}
