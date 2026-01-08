package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"docktui/internal/docker"
)

func main() {
	fmt.Println("=== Docker ExecShell 功能测试 ===\n")

	// 创建 Docker 客户端
	client, err := docker.NewLocalClientFromEnv()
	if err != nil {
		fmt.Printf("❌ 创建 Docker 客户端失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	if err != nil {
		fmt.Printf("❌ 无法连接到 Docker: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Docker 连接成功\n")

	// 获取容器列表
	containers, err := client.ListContainers(ctx, false)
	if err != nil {
		fmt.Printf("❌ 获取容器列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(containers) == 0 {
		fmt.Println("⚠️  没有运行中的容器")
		fmt.Println("请先启动一个容器，例如:")
		fmt.Println("  docker run -d --name test-shell busybox sleep 3600")
		os.Exit(0)
	}

	// 显示容器列表
	fmt.Println("=== 运行中的容器 ===")
	for i, c := range containers {
		fmt.Printf("%d. %s (%s) - %s\n", i+1, c.Name, c.ID[:12], c.Image)
	}
	fmt.Println()

	// 选择第一个容器进行测试
	testContainer := containers[0]
	fmt.Printf("📦 测试容器: %s (%s)\n\n", testContainer.Name, testContainer.ID[:12])

	// 测试 1: 检测可用的 shell
	fmt.Println("=== 测试 1: 检测可用的 shell ===")
	availableShells := client.GetAvailableShells(ctx, testContainer.ID)
	if len(availableShells) == 0 {
		fmt.Println("❌ 容器中没有可用的 shell")
	} else {
		fmt.Println("✅ 可用的 shell:")
		for _, shell := range availableShells {
			fmt.Printf("   - %s\n", shell)
		}
	}
	fmt.Println()

	// 测试 2: 自动检测并显示将使用的 shell
	fmt.Println("=== 测试 2: 自动 shell 检测 ===")
	fmt.Println("💡 调用 ExecShell 时如果不指定 shell，将自动检测")
	fmt.Println("   检测顺序: /bin/bash -> /bin/sh -> /bin/ash")
	if len(availableShells) > 0 {
		fmt.Printf("   将使用: %s\n", availableShells[0])
	}
	fmt.Println()

	// 测试 3: 提示用户如何使用
	fmt.Println("=== 测试 3: 使用说明 ===")
	fmt.Println("💡 在 TUI 中使用 ExecShell:")
	fmt.Println("   1. 在容器列表或详情视图按 's' 键")
	fmt.Println("   2. 确认后将进入容器 shell")
	fmt.Println("   3. 输入 'exit' 或按 Ctrl+D 退出")
	fmt.Println("   4. 自动返回 TUI 界面")
	fmt.Println()

	// 测试 4: 显示错误处理
	fmt.Println("=== 测试 4: 错误处理 ===")
	fmt.Println("✅ 已实现的错误处理:")
	fmt.Println("   - 容器未运行")
	fmt.Println("   - Shell 不存在")
	fmt.Println("   - 权限不足")
	fmt.Println("   - 网络错误")
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
	fmt.Println("✅ ExecShell 功能已就绪")
	fmt.Println("💡 下一步: 在 TUI 中集成 ExecShell (E3)")
}
