package biliup

import (
	"github.com/ERR0RPR0MPT/Lumika/common"
	"io"

	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type cosUploadSegments struct {
	Ok           int             `json:"ok"`
	BizID        int             `json:"biz_id"`
	PostAuth     string          `json:"post_auth"`
	PutAuth      string          `json:"put_auth"`
	FetchUrl     string          `json:"fetch_url"`
	BiliFilename string          `json:"bili_filename"`
	FetchHeaders cosFetchHeaders `json:"fetch_headers"`
	Url          string          `json:"url"`
}
type cosFetchHeaders struct {
	FetchHeaderAuthorization string `json:"Fetch-Header-Authorization"`
	XUposAuth                string `json:"X-Upos-Auth"`
	XUposFetchSource         string `json:"X-Upos-Fetch-Source"`
}
type cosFetchInfo struct {
	Url           string `json:"url"`
	Authorization string `json:"Authorization"`
}
type cosPreUploadXmlRes struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}
type partsXml struct {
	XMLName xml.Name `xml:"CompleteMultipartUpload"`
	Part    []struct {
		XMLName    xml.Name `xml:"Part"`
		PartNumber int      `xml:"PartNumber"`
		ETag       struct {
			Value string `xml:",innerxml"`
		} `xml:"ETag"`
	}
}

func cos(file *os.File, totalSize int, ret *cosUploadSegments, internal bool, ChunkSize int, thread int, UUID string) (*UploadRes, error) {
	uploadUrl := ret.Url
	if internal {
		uploadUrl = strings.Replace(uploadUrl, "cos.accelerate", "cos-internal.ap-shanghai", 1)
	}
	fmt.Println(ret.FetchHeaders)
	client := &http.Client{}
	client.Timeout = 10 * time.Second
	req, _ := http.NewRequest("POST", uploadUrl+"?uploads&output=json", nil)
	req.Header = DefaultHeader.Clone()
	req.Header.Set("Authorization", ret.PostAuth)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	resxml := cosPreUploadXmlRes{}
	err = xml.Unmarshal(body, &resxml)
	if err != nil {
		fmt.Println("marshal Cos Videos XMl error", string(body))
		//return nil, err
	}
	if resxml.UploadID == "" {
		return nil, errors.New("get cos UploadId failed")
	}
	PostHeader := DefaultHeader.Clone()
	PostHeader.Set("Authorization", ret.PutAuth)
	uploader := &chunkUploader{
		uploadId:     resxml.UploadID,
		chunks:       int(math.Ceil(float64(totalSize) / float64(ChunkSize))),
		chunkSize:    ChunkSize,
		totalSize:    totalSize,
		threads:      thread,
		url:          uploadUrl,
		chunkInfo:    make(chan chunkInfo, int(math.Ceil(float64(totalSize)/float64(ChunkSize)))+10),
		uploadMethod: "cos",
		Header:       PostHeader,
		file:         file,
		MaxThread:    make(chan struct{}, thread),
	}
	err = uploader.upload(UUID)
	if err != nil {
		return nil, err
	}
	parts := partsXml{}
	for i := 0; i < uploader.chunks; i++ {
		c := <-uploader.chunkInfo
		index := c.Order
		etag := c.Etag
		parts.Part = append(parts.Part, struct {
			XMLName    xml.Name `xml:"Part"`
			PartNumber int      `xml:"PartNumber"`
			ETag       struct {
				Value string `xml:",innerxml"`
			} `xml:"ETag"`
		}{PartNumber: index, ETag: struct {
			Value string `xml:",innerxml"`
		}{Value: etag}})
	}

	sort.Slice(parts.Part, func(i, j int) bool {
		return parts.Part[i].PartNumber < parts.Part[j].PartNumber
	})
	xmlParts, _ := xml.Marshal(parts)
	for i := 0; ; i++ {
		client := &http.Client{}
		client.Timeout = time.Second * 20
		req, _ := http.NewRequest("POST", uploadUrl+"?uploadId="+uploader.uploadId,
			bytes.NewBuffer(xmlParts))
		req.Header.Set("Authorization", ret.PostAuth)
		req.Header.Set("Content-Type", "application/xml")
		res, err := client.Do(req)
		resBody, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != 200 {
			common.LogPrintf(UUID, common.BUlStr, err, file.Name(), "第", i, "次请求合并失败，正在重试")
			fmt.Println(resBody)
			res.Body.Close()
			if i == 10 {
				common.LogPrintf(UUID, common.BUlStr, err, file.Name(), "第10次请求合并失败")
				return nil, errors.New(fmt.Sprintln(file.Name(), "第10次请求合并失败", err))
			}
			time.Sleep(time.Second * 5)
			continue
		} else {
			break
		}
	}
	for i := 0; ; i++ {
		client := &http.Client{}
		client.Timeout = time.Second * 20
		req, _ := http.NewRequest("POST", "https:"+ret.FetchUrl, nil)
		req.Header = DefaultHeader.Clone()
		req.Header.Set("X-Upos-Fetch-Source", ret.FetchHeaders.XUposFetchSource)
		req.Header.Set("X-Upos-Auth", ret.FetchHeaders.XUposAuth)
		req.Header.Set("Fetch-Header-Authorization", ret.FetchHeaders.FetchHeaderAuthorization)
		res, err := client.Do(req)
		if err != nil {
			common.LogPrintf(UUID, common.BUlStr, err, file.Name(), "第", i, "次请求B站失败，正在重试")
			if i == 100 {
				common.LogPrintf(UUID, common.BUlStr, err, file.Name(), "第100次请求B站失败")
				return nil, errors.New(fmt.Sprintln(file.Name(), "第100次请求B站失败", err))
			}
			time.Sleep(time.Second * 5)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		t := struct {
			Ok int `json:"ok"`
		}{}
		_ = json.Unmarshal(body, &t)
		res.Body.Close()
		if t.Ok == 1 {
			upRes := &UploadRes{
				Title:    strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name())),
				Filename: ret.BiliFilename,
				Desc:     "",
				Info: cosFetchInfo{
					Url:           ret.FetchHeaders.XUposFetchSource,
					Authorization: ret.FetchHeaders.FetchHeaderAuthorization,
				},
			}
			return upRes, nil
		} else {
			common.LogPrintf("", common.BUlStr, "%s第 %d次请求，B站返回错误信息，正在重试", file.Name(), i)
			if i == 100 {
				common.LogPrintf(UUID, common.BUlStr, file.Name(), "第100次请求B站失败")
				return nil, errors.New("分片上传失败")
			}
			time.Sleep(time.Second * 5)
		}
	}
}
