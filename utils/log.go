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
		for kp, kq := range DlTaskList {
			if kq.UUID == UUID {
				DlTaskList[kp].LogCat += result + "\n"
				break
			}
		}
		for kp, kq := range BDlTaskList {
			if kq.UUID == UUID {
				BDlTaskList[kp].LogCat += result + "\n"
				break
			}
		}
		for kp, kq := range AddTaskList {
			if kq.UUID == UUID {
				AddTaskList[kp].LogCat += result + "\n"
				break
			}
		}
		for kp, kq := range GetTaskList {
			if kq.UUID == UUID {
				GetTaskList[kp].LogCat += result + "\n"
				break
			}
		}
	}
	fmt.Println("", result)
}

func LogPrintf(UUID string, format string, a ...any) {
	result := formatString(format, a)
	// 为全局 Web API Log 输出
	LogVariable.WriteString(result + "\n")
	if UUID != "" {
		for kp, kq := range DlTaskList {
			if kq.UUID == UUID {
				DlTaskList[kp].LogCat += result + "\n"
				break
			}
		}
		for kp, kq := range BDlTaskList {
			if kq.UUID == UUID {
				BDlTaskList[kp].LogCat += result + "\n"
				break
			}
		}
		for kp, kq := range AddTaskList {
			if kq.UUID == UUID {
				AddTaskList[kp].LogCat += result + "\n"
				break
			}
		}
		for kp, kq := range GetTaskList {
			if kq.UUID == UUID {
				GetTaskList[kp].LogCat += result + "\n"
				break
			}
		}
	}
	fmt.Printf(format, a...)
}
