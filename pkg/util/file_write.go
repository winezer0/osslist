package util

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// WriteStringsToFile 将目标写入文件
func WriteStringsToFile(filePath string, strs []string) error {
	// 确保目录存在
	if err := EnsurePathDir(filePath, true); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 写入文件
	content := strings.Join(strs, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// WriteStringToFile 将目标写入文件
func WriteStringToFile(filePath string, content string) error {
	// 确保目录存在
	if err := EnsurePathDir(filePath, true); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}
	return nil
}

// DeleteFileIfExists 当文件存在时删除该文件
// 参数: filePath - 目标文件路径
// 返回: 操作结果或错误信息（文件不存在时返回 nil）
func DeleteFileIfExists(filePath string) error {
	// 检查文件路径是否为空
	if filePath == "" {
		return errors.New("文件路径不能为空")
	}

	// 检查文件是否存在
	_, err := os.Stat(filePath)
	if err != nil {
		// 文件不存在时直接返回（不视为错误）
		if os.IsNotExist(err) {
			return nil
		}
		// 其他错误（如权限问题、路径是目录等）返回错误信息
		return fmt.Errorf("检查文件状态失败: %w", err)
	}

	// 执行删除操作
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	return nil
}
