package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"docktui/internal/docker"
)

func main() {
	fmt.Println("=== Docker ContainerDetails 功能测试 ===\n")

	// 创建 Docker 客户端
	client, err := docker.NewLocalClientFromEnv()
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 获取容器列表
	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		log.Fatalf("❌ 获取容器列表失败: %v\n", err)
	}

	if len(containers) == 0 {
		fmt.Println("⚠️  当前没有容器")
		os.Exit(0)
	}

	// 测试第一个容器的详情
	containerID := containers[0].ID
	fmt.Printf("📦 获取容器详情: %s (%s)\n\n", containers[0].Name, containerID[:12])

	// 调用 ContainerDetails
	details, err := client.ContainerDetails(ctx, containerID)
	if err != nil {
		log.Fatalf("❌ 获取容器详情失败: %v\n", err)
	}

	// 打印详情信息
	fmt.Println("✅ 容器详情：")
	fmt.Printf("  ID: %s\n", details.ID[:12])
	fmt.Printf("  名称: %s\n", details.Name)
	fmt.Printf("  镜像: %s\n", details.Image)
	fmt.Printf("  状态: %s\n", details.State)
	fmt.Printf("  状态描述: %s\n", details.Status)
	fmt.Printf("  创建时间: %s\n", details.Created.Format("2006-01-02 15:04:05"))
	
	// 打印端口映射
	if len(details.Ports) > 0 {
		fmt.Printf("\n📡 端口映射 (%d 个):\n", len(details.Ports))
		for _, port := range details.Ports {
			fmt.Printf("  %s:%d -> %d/%s\n", port.IP, port.PublicPort, port.PrivatePort, port.Type)
		}
	}

	// 打印挂载点
	if len(details.Mounts) > 0 {
		fmt.Printf("\n💾 挂载点 (%d 个):\n", len(details.Mounts))
		for _, mount := range details.Mounts {
			fmt.Printf("  [%s] %s -> %s (%s)\n", mount.Type, mount.Source, mount.Destination, mount.Mode)
		}
	}

	// 打印环境变量
	if len(details.Env) > 0 {
		fmt.Printf("\n🔧 环境变量 (%d 个):\n", len(details.Env))
		// 只显示前 5 个
		for i, env := range details.Env {
			if i >= 5 {
				fmt.Printf("  ... 还有 %d 个\n", len(details.Env)-5)
				break
			}
			fmt.Printf("  %s\n", env)
		}
	}

	// 打印标签
	if len(details.Labels) > 0 {
		fmt.Printf("\n🏷️  标签 (%d 个):\n", len(details.Labels))
		count := 0
		for k, v := range details.Labels {
			if count >= 3 {
				fmt.Printf("  ... 还有 %d 个\n", len(details.Labels)-3)
				break
			}
			fmt.Printf("  %s = %s\n", k, v)
			count++
		}
	}

	fmt.Println("\n✅ 测试完成")
}
