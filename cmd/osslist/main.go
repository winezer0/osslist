package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/winezer0/xutils/cmdutils"
	"github.com/winezer0/xutils/logging"
	"os"
	"osslist/pkg/ossutils"
)

const (
	AppName      = "OSSListBuckets"
	AppVersion   = "0.0.4"
	BuildDate    = "2026-02-10"
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

	ExcludeExts    []string `long:"et" description:"exclude file extensions, e.g. mp4,jpg"`
	DefExcludeExts bool     `long:"ET" description:"use default exclude list (mp3, woff, woff2, css, mp4, jpg, png, avi, mov)"`
	ExcludeKeys    []string `long:"ek" description:"exclude directory keys, e.g. temp,cache"`
	DefExcludeKeys bool     `long:"EK" description:"use default exclude keys list (chunks, temp, cache)"`

	Output  string `short:"o" long:"output" description:"output result to file" default:"osslist.txt"`
	Workers int    `short:"w" long:"workers" description:"number of concurrent workers for listing files per bucket" default:"10"`

	// 日志配置
	LogFile    string `long:"lf" description:"log file path (default output only to console)" default:""`
	LogLevel   string `long:"ll" description:"log level: debug/info/warn/error" default:"info" choice:"debug" choice:"info" choice:"warn" choice:"error"`
	LogConsole string `long:"lc" description:"log format on console, Support T(time),L(level),C(caller),F(func),M(msg), <empty> or <off> not print" default:"LM"`
	Version    bool   `short:"v" long:"version" description:"display the version and exit"`
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
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.LogConsole)
	if err := logging.InitLogger(logCfg); err != nil {
		fmt.Printf("failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync()

	// 如果启用了默认排除列表，则添加默认排除项
	opts.ExcludeExts = cmdutils.FormatCmdsComma(opts.ExcludeExts)
	if opts.DefExcludeExts {
		defaultExcludes := []string{"mp3", "woff", "woff2", "css", "mp4", "jpg", "jpeg", "png", "avi", "mov", ossutils.NONE}
		opts.ExcludeExts = append(opts.ExcludeExts, defaultExcludes...)
	}

	// 如果启用了默认目录排除关键字列表，则添加默认关键字
	opts.ExcludeKeys = cmdutils.FormatCmdsComma(opts.ExcludeKeys)
	if opts.DefExcludeKeys {
		defaultExcludeKeys := []string{"chunks", "temp", "cache"}
		opts.ExcludeKeys = append(opts.ExcludeKeys, defaultExcludeKeys...)
	}

	return opts, parser
}

func main() {
	opts, _ := InitOptionsArgs(1)
	ossutils.OSSListBuckets(opts.Output, opts.BucketName, opts.Endpoint, opts.AccessKeyID, opts.AccessKeySecret, opts.Prefix, opts.ExcludeExts, opts.ExcludeKeys, opts.Workers)
}
