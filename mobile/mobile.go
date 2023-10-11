package mobile

import (
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/ERR0RPR0MPT/Lumika/utils"
)

func StartWebServer(port int, dataPath string) {
	utils.LumikaDataPathInit(dataPath)
	utils.WebServer("", port)
}

func SetInput(input string) {
	common.MobileInput = input
}

func GetInput() string {
	return common.MobileInput
}

func SetOutput(output string) {
	common.MobileOutput = output
}

func GetOutput() string {
	return common.MobileOutput
}
