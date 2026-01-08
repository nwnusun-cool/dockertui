package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/docker/docker/client"
)

// 这是一个用于验证 Docker SDK 连接的最小可行 demo
// 用途：
// 1. 验证能够连接到远程 Docker (TCP endpoint)
// 2. 打印 Docker 版本信息
// 3. 确认 Windows 环境下的 endpoint 配置方式
//
// 使用方式：
// 本地 Docker Desktop (默认):
//   go run ./cmd/docker-version-demo
//
// 远程 Docker (TCP):
//   set DOCKER_HOST=tcp://192.168.3.49:2375
//   go run ./cmd/docker-version-demo
//
// Windows Docker Desktop (named pipe):
//   set DOCKER_HOST=npipe:////./pipe/docker_engine
//   go run ./cmd/docker-version-demo

func main() {
	// 从环境变量读取 DOCKER_HOST，如果未设置则使用默认值
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "默认 (Docker SDK 自动检测)"
	}

	fmt.Printf("=== Docker SDK 连接测试 ===\n")
	fmt.Printf("目标地址: %s\n\n", dockerHost)

	// 创建 Docker 客户端
	// FromEnv: 从环境变量读取配置 (DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH 等)
	// WithAPIVersionNegotiation: 自动协商 API 版本，避免版本不匹配问题
	cli, err := sdk.NewClientWithOpts(
		sdk.FromEnv,
		sdk.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer cli.Close()

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Ping 测试连接
	fmt.Println("📡 测试连接...")
	ping, err := cli.Ping(ctx, sdk.PingOptions{})
	if err != nil {
		log.Fatalf("❌ 无法连接到 Docker 守护进程: %v\n", err)
	}
	fmt.Printf("✅ 连接成功\n")
	fmt.Printf("   API 版本: %s\n", ping.APIVersion)
	fmt.Printf("   实验性功能: %v\n\n", ping.Experimental)

	// 2. 获取服务器版本信息
	fmt.Println("📋 Docker 版本信息:")
	version, err := cli.ServerVersion(ctx, sdk.ServerVersionOptions{})
	if err != nil {
		log.Fatalf("❌ 获取版本信息失败: %v\n", err)
	}

	fmt.Printf("   版本: %s\n", version.Version)
	fmt.Printf("   API 版本: %s\n", version.APIVersion)
	fmt.Printf("   最低 API 版本: %s\n", version.MinAPIVersion)
	fmt.Printf("   操作系统: %s\n", version.Os)
	fmt.Printf("   架构: %s\n\n", version.Arch)

	// 3. 获取系统信息（可选，用于进一步验证）
	fmt.Println("🖥️  Docker 系统信息:")
	info, err := cli.Info(ctx, sdk.InfoOptions{})
	if err != nil {
		log.Printf("⚠️  获取系统信息失败: %v\n", err)
	} else {
		fmt.Printf("   容器数: %d (运行中: %d, 暂停: %d, 停止: %d)\n",
			info.Info.Containers, info.Info.ContainersRunning, info.Info.ContainersPaused, info.Info.ContainersStopped)
		fmt.Printf("   镜像数: %d\n", info.Info.Images)
		fmt.Printf("   服务器版本: %s\n", info.Info.ServerVersion)
		fmt.Printf("   存储驱动: %s\n", info.Info.Driver)
		fmt.Printf("   日志驱动: %s\n", info.Info.LoggingDriver)
		fmt.Printf("   操作系统类型: %s\n", info.Info.OSType)
		fmt.Printf("   架构: %s\n", info.Info.Architecture)
		fmt.Printf("   CPU 数: %d\n", info.Info.NCPU)
		fmt.Printf("   总内存: %.2f GB\n", float64(info.Info.MemTotal)/(1024*1024*1024))
	}

	fmt.Println("\n✅ 所有测试通过！")
}
