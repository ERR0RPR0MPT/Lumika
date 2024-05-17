package test

import (
	"fmt"
	"github.com/ERR0RPR0MPT/Lumika/utils"
	"testing"
)

func TestBV2AV(t *testing.T) {
	bvid := "BV1L9Uoa9EUx"
	aidDecoded, err := utils.BV2AV(bvid)
	if err != nil {
		panic(err)
	}
	fmt.Println(aidDecoded)
	bvidDecoded := utils.AV2BV(aidDecoded)
	fmt.Println(bvidDecoded)
}
