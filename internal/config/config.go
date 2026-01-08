package config

import (
	"os"
	"time"
)

// Config 描述 docktui 运行所需的基础配置。
// 目前只关心本地 Docker 连接和一些超时设置，后续可扩展 docker-compose 相关字段。
type Config struct {
	DockerHost     string        // Docker 守护进程地址，默认走环境变量
	RequestTimeout time.Duration // 与 Docker 通信的默认超时时间
}

// Load 从环境变量加载配置，并填充合理默认值。
func Load() (*Config, error) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		// 留空表示使用 Docker SDK 的默认行为（Unix socket / named pipe 等）
	}

	cfg := &Config{
		DockerHost:     host,
		RequestTimeout: 10 * time.Second,
	}
	return cfg, nil
}
