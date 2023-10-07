package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func DbSave(wd string, i int) {
	db := &Database{
		DlTaskList:  DlTaskList,
		BDlTaskList: BDlTaskList,
		AddTaskList: AddTaskList,
		GetTaskList: GetTaskList,
		BUlTaskList: BUlTaskList,
		VarSettings: &VarSettingsVariable,
	}
	jsonData, err := json.Marshal(db)
	if err != nil {
		LogPrintln("", DbStr, "转换为JSON时发生错误:", err)
		return
	}
	err = os.WriteFile(wd, jsonData, 0755)
	if err != nil {
		LogPrintln("", DbStr, "保存JSON文件时发生错误:", err)
		return
	}
	if i%360 == 0 {
		LogPrintln("", DbStr, "数据库已保存")
	}
}

func DbCrontab() {
	// 每隔 DefaultDbCrontabSeconds 秒存储一次数据库
	wd := filepath.Join(LumikaWorkDirPath, "db.json")
	// 每次启动前等待 20s
	time.Sleep(time.Second * 20)
	i := 0
	for {
		DbSave(wd, i)
		time.Sleep(time.Second * time.Duration(VarSettingsVariable.DefaultDbCrontabSeconds))
		i++
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
		// 更新用户配置
		if database.VarSettings == nil {
			// 使用默认配置
			database.VarSettings = &VarSettings{
				DefaultMaxThreads:               runtime.NumCPU(),
				DefaultBiliDownloadGoRoutines:   DefaultBiliDownloadGoRoutines,
				DefaultBiliDownloadsMaxQueueNum: DefaultBiliDownloadsMaxQueueNum,
				DefaultTaskWorkerGoRoutines:     DefaultTaskWorkerGoRoutines,
				DefaultDbCrontabSeconds:         DefaultDbCrontabSeconds,
			}
		}
		VarSettingsVariable = *database.VarSettings
		if VarSettingsVariable.DefaultMaxThreads <= 0 {
			VarSettingsVariable.DefaultMaxThreads = runtime.NumCPU()
		}
		if VarSettingsVariable.DefaultBiliDownloadGoRoutines <= 0 {
			VarSettingsVariable.DefaultBiliDownloadGoRoutines = DefaultBiliDownloadGoRoutines
		}
		if VarSettingsVariable.DefaultBiliDownloadsMaxQueueNum <= 0 {
			VarSettingsVariable.DefaultBiliDownloadsMaxQueueNum = DefaultBiliDownloadsMaxQueueNum
		}
		if VarSettingsVariable.DefaultTaskWorkerGoRoutines <= 0 {
			VarSettingsVariable.DefaultTaskWorkerGoRoutines = DefaultTaskWorkerGoRoutines
		}
		if VarSettingsVariable.DefaultDbCrontabSeconds <= 0 {
			VarSettingsVariable.DefaultDbCrontabSeconds = DefaultDbCrontabSeconds
		}
	} else {
		// 使用默认配置
		VarSettingsVariable = VarSettings{
			DefaultMaxThreads:               runtime.NumCPU(),
			DefaultBiliDownloadGoRoutines:   DefaultBiliDownloadGoRoutines,
			DefaultBiliDownloadsMaxQueueNum: DefaultBiliDownloadsMaxQueueNum,
			DefaultTaskWorkerGoRoutines:     DefaultTaskWorkerGoRoutines,
			DefaultDbCrontabSeconds:         DefaultDbCrontabSeconds,
		}
	}
	go DbCrontab()
}
