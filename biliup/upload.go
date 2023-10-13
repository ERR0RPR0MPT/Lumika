package biliup

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/common"
	"github.com/google/go-querystring/query"
	"github.com/tidwall/gjson"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ChunkSize = 10485760

const (
	Auto        = "auto"
	Cos         = "cos"
	CosInternal = "cos-internal"
	Bda2        = "bda2"
	Ws          = "ws"
	Qn          = "qn"
)

var BilibiliLines = []string{Auto, Cos, CosInternal, Bda2, Ws, Qn}

var DefaultHeader = http.Header{
	"User-Agent": []string{common.DefaultBiliDownloadUserAgent},
	"Referer":    []string{common.DefaultBiliDownloadReferer},
	"Connection": []string{"keep-alive"},
}

const Name = "bilibili"
const BilibiliMaxRetryTimes = 100
const ChunkUploadMaxRetryTimes = 100 // 上传分片最大重试次数
type Bvid string

type Bilibili struct {
	User        User   `json:"user"`
	Lives       string `json:"url,omitempty"`
	UploadLines string `json:"upload_lines,omitempty"`
	Threads     int    `json:"threads,omitempty"`
	WithCover   bool   `json:"with_cover,omitempty"`
	CheckParams bool   `json:"check_params,omitempty"`
	AutoFix     bool   `json:"auto_fix,omitempty"`
	Header      http.Header
	Client      http.Client
	VideoInfos
}
type SubmitRes struct {
	Aid  int    `json:"aid"`
	Bvid string `json:"bvid"`
}

