package mobile

import (
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/ERR0RPR0MPT/Lumika/utils"
)

func StartWebServer(port int, dataPath string) {
	common.MobileMode = true
	utils.LumikaDataPathInit(dataPath)
	common.AndroidInputTaskList = make(map[string]*common.AndroidTaskInfo)
	common.AndroidOutputTaskList = make(map[string]*common.AndroidTaskInfo)
	utils.WebServer("", port)
}

// GetInput 暴露接口
func GetInput() (jsonString string) {
	return common.GetInput()
}

// SetOutput 暴露接口
func SetOutput(uuid, tp, output string) {
	common.SetOutput(uuid, tp, output)
}
