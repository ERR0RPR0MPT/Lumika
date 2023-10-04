package utils

import "strings"

const (
	LumikaVersionNum                = 3
	LumikaVersionString             = "v3.8.0.beta8"
	LumikaWorkDirName               = "lumika_data"
	LumikaConfigFileName            = "lumika_config"
	InitStr                         = "Init:"
	WebStr                          = "WebServer:"
	DbStr                           = "Database:"
	EnStr                           = "Encode:"
	DeStr                           = "Decode:"
	AddStr                          = "AddInput:"
	GetStr                          = "GetInput:"
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
	DefaultTaskWorkerGoRoutines     = 5
	DefaultBiliDownloadMaxRetries   = 100
	DefaultBiliDownloadReferer      = "https://www.bilibili.com"
	DefaultUserAgent                = "Mozilla/5.0 (Windows NT 10.0; WOW64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.5666.197 Safari/537.36"
	DefaultWebServerDebugMode       = false
	DefaultWebServerHost            = ""
	DefaultWebServerPort            = 7860
	DefaultWebServerRandomPortMin   = 10000
	DefaultWebServerRandomPortMax   = 65535
)

var (
	EpPath                 string
	LumikaWorkDirPath      string
	LumikaEncodePath       string
	LumikaDecodePath       string
	LumikaEncodeOutputPath string
	LumikaDecodeOutputPath string
)

type CommonError struct {
	Msg string
}

func (e *CommonError) Error() string {
	return e.Msg
}

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

type ThreadInfo struct {
	threadIndex  int
	startOffset  int64
	endOffset    int64
	tempFilePath string
}

type FileInfo struct {
	Filename  string `json:"filename"`
	ParentDir string `json:"parentDir"`
	Type      string `json:"type"`
	SizeNum   int64  `json:"sizeNum"`
	SizeStr   string `json:"sizeStr"`
	Timestamp string `json:"timestamp"`
}

type DlTaskInfo struct {
	Url        string `json:"url"`
	FileName   string `json:"fileName"`
	Referer    string `json:"referer"`
	UserAgent  string `json:"userAgent"`
	NumThreads int    `json:"numThreads"`
}

type DlTaskListData struct {
	UUID         string      `json:"uuid"`
	TimeStamp    string      `json:"timestamp"`
	FileName     string      `json:"fileName"`
	TaskInfo     *DlTaskInfo `json:"taskInfo"`
	LogCat       string      `json:"logCat"`
	ProgressRate int         `json:"progressRate"`
	ProgressNum  float64     `json:"progressNum"`
	Status       string      `json:"status"`
	StatusMsg    string      `json:"statusMsg"`
}

type BDlTaskInfo struct {
	ResourceID string `json:"resourceID"`
}

type BDlTaskListData struct {
	UUID         string       `json:"uuid"`
	TimeStamp    string       `json:"timestamp"`
	ResourceID   string       `json:"resourceId"`
	TaskInfo     *BDlTaskInfo `json:"taskInfo"`
	LogCat       string       `json:"logCat"`
	ProgressRate int          `json:"progressRate"`
	ProgressNum  float64      `json:"progressNum"`
	Status       string       `json:"status"`
	StatusMsg    string       `json:"statusMsg"`
}

type AddTaskInfo struct {
	FileNameList     []string `json:"fileNameList"`
	DefaultM         int      `json:"defaultM"`
	DefaultK         int      `json:"defaultK"`
	MGValue          int      `json:"mgValue"`
	KGValue          int      `json:"kgValue"`
	VideoSize        int      `json:"videoSize"`
	OutputFPS        int      `json:"outputFPS"`
	EncodeMaxSeconds int      `json:"encodeMaxSeconds"`
	EncodeThread     int      `json:"encodeThread"`
	EncodeFFmpegMode string   `json:"encodeFFmpegMode"`
	DefaultSummary   string   `json:"defaultSummary"`
}

type AddTaskListData struct {
	UUID         string       `json:"uuid"`
	TimeStamp    string       `json:"timestamp"`
	TaskInfo     *AddTaskInfo `json:"taskInfo"`
	LogCat       string       `json:"logCat"`
	BaseStr      string       `json:"baseStr"`
	IsPaused     bool         `json:"isPaused"`
	ProgressRate int          `json:"progressRate"`
	ProgressNum  float64      `json:"progressNum"`
	Status       string       `json:"status"`
	StatusMsg    string       `json:"statusMsg"`
}

type GetTaskInfo struct {
	DirName      string `json:"dirName"`
	DecodeThread int    `json:"decodeThread"`
	BaseStr      string `json:"baseStr"`
}

type GetTaskListData struct {
	UUID         string       `json:"uuid"`
	TimeStamp    string       `json:"timestamp"`
	TaskInfo     *GetTaskInfo `json:"taskInfo"`
	LogCat       string       `json:"logCat"`
	IsPaused     bool         `json:"isPaused"`
	ProgressRate int          `json:"progressRate"`
	ProgressNum  float64      `json:"progressNum"`
	Status       string       `json:"status"`
	StatusMsg    string       `json:"statusMsg"`
}

var LogVariable strings.Builder

var DlTaskQueue chan *DlTaskListData
var DlTaskList []*DlTaskListData

var BDlTaskQueue chan *BDlTaskListData
var BDlTaskList []*BDlTaskListData

var AddTaskQueue chan *AddTaskListData
var AddTaskList []*AddTaskListData

var GetTaskQueue chan *GetTaskListData
var GetTaskList []*GetTaskListData
