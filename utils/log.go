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

func formatString(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func LogPrintln(UUID string, a ...any) {
	result := concatToString(a...)
	// 为全局 Web API Log 输出
	LogVariable.WriteString(result + "\n")
	if UUID != "" {
		_, exist := DlTaskList[UUID]
		if exist {
			DlTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = BDlTaskList[UUID]
		if exist {
			BDlTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = AddTaskList[UUID]
		if exist {
			AddTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = GetTaskList[UUID]
		if exist {
			GetTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = BUlTaskList[UUID]
		if exist {
			BUlTaskList[UUID].LogCat += result + "\n"
		}
	}
	fmt.Println("", result)
}

func LogPrintf(UUID string, format string, a ...any) {
	result := formatString(format, a)
	// 为全局 Web API Log 输出
	LogVariable.WriteString(result + "\n")
	if UUID != "" {
		_, exist := DlTaskList[UUID]
		if exist {
			DlTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = BDlTaskList[UUID]
		if exist {
			BDlTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = AddTaskList[UUID]
		if exist {
			AddTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = GetTaskList[UUID]
		if exist {
			GetTaskList[UUID].LogCat += result + "\n"
		}
		_, exist = BUlTaskList[UUID]
		if exist {
			BUlTaskList[UUID].LogCat += result + "\n"
		}
	}
	fmt.Printf(format, a...)
}
