package utils

const (
	LumikaVersionNum      = 3
	LumikaVersionString   = "v3.7.3"
	EnStr                 = "Encode:"
	DeStr                 = "Decode:"
	AddStr                = "Add:"
	GetStr                = "Get:"
	ArStr                 = "AutoRun:"
	ErStr                 = "Error:"
	AddMLevel             = 90
	AddKLevel             = 81
	AddMGLevel            = 200
	AddKGLevel            = 130
	EncodeVideoSizeLevel  = 32
	EncodeOutputFPSLevel  = 24
	EncodeMaxSecondsLevel = 35990
	EncodeFFmpegModeLevel = "medium"
	DefaultHashLength     = 7
	DefaultBlankSeconds   = 3
	DefaultBlankByte      = 85
	DefaultBlankStartByte = 86
	DefaultBlankEndByte   = 87
	DefaultDeleteFecFiles = true
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
