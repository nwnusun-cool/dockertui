package docker

import (
	"bufio"
	"io"
)

// StdType 标准流类型
type StdType byte

const (
	// Stdin 标准输入流
	Stdin StdType = 0
	// Stdout 标准输出流
	Stdout StdType = 1
	// Stderr 标准错误流
	Stderr StdType = 2
)

// LogEntry 表示一条日志记录
type LogEntry struct {
	Stream  StdType // 流类型（stdout 或 stderr）
	Content string  // 日志内容
}

// DemuxReader 将 Docker 多路复用格式的日志流转换为普通文本流
// Docker 日志使用多路复用格式：[stream_type(1字节)][padding(3字节)][size(4字节)][payload]
type DemuxReader struct {
	reader io.Reader
}

// NewDemuxReader 创建一个新的日志解复用读取器
func NewDemuxReader(reader io.Reader) *DemuxReader {
	return &DemuxReader{reader: reader}
}

// Read 实现 io.Reader 接口，自动处理多路复用格式
func (d *DemuxReader) Read(p []byte) (n int, err error) {
	// 读取 8 字节的头部
	header := make([]byte, 8)
	_, err = io.ReadFull(d.reader, header)
	if err != nil {
		return 0, err
	}

	// 解析头部：
	// header[0]: stream type (0=stdin, 1=stdout, 2=stderr)
	// header[1-3]: padding (unused)
	// header[4-7]: size (big-endian uint32)
	size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])

	// 读取实际数据
	if size > len(p) {
		size = len(p)
	}

	return io.ReadFull(d.reader, p[:size])
}

// ReadAllLogs 读取所有日志并返回字符串切片
// 自动处理 Docker 的多路复用格式
func ReadAllLogs(reader io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(reader)

	// 增加缓冲区大小以处理长日志行
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		// 如果是多路复用格式，跳过头部
		if len(line) > 8 && line[0] < 32 {
			line = line[8:]
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return lines, err
	}

	return lines, nil
}

// StreamLogs 流式读取日志，通过 channel 发送每一行
// 适用于 follow 模式或大量日志的场景
func StreamLogs(reader io.Reader, lineChan chan<- string, errChan chan<- error) {
	defer close(lineChan)
	defer close(errChan)

	scanner := bufio.NewScanner(reader)

	// 增加缓冲区大小
	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		// 如果是多路复用格式，跳过头部
		if len(line) > 8 && line[0] < 32 {
			line = line[8:]
		}
		lineChan <- line
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
	}
}
