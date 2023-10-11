package mobile

import (
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/ERR0RPR0MPT/Lumika/utils"
)

func StartWebServer(port int, dataPath string, ffmpegPath string, ffprobePath string) {
	common.MobileFFmpegPath = ffmpegPath
	common.MobileFFprobePath = ffprobePath
	utils.LumikaDataPathInit(dataPath)
	utils.WebServer("", port)
}
