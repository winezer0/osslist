package util

import (
	"bufio"
	"fmt"
	"os"
	"osslist/pkg/logging"
)

// FileReceiver 处理文件写入逻辑
type FileReceiver struct {
	file   *os.File
	writer *bufio.Writer
}

// NewFileReceiver 创建一个新的 FileReceiver
// 如果 path 为空，则输出到 stdout (但 bufio 可能不适用于 stdout 的实时性要求，不过这里是流式追加，问题不大)
// 实际上 main.go 中 default 是 osslist.txt，所以 path 通常不为空
func NewFileReceiver(path string) (*FileReceiver, error) {
	var f *os.File
	var err error

	if path == "" {
		f = os.Stdout
	} else {
		// 使用 O_APPEND 模式打开文件，如果文件不存在则创建
		f, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("Unable to create the output file %s: %v", path, err)
		}
	}

	return &FileReceiver{
		file:   f,
		writer: bufio.NewWriter(f),
	}, nil
}

// Start 开始从 channel 接收数据并写入文件
func (fr *FileReceiver) Start(ch <-chan string, done chan<- struct{}) {
	defer func() {
		// 刷新缓冲区
		if err := fr.writer.Flush(); err != nil {
			logging.Errorf("failed to refresh the buffer: %v", err)
		}
		// 如果不是 stdout，关闭文件
		if fr.file != os.Stdout {
			if err := fr.file.Close(); err != nil {
				logging.Errorf("failed to close the file: %v", err)
			}
		}
		// 通知主线程完成
		close(done)
	}()

	for path := range ch {
		if _, err := fmt.Fprintln(fr.writer, path); err != nil {
			logging.Errorf("failed to write to the file: %v", err)
		}
	}
}
