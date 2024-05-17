package common

import (
	"encoding/json"
	"io/fs"
	"strings"
)

const (
	LumikaVersionNum                 = 3
	LumikaVersionString              = "v3.15.0"
	LumikaGithubRepo                 = "err0rpr0mpt/lumika"
	LumikaWebGithubRepo              = "err0rpr0mpt/lumika-web"
	LumikaAndroidGithubRepo          = "err0rpr0mpt/lumika-android"
	LumikaWorkDirName                = "lumika_data"
	LumikaConfigFileName             = "lumika_config"
	InitStr                          = "Init:"
	UpdateStr                        = "Update:"
	WebStr                           = "WebServer:"
	DbStr                            = "Database:"
	EnStr                            = "Encode:"
	DeStr                            = "Decode:"
	AddStr                           = "AddInput:"
	GetStr                           = "GetInput:"
	DlStr                            = "Dl:"
	BDlStr                           = "BDl:"
	BUlStr                           = "BUl:"
	ArStr                            = "AutoRun:"
	ErStr                            = "Error:"
	AddMLevel                        = 90
	AddKLevel                        = 81
	AddMGLevel                       = 200
	AddKGLevel                       = 130
	EncodeVersion                    = 5
	EncodeVer5ColorGA                = 0
	EncodeVer5ColorBA                = 0
	EncodeVer5ColorGB                = 255
	EncodeVer5ColorBB                = 255
	EncodeVideoSizeLevel             = 224
	EncodeOutputFPSLevel             = 1
	EncodeMaxSecondsLevel            = 35990
	EncodeFFmpegModeLevel            = "medium"
	DefaultHashLength                = 7
	DefaultBlankSeconds              = 3
	DefaultBlankByte                 = 85
	DefaultBlankStartByte            = 86
	DefaultBlankEndByte              = 87
	DefaultDeleteFecFiles            = true
	DefaultBiliDownloadReferer       = "https://www.bilibili.com"
	DefaultBiliDownloadOrigin        = "https://www.bilibili.com"
	DefaultBiliDownloadUserAgent     = "Mozilla/5.0 (Windows NT 10.0; WOW64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.5666.197 Safari/537.36"
	DefaultDlMaxRetries              = 3
	DefaultWebServerDebugMode        = false
	DefaultWebServerHost             = ""
	DefaultWebServerPort             = 7860
	DefaultWebServerRandomPortMin    = 10000
	DefaultWebServerRandomPortMax    = 65535
	DefaultBiliDownloadMaxRetryTimes = 100
	DefaultBiliDownloadGoRoutines    = 4
	DefaultBiliDownloadsMaxQueueNum  = 25
	DefaultBiliUploadLines           = "ws"
	DefaultBiliUploadThreads         = 10
	DefaultTaskWorkerGoRoutines      = 5
	DefaultDbCrontabSeconds          = 10
)

var (
	EpDir                  string
	EpPath                 string
	LumikaWorkDirPath      string
	LumikaEncodePath       string
	LumikaDecodePath       string
	LumikaEncodeOutputPath string
	LumikaDecodeOutputPath string
	MobileMode             = false
	UISubFiles             fs.FS
)

type AndroidTaskInfo struct {
	UUID    string `json:"uuid"`
	Type    string `json:"type"`
	Command string `json:"command"`
	Output  string `json:"output"`
}

var AndroidInputTaskList map[string]*AndroidTaskInfo
var AndroidOutputTaskList map[string]*AndroidTaskInfo

// SetInput Go 进程设置任务，加入 map 中
func SetInput(uuid, tp, cmd string) {
	t := &AndroidTaskInfo{
		UUID:    uuid,
		Type:    tp,
		Command: cmd,
	}
	AndroidInputTaskList[uuid] = t
	return
}

// GetInput Android 进程获取任务，将任务从候选 map 中删除
func GetInput() (jsonString string) {
	var (
		k string
		v *AndroidTaskInfo
	)
	for k, v = range AndroidInputTaskList {
		break
	}
	if k == "" || v == nil {
		return ""
	}
	delete(AndroidInputTaskList, k)
	jsonStr, err := json.Marshal(v)
	if err != nil {
		LogPrintln("", "GetInput:", ErStr, err)
		return ""
	}
	return string(jsonStr)
}

