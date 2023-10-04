package utils

import (
	"os"
	"path/filepath"
)

func DbInit() {
	// 获取程序数据目录
	est, err := os.Executable()
	if err != nil {
		LogPrintln("", DbStr, InitStr, "获取程序目录失败：", err)
		return
	}
	WorkDir := filepath.Dir(est)
	LumikaWorkDir := filepath.Join(WorkDir, LumikaWorkDirName)
	// 检查是否存在数据库文件
	if _, err := os.Stat(filepath.Join(LumikaWorkDir, "db.json")); err == nil {
		// 读取数据库文件
		LogPrintln("", DbStr, InitStr, "读取数据库文件")
	} else {
		// 创建新的数据库文件
		LogPrintln("", DbStr, InitStr, "创建新的数据库文件")
	}
}
