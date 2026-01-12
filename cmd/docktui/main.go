package main

import (
	"context"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"docktui/internal/config"
	"docktui/internal/docker"
	"docktui/internal/i18n"
	"docktui/internal/ui"
)

func main() {
	// 初始化 i18n
	i18n.Init()
	
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	_ = cfg

	// 尝试连接 Docker
	dockerClient, err := docker.NewLocalClientFromEnv()
	var dockerConnected bool
	var dockerError string
	
	if err != nil {
		dockerConnected = false
		dockerError = err.Error()
	} else {
		// 测试 Docker 连接
		if err := dockerClient.Ping(context.Background()); err != nil {
			dockerConnected = false
			dockerError = err.Error()
		} else {
			dockerConnected = true
		}
	}

	// 即使 Docker 连接失败，也启动 TUI 并显示错误信息
	m := ui.NewModel(dockerClient)
	
	// 设置 Docker 连接状态
	if !dockerConnected {
		m = ui.SetDockerError(m, dockerError)
	}
	
	// 创建 TUI 程序，使用 alternate screen buffer
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // 使用替代屏幕缓冲区
		tea.WithMouseCellMotion(), // 启用鼠标支持（可选）
	)
	
	if err := p.Start(); err != nil {
		log.Fatalf("启动 TUI 失败: %v", err)
	}
}
