package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sdk "github.com/docker/docker/client"

	"docktui/internal/compose"
)

func main() {
	ctx := context.Background()

	dockerCli, err := sdk.NewClientWithOpts(
		sdk.FromEnv,
		sdk.WithAPIVersionNegotiation(),
	)
	if err != nil {
		fmt.Printf("创建 Docker 客户端失败: %v\n", err)
		os.Exit(1)
	}
	defer dockerCli.Close()

	_, err = dockerCli.Ping(ctx)
	if err != nil {
		fmt.Printf("连接 Docker 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Docker 连接成功")

	manager, err := compose.NewManager(dockerCli)
	if err != nil {
		fmt.Printf("创建 Compose 管理器失败: %v\n", err)
		os.Exit(1)
	}

	version, err := manager.GetVersion()
	if err != nil {
		fmt.Printf("获取 compose 版本失败: %v\n", err)
	} else {
		fmt.Printf("✓ Compose 版本: %s (%s)\n", version, manager.GetCommandType())
	}

	fmt.Println("\n========== 发现 Compose 项目 ==========")

	start := time.Now()
	projects, err := manager.DiscoverProjects(ctx)
	if err != nil {
		fmt.Printf("发现项目失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 发现 %d 个项目 (耗时 %v)\n", len(projects), time.Since(start))

	if len(projects) == 0 {
		fmt.Println("\n没有发现运行中的 Compose 项目")
		return
	}

	for i, p := range projects {
		fmt.Printf("\n[%d] 项目: %s\n", i+1, p.Name)
		fmt.Printf("    路径: %s\n", p.Path)
		fmt.Printf("    状态: %s\n", p.Status.String())
		fmt.Printf("    服务数: %d\n", len(p.Services))

		for _, svc := range p.Services {
			fmt.Printf("      - %s: %s (%d/%d 运行中)\n",
				svc.Name, svc.State, svc.Running, svc.Replicas)
		}
	}
}
