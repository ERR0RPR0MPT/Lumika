package utils

import (
	"encoding/json"
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func DbSave(wd string, i int) {
	db := &common.Database{
		DlTaskList:  common.DlTaskList,
		BDlTaskList: common.BDlTaskList,
		AddTaskList: common.AddTaskList,
		GetTaskList: common.GetTaskList,
		BUlTaskList: common.BUlTaskList,
		VarSettings: &common.VarSettingsVariable,
	}
	jsonData, err := json.Marshal(db)
	if err != nil {
		common.LogPrintln("", common.DbStr, "转换为JSON时发生错误:", err)
		return
	}
	err = os.WriteFile(wd, jsonData, 0755)
	if err != nil {
		common.LogPrintln("", common.DbStr, "保存JSON文件时发生错误:", err)
		return
	}
	if i%360 == 0 {
		common.LogPrintln("", common.DbStr, "数据库已保存")
	}
}

func DbCrontab() {
	// 每隔 DefaultDbCrontabSeconds 秒存储一次数据库
	wd := filepath.Join(common.LumikaWorkDirPath, "db.json")
	// 每次启动前等待 5s
	time.Sleep(time.Second * 5)
	i := 0
	for {
		DbSave(wd, i)
		time.Sleep(time.Second * time.Duration(common.VarSettingsVariable.DefaultDbCrontabSeconds))
		i++
	}
}

func DbInit() {
	common.LogPrintln("", common.DbStr, common.InitStr, "初始化数据库")
	wd := filepath.Join(common.LumikaWorkDirPath, "db.json")
	if _, err := os.Stat(wd); err == nil {
		common.LogPrintln("", common.DbStr, common.InitStr, "读取数据库文件")
		jsonData, err := os.ReadFile(wd)
		if err != nil {
			fmt.Println("读取JSON文件时发生错误:", err)
			return
		}
		err = json.Unmarshal(jsonData, &common.DatabaseVariable)
		if err != nil {
			fmt.Println("解析JSON数据时发生错误:", err)
			return
		}
		// 更新用户配置
		if common.DatabaseVariable.VarSettings == nil {
			// 使用默认配置
			common.DatabaseVariable.VarSettings = &common.VarSettings{
				DefaultMaxThreads:               runtime.NumCPU(),
				DefaultBiliDownloadGoRoutines:   common.DefaultBiliDownloadGoRoutines,
				DefaultBiliDownloadsMaxQueueNum: common.DefaultBiliDownloadsMaxQueueNum,
				DefaultTaskWorkerGoRoutines:     common.DefaultTaskWorkerGoRoutines,
				DefaultDbCrontabSeconds:         common.DefaultDbCrontabSeconds,
			}
		}
		common.VarSettingsVariable = *common.DatabaseVariable.VarSettings
		if common.VarSettingsVariable.DefaultMaxThreads <= 0 {
			common.VarSettingsVariable.DefaultMaxThreads = runtime.NumCPU()
		}
		if common.VarSettingsVariable.DefaultBiliDownloadGoRoutines <= 0 {
			common.VarSettingsVariable.DefaultBiliDownloadGoRoutines = common.DefaultBiliDownloadGoRoutines
		}
		if common.VarSettingsVariable.DefaultBiliDownloadsMaxQueueNum <= 0 {
			common.VarSettingsVariable.DefaultBiliDownloadsMaxQueueNum = common.DefaultBiliDownloadsMaxQueueNum
		}
		if common.VarSettingsVariable.DefaultTaskWorkerGoRoutines <= 0 {
			common.VarSettingsVariable.DefaultTaskWorkerGoRoutines = common.DefaultTaskWorkerGoRoutines
		}
		if common.VarSettingsVariable.DefaultDbCrontabSeconds <= 0 {
			common.VarSettingsVariable.DefaultDbCrontabSeconds = common.DefaultDbCrontabSeconds
		}
	} else {
		// 使用默认配置
		common.VarSettingsVariable = common.VarSettings{
			DefaultMaxThreads:               runtime.NumCPU(),
			DefaultBiliDownloadGoRoutines:   common.DefaultBiliDownloadGoRoutines,
			DefaultBiliDownloadsMaxQueueNum: common.DefaultBiliDownloadsMaxQueueNum,
			DefaultTaskWorkerGoRoutines:     common.DefaultTaskWorkerGoRoutines,
			DefaultDbCrontabSeconds:         common.DefaultDbCrontabSeconds,
		}
	}
	go DbCrontab()
}
