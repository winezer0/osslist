package ossutils

import (
	"fmt"
	"github.com/winezer0/xutils/cacher"
	"github.com/winezer0/xutils/hashutils"
	"github.com/winezer0/xutils/logging"
	"github.com/winezer0/xutils/utils"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// GetBuckets 获取当前账号下的所有 Bucket 列表
func GetBuckets(accessKeyID, accessKeySecret string, cacheManager *cacher.CacheManager) ([]oss.BucketProperties, error) {
	// 从缓存文件中获取
	cacheKey := hashutils.GetStrHashShort(accessKeyID + accessKeySecret)
	var bucketProperties []oss.BucketProperties
	if exist := cacheManager.GetAs(cacheKey, &bucketProperties); exist {
		return bucketProperties, nil
	}

	// 创建用于 GetBuckets 的 client（必须用全局 endpoint）
	// 注意：GetBuckets 可以在任何 Endpoint 调用，但通常建议用 oss.aliyuncs.com 或者具体的 endpoint

	client, err := oss.New("https://oss.aliyuncs.com", accessKeyID, accessKeySecret)
	if err != nil {
		logging.Errorf("创建全局 OSS 客户端失败: %v", err)
		return nil, err
	}

	bucketInfo, err := client.ListBuckets()
	if err != nil {
		return nil, err
	}

	if len(bucketInfo.Buckets) > 0 {
		cacheManager.Set(cacheKey, bucketInfo.Buckets)
	}

	return bucketInfo.Buckets, nil
}

// FilterOrFallbackBuckets 根据用户指定的 bucketName 过滤 buckets 列表。
// 如果 buckets 为空（可能因权限问题），则 fallback 创建一个 BucketProperties 占位。
func FilterOrFallbackBuckets(buckets []oss.BucketProperties, bucketName string) []oss.BucketProperties {
	// 如果用户未指定 bucketName，返回原列表
	if bucketName == "" {
		return buckets
	}

	// 情况1: buckets 非空 → 查找是否存在
	if len(buckets) > 0 {
		for _, b := range buckets {
			if b.Name == bucketName {
				return []oss.BucketProperties{b} // 找到，只返回这个
			}
		}
		logging.Errorf("no bucket %s found in %s", bucketName, utils.ToJSON(buckets))
	}

	// 情况2: buckets 为空 或 未找到指定 bucket
	// 可能原因：权限不足（ListBuckets 返回空）、bucket 不存在、跨账号等
	// 此时我们 fallback：创建一个 Location 为空的 BucketProperties
	logging.Warnf("Bucket '%s' not found in ListBuckets result. Fallback to manual entry (Location will be empty).", bucketName)
	return []oss.BucketProperties{
		NewBucketProperties(bucketName, ""), // Location 留空，后续由 buildEndpoint 处理
	}
}

// NewBucketProperties 创建一个新的 BucketProperties 实例
// 仅设置 Name 和 Location，其他字段留空（CreationDate 为零值，StorageClass 为空字符串）
func NewBucketProperties(name, location string) oss.BucketProperties {
	return oss.BucketProperties{
		Name:     name,
		Location: location,
		// CreationDate: time.Time{}  // 零值，默认就是
		// StorageClass: "",          // 默认空
	}
}

// BuildEndpoint 根据 Bucket 的 Location 构造正确的 OSS Endpoint。
// 如果 Location 为空，则使用默认 Endpoint（建议传入 "https://oss.aliyuncs.com"）。
func BuildEndpoint(bucket oss.BucketProperties, defaultEndpoint string) string {
	location := strings.TrimSpace(bucket.Location)

	// 情况1: Location 有效 → 构造标准 endpoint
	if location != "" {
		// 阿里云 ListBuckets 返回的 Location 格式为 "cn-hangzhou", "us-west-1" 等
		// 标准 endpoint 格式: https://oss-<location>.aliyuncs.com
		if strings.HasPrefix(location, "oss-") {
			// 兼容性处理：如果 Location 已包含 "oss-"（理论上不会），直接用
			return fmt.Sprintf("https://%s.aliyuncs.com", location)
		}
		return fmt.Sprintf("https://oss-%s.aliyuncs.com", location)
	}

	// 情况2: Location 为空 → 使用默认 endpoint
	if defaultEndpoint == "" {
		defaultEndpoint = "https://oss.aliyuncs.com" // 阿里云全局 endpoint
	}
	logging.Warnf("bucket %s has empty location, using fallback endpoint: %s", bucket.Name, defaultEndpoint)
	return defaultEndpoint
}

// WalkerBucket 封装单个 Bucket 的处理逻辑
func WalkerBucket(bucketName, endpoint, keyId, keySecret, prefix string, exSuffixes []string, excludeKeys []string, concurrency int, cacher *cacher.CacheManager, outChan chan<- string) error {
	client, err := oss.New(endpoint, keyId, keySecret)
	if err != nil {
		return fmt.Errorf("failed to create client for bucket %s (endpoint: %s): %v", bucketName, endpoint, err)
	}
	w := NewOSSWalker(exSuffixes, excludeKeys, concurrency, cacher)
	w.Walk(client, bucketName, prefix, outChan)
	return nil
}
