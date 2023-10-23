package utils

import (
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"os"
)

func AutoRun() {
	common.LogPrintln("", common.ArStr, "使用 \""+os.Args[0]+" help\" 查看帮助")
	common.LogPrintln("", common.ArStr, "请选择你要执行的操作:")
	common.LogPrintln("", common.ArStr, "  1. 添加")
	common.LogPrintln("", common.ArStr, "  2. 获取")
	common.LogPrintln("", common.ArStr, "  3. 编码")
	common.LogPrintln("", common.ArStr, "  4. 解码")
	common.LogPrintln("", common.ArStr, "  5. 退出")
	for {
		fmt.Print(common.ArStr, "请输入操作编号: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			common.LogPrintln("", common.ArStr, common.ErStr, "错误: 请重新输入")
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
			_, err := Encode("", common.EncodeVideoSizeLevel, common.EncodeOutputFPSLevel, common.EncodeMaxSecondsLevel, common.AddMGLevel, common.AddKGLevel, common.VarSettingsVariable.DefaultMaxThreads, common.EncodeFFmpegModeLevel, false, common.EncodeVersion, common.EncodeVer5ColorGA, common.EncodeVer5ColorBA, common.EncodeVer5ColorGB, common.EncodeVer5ColorBB, "")
			if err != nil {
				common.LogPrintln("", common.ArStr, common.ErStr, "错误: 编码失败:", err)
				break
			}
			break
		} else if input == "4" {
			clearScreen()
			err := Decode("", 0, nil, common.AddMGLevel, common.AddKGLevel, common.VarSettingsVariable.DefaultMaxThreads, "")
			if err != nil {
				common.LogPrintln("", common.ArStr, common.ErStr, "错误: 解码失败:", err)
				return
			}
			break
		} else if input == "5" {
			os.Exit(0)
		} else {
			common.LogPrintln("", common.ArStr, common.ErStr, "错误: 无效的操作编号")
			continue
		}
	}
}
