package utils

import (
	"fmt"
	"strings"
)

const (
	XorCode  int64 = 23442827791579
	MaskCode int64 = 2251799813685247
	MaxAid   int64 = 1 << 51
	ALPHABET       = "FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf"
)

var (
	EncodeMap = []int{8, 7, 0, 5, 1, 3, 2, 4, 6}
	DecodeMap = reverse(EncodeMap)
	BASE      = int64(len(ALPHABET))
	PREFIX    = "BV1"
	PrefixLen = len(PREFIX)
	CodeLen   = len(EncodeMap)
)

func reverse(arr []int) []int {
	result := make([]int, len(arr))
	for i, j := 0, len(arr)-1; i < len(arr); i, j = i+1, j-1 {
		result[i] = arr[j]
	}
	return result
}

func AV2BV(aid int64) string {
	bvid := make([]string, 9)
	tmp := (MaxAid | aid) ^ XorCode
	for i := 0; i < CodeLen; i++ {
		bvid[EncodeMap[i]] = string(ALPHABET[tmp%BASE])
		tmp /= BASE
	}
	return PREFIX + strings.Join(bvid, "")
}

func BV2AV(bvid string) (int64, error) {
	if bvid[:PrefixLen] != PREFIX {
		return 0, fmt.Errorf("invaild BV string")
	}

	bvid = bvid[PrefixLen:]
	var tmp int64
	for i := 0; i < CodeLen; i++ {
		idx := strings.IndexByte(ALPHABET, bvid[DecodeMap[i]])
		tmp = tmp*BASE + int64(idx)
	}
	return (tmp & MaskCode) ^ XorCode, nil
}
