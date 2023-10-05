package biliup

import (
	"errors"
	"github.com/google/go-querystring/query"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"strconv"
)

type chunkInfo struct {
	Order int
	Etag  string
}
type chunkUploader struct {
	uploadId     string
	chunks       int
	chunkSize    int
	totalSize    int
	threads      int
	url          string
	chunkInfo    chan chunkInfo
	uploadMethod string
	Header       http.Header
	file         *os.File
	MaxThread    chan struct{}
}
type chunkParams struct {
	UploadId   string `url:"uploadId"`
	Chunks     int    `url:"chunks"`
	Total      int    `url:"total"`
	Chunk      int    `url:"chunk"`
	Size       int    `url:"size"`
	PartNumber int    `url:"partNumber"`
	Start      int    `url:"start"`
	End        int    `url:"end"`
}

func (u *chunkUploader) upload() error {
	group := new(errgroup.Group)
	for i := 0; i < u.chunks; i++ {
		u.MaxThread <- struct{}{}
		buf := make([]byte, u.chunkSize)
		bufSize, _ := u.file.Read(buf)
		chunk := chunkParams{
			UploadId:   u.uploadId,
			Chunks:     u.chunks,
			Chunk:      i,
			Total:      u.totalSize,
			Size:       bufSize,
			PartNumber: i + 1,
			Start:      i * u.chunkSize,
			End:        i*u.chunkSize + bufSize,
		}
		group.Go(func() error {
			return u.uploadChunk(buf, chunk)
		})
	}
	if err := group.Wait(); err != nil {
		close(u.chunkInfo)
		return err
	}
	close(u.chunkInfo)
	return nil
}
func (u *chunkUploader) uploadChunk(data []byte, params chunkParams) error {
	defer func() {
		<-u.MaxThread
	}()
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	req.Header.SetMethod("PUT")
	for k, v := range u.Header {
		req.Header.Set(k, v[0])
	}
	req.SetBodyRaw(data)
	vals, _ := query.Values(params)
	req.SetRequestURI(u.url + "?" + vals.Encode())
	for i := 0; i <= ChunkUploadMaxRetryTimes; i++ {
		err := fasthttp.Do(req, resp)
		if err != nil || resp.StatusCode() != 200 {
			log.Println("上传分块出现问题，尝试重连")
			log.Println(err)
		} else {
			c := chunkInfo{
				Order: params.PartNumber,
				Etag:  "",
			}
			if u.uploadMethod == "cos" {
				c.Etag = string(resp.Header.Peek("ETag"))
				//Upos不需要ETAG
			}
			u.chunkInfo <- c
			//fasthttp.ReleaseResponse(resp)
			break
		}
		//fasthttp.ReleaseResponse(resp)
		if i == ChunkUploadMaxRetryTimes {
			log.Println("上传分块出现问题，重试次数超过限制")
			return errors.New(strconv.Itoa(u.chunks) + "分块上传失败")
		}
	}

	return nil
}
