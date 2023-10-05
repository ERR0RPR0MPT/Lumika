package biliup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

type uposUploadSegments struct {
	Ok        int    `json:"OK"`
	Auth      string `json:"auth"`
	BizID     int    `json:"biz_id"`
	ChunkSize int    `json:"chunk_size"`
	Endpoint  string `json:"endpoint"`
	Uip       string `json:"uip"`
	UposURI   string `json:"upos_uri"`
}
type preUploadJson struct {
	UploadID string `json:"upload_id"`
	Bucket   string `json:"bucket"`
	Ok       int    `json:"OK"`
	Key      string `json:"key"`
}
type uploadParam struct {
	Name     string `url:"name"`
	UploadId string `url:"uploadId"`
	BizID    int    `url:"biz_id"`
	Output   string `url:"output"`
	Profile  string `url:"profile"`
}
type partsInfo struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"eTag"`
}
type partsJson struct {
	Parts []partsInfo `json:"parts"`
}

func upos(file *os.File, totalSize int, ret *uposUploadSegments, Threads int, Header http.Header) (*UploadRes, error) {
	uploadUrl := "https:" + ret.Endpoint + "/" + strings.TrimPrefix(ret.UposURI, "upos://")
	client := &http.Client{}
	client.Timeout = time.Second * 5
	req, err := http.NewRequest("POST", uploadUrl+"?uploads&output=json", nil)
	req.Header = Header.Clone()
	req.Header.Add("X-Upos-Auth", ret.Auth)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	t := preUploadJson{}
	_ = json.Unmarshal(body, &t)

	uploader := &chunkUploader{
		uploadId:     t.UploadID,
		chunks:       int(math.Ceil(float64(totalSize) / float64(ret.ChunkSize))),
		chunkSize:    ret.ChunkSize,
		totalSize:    totalSize,
		threads:      Threads,
		url:          uploadUrl,
		chunkInfo:    make(chan chunkInfo, int(math.Ceil(float64(totalSize)/float64(ChunkSize)))+10),
		uploadMethod: "upos",
		file:         file,
		Header:       req.Header,
		MaxThread:    make(chan struct{}, Threads),
		//waitGoroutine: sync.WaitGroup{},
	}

	err = uploader.upload()
	if err != nil {
		return nil, err
	}
	part := partsJson{}
	for i := 0; i < uploader.chunks; i++ {
		c := <-uploader.chunkInfo
		index := c.Order
		part.Parts = append(part.Parts, partsInfo{
			PartNumber: index,
			ETag:       "etag",
		})
	}
	jsonPart, _ := json.Marshal(part)
	params := &uploadParam{
		Name:     filepath.Base(file.Name()),
		UploadId: t.UploadID,
		BizID:    ret.BizID,
		Output:   "json",
		Profile:  "ugcupos/bup",
	}
	p, _ := query.Values(params)
	for i := 0; ; i++ {
		req, _ := http.NewRequest("POST", uploadUrl, bytes.NewBuffer(jsonPart))
		req.URL.RawQuery = p.Encode()
		client := &http.Client{}
		req.Header.Add("X-Upos-Auth", ret.Auth)
		res, err := client.Do(req)
		if err != nil {
			log.Println(err, file.Name(), "第", i, "次请求合并失败，正在重试")
			if i == 10 {
				log.Println(err, file.Name(), "第10次请求合并失败")
				return nil, errors.New(fmt.Sprintln(file.Name(), "第10次请求合并失败", err))
			}
			time.Sleep(time.Second * 15)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		t := struct {
			Ok int `json:"OK"`
		}{}
		_ = json.Unmarshal(body, &t)
		res.Body.Close()
		if t.Ok == 1 {
			_, uposFile := filepath.Split(ret.UposURI)
			upRes := &UploadRes{
				Title:    strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name())),
				Filename: strings.TrimSuffix(filepath.Base(uposFile), filepath.Ext(uposFile)),
				Desc:     "",
			}
			return upRes, nil
		} else {
			log.Println(file.Name(), "第", i, "次请求合并失败，正在重试")
			if i == 10 {
				log.Println(file.Name(), "第10次请求合并失败")
				return nil, errors.New("分片上传失败")
			}
			time.Sleep(time.Second * 15)
		}
	}

}
