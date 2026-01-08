package main

import (
	"fmt"
	"os"

	"docktui/internal/compose"
)

func main() {
	fmt.Println("=== Docker Compose Client Demo ===")
	fmt.Println()

	// 创建客户端
	client, err := compose.NewClient()
	if err != nil {
		fmt.Printf("❌ 创建客户端失败: %v\n", err)
		os.Exit(1)
	}

	// 显示版本信息
	version, err := client.Version()
	if err != nil {
		fmt.Printf("❌ 获取版本失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 检测到 compose 命令\n")
	fmt.Printf("   命令类型: %s\n", client.CommandType())
	fmt.Printf("   版本: %s\n", version)
	fmt.Println()

	// 如果提供了项目路径，尝试获取服务状态
	if len(os.Args) > 1 {
		projectPath := os.Args[1]
		fmt.Printf("📁 项目路径: %s\n", projectPath)
		fmt.Println()

		project := &compose.Project{
			Path:         projectPath,
			ComposeFiles: []string{"docker-compose.yml"},
		}

		// 获取服务状态
		fmt.Println("📋 服务状态:")
		services, err := client.PS(project)
		if err != nil {
			fmt.Printf("   ❌ 获取服务状态失败: %v\n", err)
		} else if len(services) == 0 {
			fmt.Println("   (无运行中的服务)")
		} else {
			for _, svc := range services {
				statusIcon := "⚪"
				switch svc.State {
				case "running":
					statusIcon = "🟢"
				case "partial":
					statusIcon = "🟡"
				case "exited":
					statusIcon = "⚪"
				}
				fmt.Printf("   %s %s: %s (%d/%d)\n", statusIcon, svc.Name, svc.State, svc.Running, svc.Replicas)
			}
		}
		fmt.Println()

		// 获取配置
		fmt.Println("📄 Compose 配置:")
		config, err := client.Config(project)
		if err != nil {
			fmt.Printf("   ❌ 获取配置失败: %v\n", err)
		} else {
			// 只显示前 500 字符
			if len(config) > 500 {
				config = config[:500] + "\n   ... (truncated)"
			}
			fmt.Println(config)
		}
	} else {
		fmt.Println("💡 提示: 可以传入项目路径作为参数来测试更多功能")
		fmt.Println("   例如: go run ./cmd/compose-demo /path/to/compose/project")
	}

	fmt.Println()
	fmt.Println("=== Demo 完成 ===")
}
