package main

import (
	"errors"
	"fmt"
	"os"
	"osslist/pkg/logging"
	"osslist/pkg/ossutils"
	"osslist/pkg/scacher"
	"osslist/pkg/util"

	"github.com/jessevdk/go-flags"
)

const (
	AppName      = "OSSListBuckets"
	AppVersion   = "0.0.2"
	BuildDate    = "2025-01-20"
	AppShortDesc = "list oss files"
	AppLongDesc  = "list oss files"
)

// Options 定义命令行选项
type Options struct {
	AccessKeyID     string `short:"i" long:"accessKeyId" description:"aliyun access key id" required:"true"`
	AccessKeySecret string `short:"k" long:"accessKeySecret" description:"aliyun access key secret" required:"true"`
	BucketName      string `short:"b" long:"bucket" description:"bucket name, default fetch all bucket"`
	Endpoint        string `short:"e" long:"endpoint" description:"aliyun oss endpoint" default:"https://oss.aliyuncs.com"`

	Prefix string `short:"p" long:"prefix" description:"only list files with prefix"`

	ExcludeExts        []string `long:"et" description:"exclude file extensions, e.g. mp4,jpg"`
	DefaultExcludeExts bool     `long:"ET" description:"use default exclude list (mp3, woff, woff2, css, mp4, jpg, png, avi, mov)"`
	ExcludeKeys        []string `long:"ek" description:"exclude directory keys, e.g. temp,cache"`
	DefaultExcludeKeys bool     `long:"EK" description:"use default exclude keys list (chunks, temp, cache)"`

	Output  string `short:"o" long:"output" description:"output result to file" default:"osslist.txt"`
	Workers int    `short:"w" long:"workers" description:"number of concurrent workers for listing files per bucket" default:"10"`

	// 日志配置
	LogFile       string `long:"lf" description:"日志文件路径(默认仅输出到控制台)" default:""`
	LogLevel      string `long:"ll" description:"日志级别: debug/info/warn/error" default:"info" choice:"debug" choice:"info" choice:"warn" choice:"error"`
	ConsoleFormat string `long:"lc" description:"控制台日志格式, 支持T(time),L(level),C(caller),F(func),M(msg), <empty> or <off> not print" default:"LCM"`
	Version       bool   `short:"v" long:"version" description:"显示程序版本并退出"`
}

// InitOptionsArgs 常用的工具函数，解析parser和logging配置
func InitOptionsArgs(minimumParams int) (*Options, *flags.Parser) {
	opts := &Options{}
	parser := flags.NewParser(opts, flags.Default)
	parser.Name = AppName
	parser.Usage = "[OPTIONS]"
	parser.ShortDescription = AppShortDesc
	parser.LongDescription = AppLongDesc

	// 命令行参数数量检查 指不包含程序名本身的参数数量
	if minimumParams > 0 && len(os.Args)-1 < minimumParams {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// 命令行参数解析检查
	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			os.Exit(0)
		}
		fmt.Printf("Error:%v\n", err)
		os.Exit(1)
	}

	// 版本号输出
	if opts.Version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// 初始化日志器
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := logging.InitLogger(logCfg); err != nil {
		fmt.Printf("failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	// 如果启用了默认排除列表，则添加默认排除项
	opts.ExcludeExts = util.ParseCommaStringsToList(opts.ExcludeExts)
	if opts.DefaultExcludeExts {
		defaultExcludes := []string{"mp3", "woff", "woff2", "css", "mp4", "jpg", "jpeg", "png", "avi", "mov", ossutils.NONE}
		opts.ExcludeExts = append(opts.ExcludeExts, defaultExcludes...)
	}

	// 如果启用了默认目录排除关键字列表，则添加默认关键字
	opts.ExcludeKeys = util.ParseCommaStringsToList(opts.ExcludeKeys)
	if opts.DefaultExcludeKeys {
		defaultExcludeKeys := []string{"chunks", "temp", "cache"}
		opts.ExcludeKeys = append(opts.ExcludeKeys, defaultExcludeKeys...)
	}

	return opts, parser
}

func main() {
	opts, _ := InitOptionsArgs(1)
	OSSListBuckets(opts.Output, opts.BucketName, opts.Endpoint, opts.AccessKeyID, opts.AccessKeySecret, opts.Prefix, opts.ExcludeExts, opts.ExcludeKeys, opts.Workers)
}

func OSSListBuckets(output, bucketNameDF, endpointDF, accessKeyID, accessKeySecret, filterPrefix string, filterSuffixes []string, excludeKeys []string, workers int) {
	// 初始化缓存
	akHash := util.GetStringHash(accessKeyID+accessKeySecret, 8)
	cacher := scacher.NewCacher(fmt.Sprintf("%s.%s.cache", AppName, akHash))
	defer cacher.Sync()

	// 初始化 Writer
	fileReceiver, err := util.NewFileReceiver(output)
	if err != nil {
		logging.Fatalf("Init Writer failed: %v", err)
	}

	// 处理buckets
	logging.Info("getting the Bucket list...")
	buckets, err := ossutils.GetBuckets(accessKeyID, accessKeySecret, cacher)
	logging.Infof("found %d items bucket: %s", len(buckets), util.ToJson(buckets))
	if err != nil {
		logging.Errorf("failed to get the Bucket list: %v", err)
	}

	buckets = ossutils.FilterOrFallbackBuckets(buckets, bucketNameDF)
	logging.Infof("specify %d items bucket: %s", len(buckets), util.ToJson(buckets))
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

		endpoint := ossutils.BuildEndpoint(bucket, endpointDF)
		if err := ossutils.WalkerBucket(bucket.Name, endpoint, accessKeyID, accessKeySecret, filterPrefix, filterSuffixes, excludeKeys, workers, cacher, outChan); err != nil {
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
