package biliup

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-querystring/query"
	"github.com/tidwall/gjson"
	"io"
	"log"
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
	"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/63.0.3239.108"},
	"Referer":    []string{"https://www.bilibili.com"},
	"Connection": []string{"keep-alive"},
}

const Name = "bilibili"
const BilibiliMaxRetryTimes = 10
const ChunkUploadMaxRetryTimes = 10 // 上传分片最大重试次数
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
		fmt.Printf("upload line %s not support,Support line are %s,set uploadline to AUTO", uploadLine, BilibiliLines)
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

func (b *Bilibili) UploadFile(file *os.File) (*UploadRes, error) {
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
			time.Sleep(time.Second * 1)
			continue
		}
		content, err = io.ReadAll(res.Body)
		res.Body.Close()
		if err == nil {
			break
		}
	}
	if content == nil {
		return &UploadRes{}, errors.New("preupload query failed")
	}
	if upOs.Os == "cos-internal" || upOs.Os == "cos" {
		var internal bool
		if upOs.Os == "cos-internal" {
			internal = true
		}
		body := &cosUploadSegments{}
		_ = json.Unmarshal(content, &body)
		if body.Ok != 1 {
			return &UploadRes{}, errors.New("query Upload Parameters failed")
		}
		videoInfo, err := cos(file, int(state.Size()), body, internal, ChunkSize, b.Threads)
		return videoInfo, err

	} else if upOs.Os == "upos" {
		body := &uposUploadSegments{}
		_ = json.Unmarshal(content, &body)
		if body.Ok != 1 {
			return &UploadRes{}, errors.New("query UploadFile failed")
		}
		videoInfo, err := upos(file, int(state.Size()), body, b.Threads, b.Header)
		return videoInfo, err
	}
	return &UploadRes{}, errors.New("unknown upload os")
}

func (b *Bilibili) FolderUpload(folder string) ([]*UploadRes, []UploadedFile, error) {
	dir, err := os.ReadDir(folder)
	if err != nil {
		fmt.Printf("read dir error:%s", err)
		return nil, nil, err
	}
	var uploadedFiles []UploadedFile
	var submitFiles []*UploadRes
	for _, file := range dir {
		filename := filepath.Join(folder, file.Name())
		uploadFile, err := os.Open(filename)
		if err != nil {
			log.Printf("open file %s error:%s", filename, err)
			continue
		}
		videoPart, err := b.UploadFile(uploadFile)
		if err != nil {
			log.Printf("UploadFile file error:%s", err)
			uploadFile.Close()
			continue
		}
		uploadedFiles = append(uploadedFiles, UploadedFile{
			FilePath: folder,
			FileName: file.Name(),
		})
		submitFiles = append(submitFiles, videoPart)
		uploadFile.Close()
	}
	return submitFiles, uploadedFiles, nil
}
func UploadFolderWithSubmit(uploadPath string, Biliup Bilibili) (reqBody interface{}, uf []UploadedFile, err error) {
	var submitFiles []*UploadRes
	if !filepath.IsAbs(uploadPath) {
		pwd, _ := os.Getwd()
		uploadPath = filepath.Join(pwd, uploadPath)
	}
	fmt.Println(uploadPath)
	submitFiles, uploadedFile, err := Biliup.FolderUpload(uploadPath)
	if err != nil {
		fmt.Printf("UploadFile file error:%s", err)
		return "", nil, err
	}
	reqBody, err = Biliup.Submit(submitFiles)
	if err != nil {
		fmt.Printf("Submit file error:%s", err)
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
