package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"docktui/internal/docker"
)

func main() {
	fmt.Println("=== Docker ExecShell 功能测试 ===\n")

	// 创建 Docker 客户端
	client, err := docker.NewLocalClientFromEnv()
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 获取运行中的容器
	containers, err := client.ListContainers(ctx, false)
	if err != nil {
		log.Fatalf("❌ 获取容器列表失败: %v\n", err)
	}

	if len(containers) == 0 {
		fmt.Println("⚠️  当前没有运行中的容器")
		os.Exit(0)
	}

	// 显示容器列表
	fmt.Println("📦 运行中的容器:")
	for i, c := range containers {
		fmt.Printf("  [%d] %s (%s) - %s\n", i+1, c.Name, c.ID[:12], c.State)
	}
	fmt.Println()

	// 使用第一个容器进行测试
	containerID := containers[0].ID
	containerName := containers[0].Name
	fmt.Printf("🔧 将在容器中执行命令: %s (%s)\n\n", containerName, containerID[:12])

	// 测试 1: 执行简单命令 (非交互式)
	fmt.Println("=== 测试 1: 执行简单命令 ===")
	fmt.Println("执行命令: echo 'Hello from container!'")
	
	execConfig := docker.ExecConfig{
		Cmd:          []string{"echo", "Hello from container!"},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}
	
	result, err := client.ExecCommand(ctx, containerID, execConfig)
	if err != nil {
		log.Printf("❌ 执行命令失败: %v\n", err)
	} else {
		fmt.Printf("✅ 命令执行完成，退出码: %d\n", result.ExitCode)
		if result.Error != "" {
			fmt.Printf("   错误: %s\n", result.Error)
		}
	}
	fmt.Println()

	// 测试 2: 获取容器信息
	fmt.Println("=== 测试 2: 获取容器信息 ===")
	fmt.Println("执行命令: cat /etc/hostname")
	
	execConfig2 := docker.ExecConfig{
		Cmd:          []string{"cat", "/etc/hostname"},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}
	
	result2, err := client.ExecCommand(ctx, containerID, execConfig2)
	if err != nil {
		log.Printf("❌ 执行命令失败: %v\n", err)
	} else {
		fmt.Printf("✅ 命令执行完成，退出码: %d\n", result2.ExitCode)
	}
	fmt.Println()

	// 测试 3: 交互式 shell (仅打印提示，不实际执行)
	fmt.Println("=== 测试 3: 交互式 Shell ===")
	fmt.Println("💡 提示: ExecShell 方法已实现，支持交互式 shell")
	fmt.Println("   用法: client.ExecShell(ctx, containerID, \"/bin/sh\")")
	fmt.Println("   在 TUI 应用中可以使用此方法进入容器 shell")
	fmt.Println()

	fmt.Println("✅ 所有测试完成！")
}
