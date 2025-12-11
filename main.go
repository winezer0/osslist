package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	flags "github.com/jessevdk/go-flags"
	"osslist/pkg/logging"
)

// Options 定义命令行选项
type Options struct {
	AccessKeyID     string   `short:"i" long:"accessKeyId" description:"aliyun access key id" required:"true"`
	AccessKeySecret string   `short:"k" long:"accessKeySecret" description:"aliyun access key secret" required:"true"`
	BucketName      string   `short:"b" long:"bucket" description:"bucket name, default fetch all bucket"`
	Endpoint        string   `short:"e" long:"endpoint" description:"aliyun oss endpoint" default:"https://oss-cn-beijing.aliyuncs.com"`
	Prefix          string   `short:"p" long:"prefix" description:"only list files with prefix"`
	Output          string   `short:"o" long:"output" description:"output result to file" default:"osslist.txt"`
	Exclude         []string `short:"x" long:"exclude" description:"exclude file extensions, e.g. mp4,jpg"`
	DefaultExclude  bool     `short:"X" long:"default-exclude" description:"use default exclude list (mp3, woff, woff2, css, mp4, jpg, png, avi, mov)"`
	JSONOutput      string   `short:"j" long:"json" description:"output result to JSON file"`
}

// TreeNode 表示树形结构的节点
type TreeNode struct {
	Name     string     `json:"name"`
	IsDir    bool       `json:"is_dir"`
	FullPath string     `json:"full_path,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}

var opts Options

// 全局 writer，可以是 os.Stdout 或文件
var out io.Writer = os.Stdout

// 全局日志器
var logger *logging.Logger

func main() {
	// 初始化日志器
	logConfig := logging.NewLogConfig("info", "", "TLCM")
	var err error
	logger, err = logging.CreateLogger("osslist", logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	_, err = flags.Parse(&opts)
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(0)
		}
		logger.Errorf("参数解析错误: %v", err)
		os.Exit(1)
	}

	// 如果指定了 output 文件，则打开文件作为输出
	if opts.Output != "" {
		file, err := os.Create(opts.Output)
		if err != nil {
			logger.Fatalf("无法创建输出文件 %s: %v", opts.Output, err)
		}
		defer file.Close()
		out = file
	}

	run()
}

func run() {
	client, err := oss.New(opts.Endpoint, opts.AccessKeyID, opts.AccessKeySecret)
	if err != nil {
		logger.Errorf("创建 OSS 客户端失败: %v", err)
		return
	}

	if opts.BucketName != "" {
		logger.Infof("正在遍历 Bucket: %s", opts.BucketName)
		fmt.Fprintf(os.Stderr, "正在遍历 Bucket: %s\n", opts.BucketName)
		tree := walkAndGetTree(client, opts.BucketName, opts.Prefix, "")
		outputResults([]*TreeNode{tree})
	} else {
		fmt.Fprintln(os.Stderr, "正在获取 Bucket 列表...")
		logger.Info("正在获取 Bucket 列表...")
		buckets, err := ListBuckets()
		if err != nil {
			logger.Errorf("获取 Bucket 列表失败: %v", err)
			return
		}

		if len(buckets.Buckets) == 0 {
			fmt.Fprintln(os.Stderr, "当前账号下没有 Bucket。")
			logger.Info("当前账号下没有 Bucket")
			return
		}

		fmt.Fprintf(os.Stderr, "发现 %d 个 Bucket\n", len(buckets.Buckets))
		logger.Infof("发现 %d 个 Bucket", len(buckets.Buckets))

		// 使用 WaitGroup 实现并发操作
		var wg sync.WaitGroup
		// 限制并发数，防止请求过于频繁
		concurrency := 5
		semaphore := make(chan struct{}, concurrency)
		
		// 存储所有bucket的结果
		results := make([]*TreeNode, len(buckets.Buckets))
		errors := make([]error, len(buckets.Buckets))

		for index, b := range buckets.Buckets {
			index := index // 在闭包中使用
			b := b       // 在闭包中使用
			
			wg.Add(1)
			go func() {
				defer wg.Done()
				semaphore <- struct{}{} // 获取信号量
				defer func() { <-semaphore }() // 释放信号量

				fmt.Fprintf(os.Stderr, "%d/%d %s\n", index+1, len(buckets.Buckets), b.Name)
				logger.Infof("开始处理 Bucket: %s (%d/%d)", b.Name, index+1, len(buckets.Buckets))
				
				results[index] = walkAndGetTree(client, b.Name, opts.Prefix, "")
				logger.Infof("完成处理 Bucket: %s", b.Name)
			}()
		}

		wg.Wait()
		
		// 检查是否有错误
		hasError := false
		for i, err := range errors {
			if err != nil {
				logger.Errorf("处理 bucket %s 时出错: %v", buckets.Buckets[i].Name, err)
				hasError = true
			}
		}
		
		if hasError {
			logger.Warn("部分bucket处理出现错误，请查看日志")
		}
		
		outputResults(results)
	}
}

func ListBuckets() (oss.ListBucketsResult, error) {
	// 创建用于 ListBuckets 的 client（必须用全局 endpoint）
	globalClient, err := oss.New("https://oss.aliyuncs.com", opts.AccessKeyID, opts.AccessKeySecret)
	if err != nil {
		logger.Errorf("创建全局 OSS 客户端失败: %v", err)
		return oss.ListBucketsResult{}, err
	}
	return globalClient.ListBuckets()
}

// walkAndGetTree 递归遍历并构建树形结构
func walkAndGetTree(client *oss.Client, bucketName, prefix, basePath string) *TreeNode {
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		logger.Errorf("获取 Bucket %s 失败: %v", bucketName, err)
		return nil
	}

	// 创建根节点
	root := &TreeNode{
		Name:     bucketName,
		IsDir:    true,
		FullPath: "",
		Children: []*TreeNode{},
	}
	
	// 如果是指定的bucket，设置正确的名字
	if opts.BucketName != "" {
		root.Name = filepath.Base(strings.TrimRight(prefix, "/"))
		if root.Name == "" {
			root.Name = bucketName
		}
	}

	walkTreeRecursive(bucket, root, prefix, basePath)
	return root
}

// walkTreeRecursive 递归遍历OSS对象构建树形结构
func walkTreeRecursive(bucket *oss.Bucket, parentNode *TreeNode, prefix, basePath string) {
	marker := ""

	for {
		options := []oss.Option{oss.Marker(marker), oss.Delimiter("/")}
		if prefix != "" {
			options = append(options, oss.Prefix(prefix))
		}

		res, err := bucket.ListObjects(options...)
		if err != nil {
			logger.Errorf("Bucket %s 列表获取失败: %v", bucket.BucketName, err)
			return
		}

		// 处理目录（CommonPrefixes）
		for _, commonPrefix := range res.CommonPrefixes {
			dirName := baseName(commonPrefix, prefix)
			if dirName == "" {
				continue
			}

			dirNode := &TreeNode{
				Name:     dirName,
				IsDir:    true,
				FullPath: commonPrefix,
				Children: []*TreeNode{},
			}

			parentNode.Children = append(parentNode.Children, dirNode)
			// 递归处理子目录
			walkTreeRecursive(bucket, dirNode, commonPrefix, basePath+dirName+"/")
		}

		// 处理文件
		for _, obj := range res.Objects {
			if obj.Key == prefix { // 跳过目录自身（如果存在）
				continue
			}
			
			fileName := strings.TrimPrefix(obj.Key, prefix)
			if !strings.Contains(fileName, "/") && fileName != "" {
				// 检查是否应该排除该文件
				if !shouldExcludeFile(fileName) {
					fileNode := &TreeNode{
						Name:     fileName,
						IsDir:    false,
						FullPath: obj.Key,
					}
					parentNode.Children = append(parentNode.Children, fileNode)
				}
			}
		}

		if res.IsTruncated {
			marker = res.NextMarker
		} else {
			break
		}
	}
}

// baseName 提取相对于 prefix 的最后一级目录名
func baseName(commonPrefix, parentPrefix string) string {
	rel := strings.TrimPrefix(commonPrefix, parentPrefix)
	rel = strings.TrimSuffix(rel, "/")
	parts := strings.Split(rel, "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return parts[0]
}

// shouldExcludeFile 检查文件是否应该被排除
func shouldExcludeFile(filename string) bool {
	// 合并用户指定的排除列表和默认排除列表
	var excludeList []string
	
	// 如果启用了默认排除列表，则添加默认排除项
	if opts.DefaultExclude {
		defaultExcludes := []string{"mp3", "woff", "woff2", "css", "mp4", "jpg", "png", "avi", "mov"}
		excludeList = append(excludeList, defaultExcludes...)
	}
	
	// 添加用户指定的排除项
	if len(opts.Exclude) > 0 {
		excludeList = append(excludeList, opts.Exclude...)
	}
	
	// 如果没有任何排除规则，直接返回false
	if len(excludeList) == 0 {
		return false
	}

	// 获取文件扩展名
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	
	// 检查扩展名是否在排除列表中
	for _, excludeExt := range excludeList {
		if strings.ToLower(strings.TrimSpace(excludeExt)) == ext {
			logger.Debugf("排除文件 %s (扩展名: %s)", filename, ext)
			return true
		}
	}
	
	return false
}

// outputResults 输出结果到文件或控制台
func outputResults(trees []*TreeNode) {
	// 如果指定了JSON输出文件，则输出JSON格式
	if opts.JSONOutput != "" {
		file, err := os.Create(opts.JSONOutput)
		if err != nil {
			logger.Fatalf("无法创建JSON输出文件 %s: %v", opts.JSONOutput, err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(trees); err != nil {
			logger.Errorf("写入JSON文件失败: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "结果已保存至 %s (JSON格式)\n", opts.JSONOutput)
		}
	}

	// 输出文本格式到指定的out（可能是文件或stdout）
	for _, tree := range trees {
		if tree != nil {
			printTree(tree, "", true)
		}
	}
}

// printTree 打印树形结构
func printTree(node *TreeNode, indent string, isLast bool) {
	if node == nil {
		return
	}

	prefix := "├── "
	if isLast {
		prefix = "└── "
	}

	fmt.Fprint(out, indent+prefix+node.Name)
	if node.IsDir {
		fmt.Fprint(out, "/")
	}
	fmt.Fprint(out, "\n")

	if len(node.Children) > 0 {
		childIndent := indent + "│   "
		if isLast {
			childIndent = indent + "    "
		}

		for i, child := range node.Children {
			isChildLast := i == len(node.Children)-1
			printTree(child, childIndent, isChildLast)
		}
	}
}