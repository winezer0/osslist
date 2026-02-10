package ossutils

import (
	"fmt"
	"github.com/winezer0/xutils/cacher"
	"github.com/winezer0/xutils/hashutils"
	"github.com/winezer0/xutils/logging"
	"github.com/winezer0/xutils/utils"
)

func OSSListBuckets(outputFile, bucketNameDF, endpointDF, accessKeyID, accessKeySecret, filterPrefix string, filterSuffixes []string, excludeKeys []string, workers int) {
	// 初始化缓存
	akHash := hashutils.GetStrHashShort(accessKeyID + accessKeySecret)
	cacheFile := fmt.Sprintf("osslist.%s.cache", akHash)
	cacheManager := cacher.NewCacheManager(cacheFile)
	defer cacheManager.Close()

	// 初始化 Writer
	fileReceiver, err := utils.NewFileReceiver(outputFile)
	if err != nil {
		logging.Fatalf("Init Writer failed: %v", err)
	}

	// 处理buckets
	logging.Info("getting the Bucket list...")
	buckets, err := GetBuckets(accessKeyID, accessKeySecret, cacheManager)
	logging.Infof("found %d items bucket: %s", len(buckets), utils.ToJSON(buckets))
	if err != nil {
		logging.Errorf("failed to get the Bucket list: %v", err)
	}

	buckets = FilterOrFallbackBuckets(buckets, bucketNameDF)
	logging.Infof("specify %d items bucket: %s", len(buckets), utils.ToJSON(buckets))
	if len(buckets) == 0 {
		logging.Fatalf("please manual specify bucket name or the account not has any bucket")
	}

	// 创建 channel 用于接收文件路径
	outChan := make(chan string, 1000)         // 缓冲区大小可根据需要调整
	writerDone := make(chan struct{})          // 用于等待 writer 结束
	go fileReceiver.Start(outChan, writerDone) // 启动 Writer Goroutine

	// 串行遍历所有 Bucket，避免并发爆炸（Bucket并发 * 内部并发）
	// 由 opts.Workers 控制全局最大并发数
	for index, bucket := range buckets {
		logging.Infof("start processing bucket: %s (%d/%d)", bucket.Name, index+1, len(buckets))

		endpoint := BuildEndpoint(bucket, endpointDF)
		if err := WalkerBucket(bucket.Name, endpoint, accessKeyID, accessKeySecret, filterPrefix, filterSuffixes, excludeKeys, workers, cacheManager, outChan); err != nil {
			logging.Error(err)
			continue
		}

		logging.Infof("bucket: %s tasks have been completed.", bucket.Name)
	}

	// 所有生产者完成后，关闭 channel
	close(outChan)
	// 等待 Writer 完成写入
	<-writerDone
	logging.Info("all tasks have been completed.")
}