func (b *Bilibili) SetVideoInfos(v interface{}) error {
	info, ok := v.(VideoInfos)
	if !ok {
		return errors.New("not A Bilibili VideoInfos")
	}
	b.VideoInfos = info
	return nil
}
func (b *Bilibili) SetThreads(Thread uint) {
	b.Threads = int(Thread)
}
func (b *Bilibili) SetUploadLine(uploadLine string) {
	if !InArray(BilibiliLines, uploadLine) {
		b.UploadLines = Auto
		common.LogPrintf("", common.BUlStr, "upload line %s not support,Support line are %s,set uploadline to AUTO", uploadLine, BilibiliLines)
	} else {
		b.UploadLines = uploadLine
	}
}
func New(u User) (*Bilibili, error) {
	jar, _ := cookiejar.New(nil)
	b := Bilibili{
		User:        u,
		Threads:     3,
		UploadLines: Auto,
		Header:      DefaultHeader,
		Client: http.Client{
			Jar:     jar,
			Timeout: time.Second * 5,
		},
	}
	err := CookieLoginCheck(u, &b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

type VideoInfos struct {
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
type User struct {
	SESSDATA        string `json:"SESSDATA"`
	BiliJct         string `json:"bili_jct"`
	DedeUserID      string `json:"DedeUserID"`
	DedeuseridCkmd5 string `json:"DedeUserID__ckMd5"`
	AccessToken     string `json:"access_token"`
}

type TokenInfo struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	Mid          int    `json:"mid"`
	RefreshToken string `json:"refresh_token"`
}

type UploadedVideoInfo struct {
	title    string
	filename string
	desc     string
}
type uploadOs struct {
	Os       string `json:"os"`
	Query    string `json:"query"`
	ProbeUrl string `json:"probe_url"`
}

var defaultOs = uploadOs{
	Os:       "upos",
	Query:    "upcdn=bda2&probe_version=20211012",
	ProbeUrl: "//upos-sz-upcdnbda2.bilivideo.com/OK",
}

type UploadedFile struct {
	FilePath string
	FileName string
}

func NewUser(SESSDATA, BiliJct, DedeUserId, DedeUseridCkmd5, AccessToken string) User {
	return User{
		SESSDATA:        SESSDATA,
		BiliJct:         BiliJct,
		DedeUserID:      DedeUserId,
		DedeuseridCkmd5: DedeUseridCkmd5,
		AccessToken:     AccessToken,
	}
}

func CookieLoginCheck(u User, b *Bilibili) error {
	cookie := []*http.Cookie{{Name: "SESSDATA", Value: u.SESSDATA},
		{Name: "DedeUserID", Value: u.DedeUserID},
		{Name: "DedeUserID__ckMd5", Value: u.DedeuseridCkmd5},
		{Name: "bili_jct", Value: u.BiliJct}}
	urlObj, _ := url.Parse("https://api.bilibili.com")
	b.Client.Jar.SetCookies(urlObj, cookie)
	apiUrl := "https://api.bilibili.com/x/web-interface/nav"
	req, _ := http.NewRequest("GET", apiUrl, nil)
	res, err := b.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var t struct {
		Code int `json:"code"`
	}
	err = json.Unmarshal(body, &t)
	if err != nil {
		return fmt.Errorf("b站返回无法解析的数据: %s", string(body))
	}
	if t.Code != 0 {
		return errors.New("cookie login failed")
	}
	urlObj, _ = url.Parse("https://member.bilibili.com")
	b.Client.Jar.SetCookies(urlObj, cookie)
	return nil
}
func selectUploadOs(lines string) uploadOs {
	var o uploadOs
	if lines == Auto {
		res, err := http.Get("https://member.bilibili.com/preupload?r=probe")
		if err != nil {
			return defaultOs
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		lineInfo := struct {
			Ok    int        `json:"OK"`
			Lines []uploadOs `json:"lines"`
		}{}
		_ = json.Unmarshal(body, &lineInfo)
		if lineInfo.Ok != 1 {
			return defaultOs
		}
		fastestLine := make(chan uploadOs, 1)
		timer := time.NewTimer(time.Second * 10)
		for _, line := range lineInfo.Lines {
			line := line
			go func() {
				res, _ := http.Get("https" + line.ProbeUrl)
				if res.StatusCode == 200 {
					fastestLine <- line
				}
			}()
		}
		select {
		case <-timer.C:
			return defaultOs
		case line := <-fastestLine:
			return line
		}
	} else {
		if lines == Bda2 {
			o = uploadOs{
				Os:       "upos",
				Query:    "upcdn=bda2&probe_version=20211012",
				ProbeUrl: "//upos-sz-upcdnbda2.bilivideo.com/OK",
			}
		} else if lines == Ws {
			o = uploadOs{
				Os:       "upos",
				Query:    "upcdn=ws&probe_version=20211012",
				ProbeUrl: "//upos-sz-upcdnws.bilivideo.com/OK",
			}
		} else if lines == Qn {
			o = uploadOs{
				Os:       "upos",
				Query:    "upcdn=qn&probe_version=20211012",
				ProbeUrl: "//upos-sz-upcdnqn.bilivideo.com/OK",
			}
		} else if lines == Cos {
			o = uploadOs{
				Os:       "cos",
				Query:    "",
				ProbeUrl: "",
			}
		} else if lines == CosInternal {
			o = uploadOs{
				Os:       "cos-internal",
				Query:    "",
				ProbeUrl: "",
			}
		}
	}
	return o
}

func (b *Bilibili) UploadFile(file *os.File, UUID string) (*UploadRes, error) {
	upOs := selectUploadOs(b.UploadLines)
	state, _ := file.Stat()
	q := struct {
		R       string `url:"r"`
		Profile string `url:"profile"`
		Ssl     int    `url:"ssl"`
		Version string `url:"version"`
		Build   int    `url:"build"`
		Name    string `url:"name"`
		Size    int    `url:"size"`
	}{
		Ssl:     0,
		Version: "2.8.1.2",
		Build:   2081200,
		Name:    filepath.Base(file.Name()),
		Size:    int(state.Size()),
	}
	if upOs.Os == "cos-internal" {
		q.R = "cos"
	} else {
		q.R = upOs.Os
	}
	if upOs.Os == "upos" {
		q.Profile = "ugcupos/bup"
	} else {
		q.Profile = "ugcupos/bupfetch"
	}
	v, _ := query.Values(q)

	req, _ := http.NewRequest("GET", "https://member.bilibili.com/preupload?"+upOs.Query+v.Encode(), nil)
	var content []byte

	for i := 0; i < BilibiliMaxRetryTimes; i++ {
		res, err := b.Client.Do(req)
		if err != nil {
			res.Body.Close()
			common.LogPrintln(UUID, common.BUlStr, "preupload 数据获取失败, 正在重试第", i, "次:", file.Name())
			time.Sleep(time.Second * 1)
			continue
		}
		content, err = io.ReadAll(res.Body)
		res.Body.Close()
		if err == nil {
			common.LogPrintln(UUID, common.BUlStr, "preupload 数据获取成功:", file.Name())
			break
		}
	}
	if content == nil {
		common.LogPrintln(UUID, common.BUlStr, "preupload 数据获取失败:", file.Name())
		return &UploadRes{}, errors.New("preupload 数据获取失败")
	}
	if upOs.Os == "cos-internal" || upOs.Os == "cos" {
		var internal bool
		if upOs.Os == "cos-internal" {
			internal = true
		}
		body := &cosUploadSegments{}
		_ = json.Unmarshal(content, &body)
		if body.Ok != 1 {
			common.LogPrintln(UUID, common.BUlStr, "query Upload Parameters failed:", file.Name())
			return &UploadRes{}, errors.New("query Upload Parameters failed")
		}
		common.LogPrintln(UUID, common.BUlStr, "使用 cos 方式上传文件:", file.Name())
		var videoInfo *UploadRes
		var err error
		for i := 0; i < 10; i++ {
			videoInfo, err = cos(file, int(state.Size()), body, internal, ChunkSize, b.Threads, UUID)
			if err != nil {
				common.LogPrintln(UUID, common.BUlStr, file.Name(), "上传失败(cos)，正在重试第", i, "次:", file.Name())
				continue
			}
			break
		}
		return videoInfo, err

	} else if upOs.Os == "upos" {
		body := &uposUploadSegments{}
		_ = json.Unmarshal(content, &body)
		if body.Ok != 1 {
			common.LogPrintln(UUID, common.BUlStr, "query UploadFile failed:", file.Name())
			return &UploadRes{}, errors.New("query UploadFile failed")
		}
		common.LogPrintln(UUID, common.BUlStr, "使用 upos 方式上传文件:", file.Name())
		var videoInfo *UploadRes
		var err error
		for i := 0; i < 10; i++ {
			videoInfo, err = upos(file, int(state.Size()), body, b.Threads, b.Header, UUID)
			if err != nil {
				common.LogPrintln(UUID, common.BUlStr, file.Name(), "上传失败(upos)，正在重试第", i, "次:", file.Name())
				continue
			}
			break
		}
		return videoInfo, err
	}
	common.LogPrintln(UUID, common.BUlStr, "未知的上传线路")
	return &UploadRes{}, errors.New("未知的上传线路")
}

func (b *Bilibili) FolderUpload(folder string, UUID string) ([]*UploadRes, []UploadedFile, error) {
	dir, err := os.ReadDir(folder)
	if err != nil {
		common.LogPrintf(UUID, common.BUlStr, "目录读取失败:%s", err)
		return nil, nil, err
	}
	var uploadedFiles []UploadedFile
	var submitFiles []*UploadRes
	uploadedFilesNum := 0

	for _, file := range dir {
		for i := 0; i < BilibiliMaxRetryTimes; i++ {
			filename := filepath.Join(folder, file.Name())
			uploadFile, err := os.Open(filename)
			if err != nil {
				common.LogPrintf(UUID, common.BUlStr, "打开文件 %s 出现错误:%s", filename, err)
				break
			}
			common.LogPrintln(UUID, common.BUlStr, "开始上传文件:", file.Name())
			videoPart, err := b.UploadFile(uploadFile, UUID)
			if err != nil {
				common.LogPrintf(UUID, common.BUlStr, "UploadFile file error:%s", err)
				common.LogPrintln(UUID, common.BUlStr, "上传文件失败，正在重试第", i, "次:", file.Name())
				uploadFile.Close()
				continue
			}

			if UUID != "" {
				_, exist := common.BUlTaskList[UUID]
				if exist {
					common.BUlTaskList[UUID].ProgressRate++
					common.BUlTaskList[UUID].ProgressNum = float64(common.BUlTaskList[UUID].ProgressRate) / float64(len(dir)) * 100
				} else {
					common.LogPrintln(UUID, common.BUlStr, common.ErStr, "当前任务被用户删除", err)
					return nil, nil, &common.CommonError{Msg: "当前任务被用户删除"}
				}
			}

			uploadedFilesNum++
			common.LogPrintln(UUID, common.BUlStr, "已成功上传第", uploadedFilesNum, "个文件:", file.Name())
			uploadedFiles = append(uploadedFiles, UploadedFile{
				FilePath: folder,
				FileName: file.Name(),
			})
			submitFiles = append(submitFiles, videoPart)
			uploadFile.Close()
			break
		}
	}
	return submitFiles, uploadedFiles, nil
}

func UploadFolderWithSubmit(uploadPath string, Biliup Bilibili, UUID string) (reqBody interface{}, uf []UploadedFile, err error) {
	var submitFiles []*UploadRes
	if !filepath.IsAbs(uploadPath) {
		pwd, _ := os.Getwd()
		uploadPath = filepath.Join(pwd, uploadPath)
	}
	common.LogPrintf(UUID, common.BUlStr, "开始上传目录到哔哩源:", uploadPath)
	submitFiles, uploadedFile, err := Biliup.FolderUpload(uploadPath, UUID)
	if err != nil {
		common.LogPrintf(UUID, common.BUlStr, "视频上传失败:%s", err)
		return "", nil, err
	}
	// 避免同一时间提交多个稿件
	isSuccess := false
	for i := 0; i < BilibiliMaxRetryTimes; i++ {
		reqBody, err = Biliup.Submit(submitFiles)
		if err != nil {
			rand.Seed(time.Now().UnixNano())
			delayTime := rand.Intn(60) + 1
			common.LogPrintf(UUID, common.BUlStr, "视频提交失败，准备在", delayTime, "秒后重试:%s", err)
			time.Sleep(time.Duration(delayTime) * time.Second)
			continue
		}
		isSuccess = true
		break
	}
	if !isSuccess {
		return "", nil, err
	}
	return reqBody, uploadedFile, nil
}

func (b *Bilibili) GetBiliCoverUrl(base64 string) (string, error) {
	// 对传入参数进行urlCode编码
	urlCode := url.QueryEscape(base64)
	timeStamp := time.Now().UnixMilli()
	urla := "https://member.bilibili.com/x/vu/web/cover/up?t=" + strconv.FormatInt(timeStamp, 10)
	method := "POST"
	payload := strings.NewReader("cover=" + urlCode + "&csrf=" + b.User.AccessToken)
	req, err := http.NewRequest(method, urla, payload)
	if err != nil {
		return "", err
	}
	res, err := b.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return gjson.GetBytes(body, "data.urla").String(), nil
}
