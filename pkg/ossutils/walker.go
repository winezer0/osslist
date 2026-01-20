package ossutils

import (
	"path"
	"strings"
	"sync"

	"osslist/pkg/logging"
	"osslist/pkg/scacher"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSWalker 处理 OSS 遍历逻辑
type OSSWalker struct {
	Concurrency   int
	Cacher        *scacher.Cacher
	excludeExtMap map[string]struct{}
	excludeKeyMap map[string]struct{}
}

// NewOSSWalker 创建一个新的 OSSWalker 实例
func NewOSSWalker(exclude []string, excludeKeys []string, concurrency int, cacher *scacher.Cacher) *OSSWalker {
	if concurrency <= 0 {
		concurrency = 1
	}

	// 初始化排除映射表 (文件扩展名)
	excludeExtMap := make(map[string]struct{})
	for _, ext := range exclude {
		key := strings.ToLower(strings.TrimSpace(ext))
		if key != "" {
			excludeExtMap[key] = struct{}{}
		}
	}

	// 初始化排除映射表 (目录关键字)
	excludeKeyMap := make(map[string]struct{})
	for _, key := range excludeKeys {
		k := strings.ToLower(strings.TrimSpace(key))
		if k != "" {
			excludeKeyMap[k] = struct{}{}
		}
	}

	return &OSSWalker{
		Concurrency:   concurrency,
		Cacher:        cacher,
		excludeExtMap: excludeExtMap,
		excludeKeyMap: excludeKeyMap,
	}
}

// Walk 遍历 OSS 对象并将文件路径发送到 channel
func (w *OSSWalker) Walk(client *oss.Client, bucketName, prefix string, outChan chan<- string) {
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		logging.Errorf("failed to obtain bucket %s error: %v", bucketName, err)
		return
	}

	var wg sync.WaitGroup
	// 使用带缓冲的 channel 作为信号量控制并发数
	sem := make(chan struct{}, w.Concurrency)

	wg.Add(1)
	go w.walkRecursive(bucket, prefix, outChan, &wg, sem)

	// 等待所有 goroutine 完成
	wg.Wait()
}

// walkRecursive 递归遍历OSS对象
func (w *OSSWalker) walkRecursive(bucket *oss.Bucket, prefix string, outChan chan<- string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()

	// 检查缓存
	cacheKey := bucket.BucketName + ":" + prefix
	skipFiles := false
	if w.Cacher != nil {
		if _, ok := w.Cacher.Get(cacheKey); ok {
			logging.Debugf("cache hit for %s, skipping files", prefix)
			// 缓存命中，标记跳过文件处理，但仍需继续遍历以发现子目录
			skipFiles = true
		}
	}

	// 获取信号量
	sem <- struct{}{}
	defer func() { <-sem }()

	marker := ""

	for {
		options := []oss.Option{oss.Marker(marker), oss.Delimiter("/")}
		if prefix != "" {
			options = append(options, oss.Prefix(prefix))
		}

		res, err := bucket.ListObjects(options...)
		if err != nil {
			logging.Errorf("bucket %s get objects list error: %v", bucket.BucketName, err)
			return
		}

		// 处理目录（CommonPrefixes）
		for _, commonPrefix := range res.CommonPrefixes {
			// 检查是否包含被排除的关键字
			if w.shouldExcludeDir(commonPrefix) {
				logging.Infof("skipping excluded directory: %s", commonPrefix)
				continue
			}

			// 并发递归处理子目录
			wg.Add(1)
			go w.walkRecursive(bucket, commonPrefix, outChan, wg, sem)
		}

		// 如果缓存命中，说明该目录已扫描过，跳过文件输出
		if !skipFiles {
			// 处理文件
			for _, obj := range res.Objects {
				if obj.Key == prefix { // 跳过目录自身（如果存在）
					continue
				}
				// 检查是否应该排除该文件
				// 提取文件名部分用于检查扩展名
				nameCheck := path.Base(obj.Key)
				if !w.shouldExcludeFile(nameCheck) {
					outChan <- obj.Key
					logging.Infof("discover the file: %s", obj.Key)
				}
			}
		}

		if res.IsTruncated {
			marker = res.NextMarker
		} else {
			break
		}
	}

	// 遍历完成，写入缓存
	// 仅记录 "1" 表示该目录已完成扫描
	if w.Cacher != nil {
		w.Cacher.Set(cacheKey, "1")
	}
}

const NONE = "none"

// shouldExcludeFile 检查文件是否应该被排除
func (w *OSSWalker) shouldExcludeFile(filename string) bool {

	// 如果没有任何排除规则，直接返回false
	if len(w.excludeExtMap) == 0 {
		return false
	}

	// 获取文件扩展名
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(filename), "."))

	// 处理无后缀文件的情况，当扩展名为空时，视为 "NONE"
	checkExt := ext
	if checkExt == "" {
		checkExt = NONE
	}

	// 检查扩展名是否在排除映射表中
	if _, ok := w.excludeExtMap[checkExt]; ok {
		//logging.Debugf("exclude file %s (extension: %s)", filename, ext)
		return true
	}

	return false
}

// shouldExcludeDir 检查目录是否应该被排除
func (w *OSSWalker) shouldExcludeDir(dirPath string) bool {
	if len(w.excludeKeyMap) == 0 {
		return false
	}

	// dirPath 通常是 "folder/subfolder/" 的形式
	// 我们需要拆分路径，检查每一级目录是否在排除列表中
	// 或者，根据需求，只要路径中包含关键字就排除？
	// 用户需求：当路径中的目录等于指定关键字列表时，不进行遍历
	// 这意味着我们需要检查路径中的每一个 path segment

	// 去除末尾的 "/"
	trimmedPath := strings.TrimSuffix(dirPath, "/")
	// 按 "/" 分割
	parts := strings.Split(trimmedPath, "/")

	for _, part := range parts {
		if part == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(part))
		if _, ok := w.excludeKeyMap[key]; ok {
			return true
		}
	}

	return false
}
