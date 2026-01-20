package util

import (
	"crypto/md5"
	"encoding/hex"
)

// GetStringHash 获取路径的哈希值
func GetStringHash(str string, length int) string {
	if length <= 0 || length > 32 {
		length = 32
	}

	hash := md5.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))[:length]
}
