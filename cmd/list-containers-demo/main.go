package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"docktui/internal/docker"
)

// 这是一个用于测试 ListContainers 功能的 demo
// 验证 D2 任务中实现的容器列表获取功能

func main() {
	// 从环境变量读取 DOCKER_HOST
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "默认 (Docker SDK 自动检测)"
	}

	fmt.Printf("=== 容器列表测试 ===\n")
	fmt.Printf("目标地址: %s\n\n", dockerHost)

	// 创建 Docker 客户端
	client, err := docker.NewLocalClientFromEnv()
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试连接
	fmt.Println("📡 测试连接...")
	if err := client.Ping(ctx); err != nil {
		log.Fatalf("❌ 无法连接到 Docker 守护进程: %v\n", err)
	}
	fmt.Println("✅ 连接成功\n")

	// 获取所有容器列表（包括停止的）
	fmt.Println("📦 获取容器列表（包括停止的）...")
	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		log.Fatalf("❌ 获取容器列表失败: %v\n", err)
	}

	fmt.Printf("找到 %d 个容器：\n\n", len(containers))

	if len(containers) == 0 {
		fmt.Println("  (无容器)")
	} else {
		// 打印容器列表
		for i, c := range containers {
			fmt.Printf("%d. %s\n", i+1, c.Name)
			fmt.Printf("   ID:     %s\n", c.ID[:12]) // 只显示前12位
			fmt.Printf("   镜像:   %s\n", c.Image)
			fmt.Printf("   状态:   %s (%s)\n", c.State, c.Status)
			fmt.Printf("   创建于: %s\n", c.Created.Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
	}

	// 只获取运行中的容器
	fmt.Println("🏃 获取运行中的容器...")
	runningContainers, err := client.ListContainers(ctx, false)
	if err != nil {
		log.Fatalf("❌ 获取运行中容器失败: %v\n", err)
	}

	fmt.Printf("运行中: %d 个\n", len(runningContainers))

	fmt.Println("\n✅ 测试通过！")
}
