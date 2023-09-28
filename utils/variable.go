package utils

const (
	LumikaVersionNum                = 3
	LumikaVersionString             = "v3.8.0.beta3"
	LumikaWorkDirName               = "lumika_data"
	InitStr                         = "Init:"
	WebStr                          = "WebServer:"
	DbStr                           = "Database:"
	EnStr                           = "Encode:"
	DeStr                           = "Decode:"
	AddStr                          = "Add:"
	GetStr                          = "Get:"
	BDlStr                          = "BDl:"
	ArStr                           = "AutoRun:"
	ErStr                           = "Error:"
	AddMLevel                       = 90
	AddKLevel                       = 81
	AddMGLevel                      = 200
	AddKGLevel                      = 130
	EncodeVideoSizeLevel            = 32
	EncodeOutputFPSLevel            = 24
	EncodeMaxSecondsLevel           = 35990
	EncodeFFmpegModeLevel           = "medium"
	DefaultHashLength               = 7
	DefaultBlankSeconds             = 3
	DefaultBlankByte                = 85
	DefaultBlankStartByte           = 86
	DefaultBlankEndByte             = 87
	DefaultDeleteFecFiles           = true
	DefaultBiliDownloadGoRoutines   = 20
	DefaultBiliDownloadsMaxQueueNum = 5
	DefaultBiliDownloadMaxRetries   = 100
	DefaultBiliDownloadReferer      = "https://www.bilibili.com"
	DefaultUserAgent                = "Mozilla/5.0 (Windows NT 10.0; WOW64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.5666.197 Safari/537.36"
	DefaultWebServerDebugMode       = false
	DefaultWebServerBindAddress     = ":7860"
)

type FecFileConfig struct {
	Version       int      `json:"v"`
	Name          string   `json:"n"`
	Summary       string   `json:"s"`
	Hash          string   `json:"h"`
	M             int      `json:"m"`
	K             int      `json:"k"`
	MG            int      `json:"mg"`
	KG            int      `json:"kg"`
	Length        int64    `json:"l"`
	SegmentLength int64    `json:"sl"`
	FecHashList   []string `json:"fhl"`
}

type DlTaskListData struct {
	UUID         string `json:"uuid"`
	Type         string `json:"type"`
	TimeStamp    string `json:"timestamp"`
	ResourceID   string `json:"resourceId"`
	FileName     string `json:"fileName"`
	ProgressRate int    `json:"progressRate"`
}

type FileInfo struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
}

var DlTaskQueue chan *DlTaskInfo
var DlTaskList []DlTaskListData
var DirectoryData []FileInfo
