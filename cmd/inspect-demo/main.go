package main

import (
	"context"
	"fmt"
	"log"
	"os"

	sdk "github.com/docker/docker/client"
)

func main() {
	fmt.Println("=== Docker ContainerInspect API 测试 ===")

	// 创建 Docker 客户端
	cli, err := sdk.NewClientWithOpts(
		sdk.FromEnv,
		sdk.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("❌ 创建 Docker 客户端失败: %v\n", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// 获取容器列表
	containers, err := cli.ContainerList(ctx, sdk.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("❌ 获取容器列表失败: %v\n", err)
	}

	if len(containers.Items) == 0 {
		fmt.Println("⚠️  当前没有容器")
		os.Exit(0)
	}

	// 取第一个容器进行测试
	containerID := containers.Items[0].ID
	fmt.Printf("📦 检查容器: %s\n\n", containerID)

	// 调用 ContainerInspect
	inspect, err := cli.ContainerInspect(ctx, containerID, sdk.ContainerInspectOptions{})
	if err != nil {
		log.Fatalf("❌ 获取容器详情失败: %v\n", err)
	}

	// 打印结构信息
	fmt.Printf("ContainerInspect 返回类型: %T\n\n", inspect)
	
	// 打印端口信息
	if inspect.Container.NetworkSettings != nil {
		fmt.Println("端口映射:")
		for port, bindings := range inspect.Container.NetworkSettings.Ports {
			fmt.Printf("  Port 类型: %T, 值: %+v\n", port, port)
			for _, binding := range bindings {
				fmt.Printf("    Binding: %+v\n", binding)
			}
		}
	}
	
	fmt.Println("\n✅ 测试完成")
}