// SetOutput Android 进程设置返回结果，加入输出任务 map 中
func SetOutput(uuid, tp, output string) {
	t := &AndroidTaskInfo{
		UUID:   uuid,
		Type:   tp,
		Output: output,
	}
	AndroidOutputTaskList[uuid] = t
	LogPrintln("", "SetOutput: 已保存:", uuid, tp, output)
}

// GetOutput Go 进程获取结果
func GetOutput(uuid string) (jsonString string) {
	v, ok := AndroidOutputTaskList[uuid]
	if !ok {
		return ""
	}
	LogPrintln("", "GetOutput: 编码进程拿到数据:", uuid, v.Type, v.Output)
	delete(AndroidOutputTaskList, uuid)
	jsonStr, err := json.Marshal(v)
	if err != nil {
		LogPrintln("", "GetOutput:", ErStr, err)
		return ""
	}
	return string(jsonStr)
}

type CommonError struct {
	Msg string
}

func (e *CommonError) Error() string {
	return e.Msg
}

type Database struct {
	DlTaskList  map[string]*DlTaskListData  `json:"dlTaskList"`
	BDlTaskList map[string]*BDlTaskListData `json:"bDlTaskList"`
	AddTaskList map[string]*AddTaskListData `json:"addTaskList"`
	GetTaskList map[string]*GetTaskListData `json:"getTaskList"`
	BUlTaskList map[string]*BUlTaskListData `json:"bUlTaskList"`
	VarSettings *VarSettings                `json:"VarSettingsVariable"`
}

type VarSettings struct {
	DefaultMaxThreads               int `json:"defaultMaxThreads"`
	DefaultBiliDownloadGoRoutines   int `json:"defaultBiliDownloadGoRoutines"`
	DefaultBiliDownloadsMaxQueueNum int `json:"defaultBiliDownloadsMaxQueueNum"`
	DefaultTaskWorkerGoRoutines     int `json:"defaultTaskWorkerGoRoutines"`
	DefaultDbCrontabSeconds         int `json:"defaultDbCrontabSeconds"`
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
	ThreadIndex  int
	StartOffset  int64
	EndOffset    int64
	TempFilePath string
}

type FileInfo struct {
	Filename  string `json:"filename"`
	ParentDir string `json:"parentDir"`
	Type      string `json:"type"`
	SizeNum   int64  `json:"sizeNum"`
	SizeStr   string `json:"sizeStr"`
	Timestamp string `json:"timestamp"`
}

type SystemResourceUsage struct {
	OSName                string  `json:"osName"`
	ExecuteTime           string  `json:"executeTime"`
	CpuUsagePercent       float64 `json:"cpuUsagePercent"`
	MemUsageTotalAndUsed  string  `json:"memUsageTotalAndUsed"`
	MemUsagePercent       float64 `json:"memUsagePercent"`
	DiskUsageTotalAndUsed string  `json:"diskUsageTotalAndUsed"`
	DiskUsagePercent      float64 `json:"diskUsagePercent"`
	NetworkInterfaceName  string  `json:"networkInterfaceName"`
	UploadSpeed           string  `json:"uploadSpeed"`
	DownloadSpeed         string  `json:"downloadSpeed"`
	UploadTotal           string  `json:"uploadTotal"`
	DownloadTotal         string  `json:"downloadTotal"`
}

type DlTaskInfo struct {
	Url        string `json:"url"`
	FileName   string `json:"fileName"`
	Referer    string `json:"referer"`
	Origin     string `json:"origin"`
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
	Duration     string      `json:"duration"`
}

type BDlTaskInfo struct {
	ResourceID string `json:"resourceID"`
	ParentDir  string `json:"parentDir"`
	BaseStr    string `json:"baseStr"`
}

