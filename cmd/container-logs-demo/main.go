package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"docktui/internal/docker"
)

func main() {
	fmt.Println("=== Docker ContainerLogs 功能测试 ===\n")

	// 创建 Docker 客户端
	client, err := docker.NewLocalClientFromEnv()
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 获取容器列表
	containers, err := client.ListContainers(ctx, false)
	if err != nil {
		log.Fatalf("❌ 获取容器列表失败: %v\n", err)
	}

	if len(containers) == 0 {
		fmt.Println("⚠️  当前没有运行中的容器")
		os.Exit(0)
	}

	// 使用第一个运行中的容器
	containerID := containers[0].ID
	containerName := containers[0].Name
	fmt.Printf("📦 读取容器日志: %s (%s)\n\n", containerName, containerID[:12])

	// 测试 1: 获取最近 10 行日志
	fmt.Println("--- 测试 1: 获取最近 10 行日志 ---")
	testLastNLines(ctx, client, containerID, 10)

	// 测试 2: 获取带时间戳的日志
	fmt.Println("\n--- 测试 2: 获取带时间戳的最近 5 行日志 ---")
	testWithTimestamps(ctx, client, containerID, 5)

	// 测试 3: Follow 模式（可中断）
	fmt.Println("\n--- 测试 3: Follow 模式（按 Ctrl+C 中断）---")
	testFollowMode(ctx, client, containerID)

	fmt.Println("\n✅ 测试完成")
}

// testLastNLines 测试获取最近 N 行日志
func testLastNLines(ctx context.Context, client docker.Client, containerID string, tail int) {
	opts := docker.LogOptions{
		Tail:       tail,
		Timestamps: false,
	}

	logReader, err := client.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		log.Printf("❌ 获取日志失败: %v\n", err)
		return
	}
	defer logReader.Close()

	// 读取并打印日志
	printLogs(logReader, tail)
}

// testWithTimestamps 测试带时间戳的日志
func testWithTimestamps(ctx context.Context, client docker.Client, containerID string, tail int) {
	opts := docker.LogOptions{
		Tail:       tail,
		Timestamps: true,
	}

	logReader, err := client.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		log.Printf("❌ 获取日志失败: %v\n", err)
		return
	}
	defer logReader.Close()

	printLogs(logReader, tail)
}

// testFollowMode 测试 follow 模式
func testFollowMode(ctx context.Context, client docker.Client, containerID string) {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 捕获中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n⚠️  收到中断信号，停止 follow...")
		cancel()
	}()

	opts := docker.LogOptions{
		Follow:     true,
		Tail:       10,
		Timestamps: true,
	}

	logReader, err := client.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		log.Printf("❌ 获取日志失败: %v\n", err)
		return
	}
	defer logReader.Close()

	fmt.Println("📄 实时日志流（最近 10 行 + 新日志）：")
	fmt.Println("---")

	// 使用 goroutine 读取日志
	scanner := bufio.NewScanner(logReader)
	lineCount := 0

	// 设置超时（演示用，10 秒后自动停止）
	timeout := time.After(10 * time.Second)
	done := make(chan bool)

	go func() {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				done <- true
				return
			default:
				line := scanner.Text()
				// 跳过 Docker 多路复用头部（如果存在）
				if len(line) > 8 && line[0] < 32 {
					line = line[8:]
				}
				fmt.Println(line)
				lineCount++
			}
		}
		done <- true
	}()

	// 等待完成或超时
	select {
	case <-done:
		if err := scanner.Err(); err != nil && err != context.Canceled {
			log.Printf("读取日志出错: %v", err)
		}
	case <-timeout:
		fmt.Println("\n⏱️  演示超时（10秒），自动停止")
		cancel()
		<-done
	}

	fmt.Printf("---\n共读取 %d 行日志\n", lineCount)
}

// printLogs 打印日志内容
func printLogs(reader io.Reader, maxLines int) {
	scanner := bufio.NewScanner(reader)
	lineCount := 0

	fmt.Println("📄 日志内容：")
	fmt.Println("---")

	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		// Docker 日志可能包含多路复用头部（8 字节），需要处理
		// 格式: [stream_type(1字节)][padding(3字节)][size(4字节)][payload]
		if len(line) > 8 && line[0] < 32 {
			// 跳过头部 8 字节
			line = line[8:]
		}
		fmt.Println(line)
		lineCount++
	}

	fmt.Println("---")

	if err := scanner.Err(); err != nil {
		log.Printf("读取日志出错: %v", err)
	}

	if lineCount == 0 {
		fmt.Println("（无日志输出）")
	}
}
