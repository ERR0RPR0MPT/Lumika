package utils

import (
	"fmt"
	"strings"
)

func concatToString(args ...interface{}) string {
	strSlice := make([]string, len(args))
	for i, arg := range args {
		strSlice[i] = fmt.Sprint(arg)
	}
	return strings.Join(strSlice, " ")
}

func LogPrint(UUID string, a ...any) {
	result := concatToString(a...)
	// 为全局 Web API Log 输出
	LogVariable += result + "\n"
	if UUID != "" {
		for kp, kq := range AddTaskList {
			if kq.UUID == UUID {
				AddTaskList[kp].LogCat += result + "\n"
				break
			}
		}
	}
	fmt.Println(result)
}
