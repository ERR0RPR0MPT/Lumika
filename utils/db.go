package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func DbCrontab() {
	// 每隔 DefaultDbCrontabSeconds 秒存储一次数据库
	wd := filepath.Join(LumikaWorkDirPath, "db.json")
	for {
		time.Sleep(time.Second * DefaultDbCrontabSeconds)
		db := &Database{
			DlTaskList:  DlTaskList,
			BDlTaskList: BDlTaskList,
			AddTaskList: AddTaskList,
			GetTaskList: GetTaskList,
			BUlTaskList: BUlTaskList,
		}
		jsonData, err := json.Marshal(db)
		if err != nil {
			LogPrintln("", DbStr, "转换为JSON时发生错误:", err)
			return
		}
		err = os.WriteFile(wd, jsonData, 0644)
		if err != nil {
			LogPrintln("", DbStr, "保存JSON文件时发生错误:", err)
			return
		}
	}
}

func DbInit() {
	LogPrintln("", DbStr, InitStr, "初始化数据库")
	wd := filepath.Join(LumikaWorkDirPath, "db.json")
	if _, err := os.Stat(wd); err == nil {
		LogPrintln("", DbStr, InitStr, "读取数据库文件")
		jsonData, err := os.ReadFile(wd)
		if err != nil {
			fmt.Println("读取JSON文件时发生错误:", err)
			return
		}
		err = json.Unmarshal(jsonData, &database)
		if err != nil {
			fmt.Println("解析JSON数据时发生错误:", err)
			return
		}
	}
	go DbCrontab()
}
