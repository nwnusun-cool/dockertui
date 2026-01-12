package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/docker/docker/client"
)

// Docker SDK 连接验证 demo
// 用法:
//   go run ./examples/docker-version
//   DOCKER_HOST=tcp://192.168.1.100:2375 go run ./examples/docker-version

func main() {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "默认 (Docker SDK 自动检测)"
	}

	fmt.Printf("=== Docker SDK 连接测试 ===\n")
	fmt.Printf("目标地址: %s\n\n", dockerHost)

	cli, err := sdk.NewClientWithOpts(
		sdk.FromEnv,
		sdk.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("📡 测试连接...")
	ping, err := cli.Ping(ctx, sdk.PingOptions{})
	if err != nil {
		log.Fatalf("❌ 无法连接到 Docker 守护进程: %v\n", err)
	}
	fmt.Printf("✅ 连接成功 (API: %s)\n\n", ping.APIVersion)

	version, err := cli.ServerVersion(ctx, sdk.ServerVersionOptions{})
	if err != nil {
		log.Fatalf("❌ 获取版本信息失败: %v\n", err)
	}

	fmt.Printf("📋 Docker 版本: %s\n", version.Version)
	fmt.Printf("   操作系统: %s/%s\n", version.Os, version.Arch)

	fmt.Println("\n✅ 测试通过！")
}
