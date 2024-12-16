package utils

import "strings"

// GetLastElement 获取目录中最后一个元素
func GetLastElement(path string) string {
	if path == "" {
		return ""
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	elements := strings.Split(path, "/")
	return LowerString(elements[len(elements)-1])
}
