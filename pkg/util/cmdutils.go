package util

import (
	"strings"
)

// ParseCommaStringsToList 通用函数：将字符串或字符串切片转换为目标类型的切片
func ParseCommaStringsToList(input []string) []string {
	var strList []string
	// 将单个字符串按逗号分割
	if len(input) == 1 && input[0] != "" {
		strList = parseCommaStringToList(input[0])
	} else {
		strList = input
	}
	//清理其中的空白字符数据
	strList = normalizeStrList(strList)
	return strList
}

// parseCommaStringToList 解析逗号分隔字符串为列表
func parseCommaStringToList(CommaStr string) []string {
	var result []string
	if CommaStr != "" {
		strList := strings.Split(CommaStr, ",")
		for _, str := range strList {
			str = strings.TrimSpace(str)
			if str == "" {
				continue
			}
			result = append(result, str)
		}
	}
	return result
}

// normalizeStrList 规范化字符串切片：去除空格、过滤空字符串
func normalizeStrList(input []string) []string {
	result := make([]string, 0, len(input))
	for _, s := range input {
		if s == "" {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		result = append(result, s)
	}
	return result
}
