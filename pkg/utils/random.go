package utils

import "math/rand/v2"

// Random 从数组中随机选择一个元素
func Random(arr []string) string {
	if len(arr) == 0 {
		return ""
	}
	return arr[RandomInt(len(arr))]
}

// RandomInt 生成一个随机数
func RandomInt(max int) int {
	return rand.IntN(max)
}
