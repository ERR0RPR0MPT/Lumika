package utils

import (
	"fmt"
	bg "github.com/iyear/biligo"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func BDl(AVOrBVStr string) {
	var aid int64
	if !strings.Contains(AVOrBVStr[:2], "av") {
		aid = bg.BV2AV(AVOrBVStr)
	} else {
		anum, err := strconv.ParseInt(AVOrBVStr[2:], 10, 64)
		if err != nil {
			fmt.Println(BDlStr, "转换失败:", err)
			return
		}
		aid = anum
	}
	b := bg.NewCommClient(&bg.CommSetting{})
	info, err := b.VideoGetInfo(aid)
	if err != nil {
		fmt.Println(BDlStr, "VideoGetInfo 失败:", err)
		return
	}
	fmt.Println(BDlStr, "视频标题:", info.Title)
	fmt.Println(BDlStr, "视频 aid:", info.AID)
	fmt.Println(BDlStr, "视频 BVid:", info.BVID)
	fmt.Println(BDlStr, "视频 cid:", info.CID)
	fmt.Println(BDlStr, "视频简介", info.Desc)
	fmt.Println(BDlStr, "总时长:", info.Duration)
	fmt.Println(BDlStr, "视频分 P 数量:", len(info.Pages))

	fmt.Println(BDlStr, "创建下载目录...")
	SuitableFileName := ReplaceInvalidCharacters(info.Title, '-')
	// 检查是否已经存在下载目录
	if _, err := os.Stat(filepath.Join("lumika_data", SuitableFileName)); err == nil {
		fmt.Println(BDlStr, "下载目录已存在，跳过创建下载目录")
	} else if os.IsNotExist(err) {
		fmt.Println(BDlStr, "下载目录不存在，创建下载目录")
		// 创建目录
		err = os.Mkdir(filepath.Join("lumika_data", SuitableFileName), os.ModePerm)
		if err != nil {
			fmt.Println(BDlStr, "创建下载目录失败:", err)
			return
		}
	} else {
		fmt.Println(BDlStr, "检查下载目录失败:", err)
		return
	}

	fmt.Println(BDlStr, "遍历所有分 P ...")
	// 启动多个goroutine
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, DefaultBiliDownloadsMaxQueueNum)
	allStartTime := time.Now()
	for pi := range info.Pages {
		fmt.Println(BDlStr, "尝试获取 "+strconv.Itoa(pi+1)+"P 的视频地址...")
		wg.Add(1)               // 增加计数器
		semaphore <- struct{}{} // 协程获取信号量，若已满则阻塞
		go func(pi int) {
			defer func() {
				<-semaphore // 协程释放信号量
				wg.Done()
			}()
			videoPlayURLResult, err := b.VideoGetPlayURL(aid, info.Pages[pi].CID, 16, 1)
			if err != nil {
				fmt.Println(BDlStr, "获取视频地址失败，跳过本分P视频")
				return
			}
			durl := videoPlayURLResult.DURL[0].URL
			videoName := strconv.Itoa(pi+1) + "-" + SuitableFileName + ".mp4"
			filePath := filepath.Join("lumika_data", SuitableFileName, videoName)
			fmt.Println(BDlStr, "视频地址:", durl)
			fmt.Println(BDlStr, "尝试下载视频...")
			err = Dl(durl, filePath, DefaultBiliDownloadReferer, DefaultUserAgent, DefaultBiliDownloadGoRoutines)
			if err != nil {
				fmt.Println(BDlStr, "下载视频("+videoName+")失败，跳过本分P视频:", err)
				return
			}
		}(pi)
	}
	wg.Wait()
	allEndTime := time.Now()
	allDuration := allEndTime.Sub(allStartTime)
	fmt.Printf(BDlStr+" 总共耗时%f秒\n", allDuration.Seconds())
	fmt.Println(BDlStr, "视频全部下载完成")
}
