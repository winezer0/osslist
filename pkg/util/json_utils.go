package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

// ToJsonWithErr 将任意 map 转换为格式化的 JSON 字符串（用于输出）
func ToJsonWithErr(v interface{}) (string, error) {
	bytes, err := ToJsonBytesWithErr(v)
	return string(bytes), err
}

// ToJsonBytesWithErr  将任意 map 转换为格式化的 JSON 字符串（用于输出）
func ToJsonBytesWithErr(v interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	return data, err
}

func WriteJson(filePath string, v interface{}) error {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}

	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("文件写入失败: %w", err)
	}

	return nil
}

// ReadJson 从指定JSON文件读取数据并反序列化到目标对象
func ReadJson(filePath string, target interface{}) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return err // 返回文件打开错误（如文件不存在）
	}
	defer file.Close() // 确保文件会被关闭

	// 创建JSON解码器并解析到目标对象
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return err // 返回JSON解析错误（如格式不正确）
	}

	return nil
}

// ParseJSON 尝试将任意字符串解析为 jq 可用的标准 interface{}
// 支持：
//   - 单个 JSON 对象：{"a":1}
//   - JSON 数组：[{"a":1}, {"b":2}]
//   - JSONL（每行一个 JSON）：自动合并为数组
func ParseJSON(s string) (interface{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("输入为空")
	}

	// 尝试整体解析为单个 JSON 值（对象或数组）
	var single interface{}
	if err := json.Unmarshal([]byte(s), &single); err == nil {
		return single, nil
	}

	// 否则尝试按 JSONL（逐行）解析
	return parseJSONLines(s)
}

// parseJSONLines 将多行 JSON 字符串解析为 []interface{}
func parseJSONLines(s string) ([]interface{}, error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var results []interface{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var obj interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return nil, fmt.Errorf("无效 JSON 行: %s (%w)", line, err)
		}
		results = append(results, obj)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到有效 JSON 行")
	}

	return results, nil
}
