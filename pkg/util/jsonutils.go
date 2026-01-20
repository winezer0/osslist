package util

import (
	"encoding/json"
	"fmt"
)

// ToJson 将任意 map 转换为格式化的 JSON 字符串（用于输出）
func ToJson(v interface{}) string {
	return string(ToJsonBytes(v))
}

// ToJsonBytes  将任意 map 转换为格式化的 JSON 字符串（用于输出）
func ToJsonBytes(v interface{}) []byte {
	data, _ := json.MarshalIndent(v, "", "  ")
	return data
}

// LoadJSONString 从 JSON 字符串内容反序列化数据
func LoadJSONString(jsonString string, v interface{}) error {
	// 检查输入字符串是否为空
	if jsonString == "" {
		return fmt.Errorf("json string is empty")
	}
	// 将字符串转换为字节切片，以便 json.Unmarshal 处理
	data := []byte(jsonString)
	// 使用 json.Unmarshal 将字节数据解析到 v 指向的结构中
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON string: %v", err)
	}
	return nil
}