type BDlTaskListData struct {
	UUID         string       `json:"uuid"`
	TimeStamp    string       `json:"timestamp"`
	ResourceID   string       `json:"resourceId"`
	TaskInfo     *BDlTaskInfo `json:"taskInfo"`
	BaseStr      string       `json:"baseStr"`
	LogCat       string       `json:"logCat"`
	ProgressRate int          `json:"progressRate"`
	ProgressNum  float64      `json:"progressNum"`
	Status       string       `json:"status"`
	StatusMsg    string       `json:"statusMsg"`
	Duration     string       `json:"duration"`
}

type AddTaskInfo struct {
	FileNameList      []string `json:"fileNameList"`
	DefaultM          int      `json:"defaultM"`
	DefaultK          int      `json:"defaultK"`
	MGValue           int      `json:"mgValue"`
	KGValue           int      `json:"kgValue"`
	VideoSize         int      `json:"videoSize"`
	OutputFPS         int      `json:"outputFPS"`
	EncodeMaxSeconds  int      `json:"encodeMaxSeconds"`
	EncodeThread      int      `json:"encodeThread"`
	EncodeVersion     int      `json:"encodeVersion"`
	EncodeVer5ColorGA int      `json:"encodeVer5ColorGA"`
	EncodeVer5ColorBA int      `json:"encodeVer5ColorBA"`
	EncodeVer5ColorGB int      `json:"encodeVer5ColorGB"`
	EncodeVer5ColorBB int      `json:"encodeVer5ColorBB"`
	EncodeFFmpegMode  string   `json:"encodeFFmpegMode"`
	DefaultSummary    string   `json:"defaultSummary"`
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
	Duration     string       `json:"duration"`
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
	Filename     string       `json:"filename"`
	ProgressRate int          `json:"progressRate"`
	ProgressNum  float64      `json:"progressNum"`
	Status       string       `json:"status"`
	StatusMsg    string       `json:"statusMsg"`
	Duration     string       `json:"duration"`
}

type biliUpUser struct {
	SESSDATA        string `json:"SESSDATA"`
	BiliJct         string `json:"bili_jct"`
	DedeUserID      string `json:"DedeUserID"`
	DedeuseridCkmd5 string `json:"DedeUserID__ckMd5"`
	AccessToken     string `json:"access_token"`
}

type biliUpVideoInfos struct {
	Tid         int      `json:"tid"`
	Title       string   `json:"title"`
	Aid         string   `json:"aid,omitempty"`
	Tag         []string `json:"tag,omitempty"`
	Source      string   `json:"source,omitempty"`
	Cover       string   `json:"cover,omitempty"`
	CoverPath   string   `json:"cover_path,omitempty"`
	Description string   `json:"description,omitempty"`
	Copyright   int      `json:"copyright,omitempty"`
}

type BUlTaskInfo struct {
	FileName    string           `json:"fileName"`
	Cookie      *biliUpUser      `json:"cookie"`
	UploadLines string           `json:"uploadLines"`
	Threads     int              `json:"threads"`
	VideoInfos  biliUpVideoInfos `json:"videoInfos"`
}

type BUlTaskListData struct {
	UUID         string       `json:"uuid"`
	TimeStamp    string       `json:"timestamp"`
	FileName     string       `json:"fileName"`
	BVID         string       `json:"bvid"`
	TaskInfo     *BUlTaskInfo `json:"taskInfo"`
	LogCat       string       `json:"logCat"`
	ProgressRate int          `json:"progressRate"`
	ProgressNum  float64      `json:"progressNum"`
	Status       string       `json:"status"`
	StatusMsg    string       `json:"statusMsg"`
	Duration     string       `json:"duration"`
}

var StartTimestamp int64

var DatabaseVariable Database

var VarSettingsVariable VarSettings

var LogVariable strings.Builder

var DlTaskQueue chan *DlTaskListData
var DlTaskList map[string]*DlTaskListData

var BDlTaskQueue chan *BDlTaskListData
var BDlTaskList map[string]*BDlTaskListData

var AddTaskQueue chan *AddTaskListData
var AddTaskList map[string]*AddTaskListData

var GetTaskQueue chan *GetTaskListData
var GetTaskList map[string]*GetTaskListData

var BUlTaskQueue chan *BUlTaskListData
var BUlTaskList map[string]*BUlTaskListData

var UploadTotalStart int64 = -1
var DownloadTotalStart int64 = -1
