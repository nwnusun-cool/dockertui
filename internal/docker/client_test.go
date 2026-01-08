package docker

import (
	"context"
	"testing"
	"time"
)

// TestContainer_DataStructure 测试 Container 数据结构
func TestContainer_DataStructure(t *testing.T) {
	now := time.Now()
	container := Container{
		ID:      "abc123",
		Name:    "test-container",
		Image:   "nginx:latest",
		State:   "running",
		Status:  "Up 2 hours",
		Created: now,
	}

	if container.ID != "abc123" {
		t.Errorf("Expected ID 'abc123', got '%s'", container.ID)
	}
	if container.Name != "test-container" {
		t.Errorf("Expected Name 'test-container', got '%s'", container.Name)
	}
	if container.Image != "nginx:latest" {
		t.Errorf("Expected Image 'nginx:latest', got '%s'", container.Image)
	}
	if container.State != "running" {
		t.Errorf("Expected State 'running', got '%s'", container.State)
	}
	if container.Status != "Up 2 hours" {
		t.Errorf("Expected Status 'Up 2 hours', got '%s'", container.Status)
	}
	if !container.Created.Equal(now) {
		t.Errorf("Expected Created time to match, got difference")
	}
}

// TestLocalClient_NilCheck 测试空客户端的错误处理
func TestLocalClient_NilCheck(t *testing.T) {
	var client *LocalClient
	ctx := context.Background()

	// 测试 Ping
	err := client.Ping(ctx)
	if err == nil {
		t.Error("Expected error for nil client Ping, got nil")
	}
	if err.Error() != "Docker 客户端未初始化" {
		t.Errorf("Expected '客户端未初始化' error, got: %v", err)
	}

	// 测试 ListContainers
	_, err = client.ListContainers(ctx, true)
	if err == nil {
		t.Error("Expected error for nil client ListContainers, got nil")
	}
	if err.Error() != "Docker 客户端未初始化" {
		t.Errorf("Expected '客户端未初始化' error, got: %v", err)
	}

	// 测试 Close
	err = client.Close()
	if err != nil {
		t.Errorf("Expected nil error for nil client Close, got: %v", err)
	}
}

// TestLocalClient_UnimplementedMethods 测试尚未实现的方法返回正确的错误
func TestLocalClient_UnimplementedMethods(t *testing.T) {
	// 注意：空的 LocalClient 会先检查 cli 是否为 nil
	// 所以即使方法未实现，也会返回"客户端未初始化"错误
	client := &LocalClient{}
	ctx := context.Background()

	// 测试 ContainerDetails
	_, err := client.ContainerDetails(ctx, "test-id")
	if err == nil {
		t.Error("Expected error for unimplemented ContainerDetails, got nil")
	}
	// 空客户端会返回"未初始化"或"尚未实现"
	if err.Error() != "Docker 客户端未初始化" && err.Error() != "容器详情获取功能尚未实现" {
		t.Errorf("Expected '未初始化' or '尚未实现' error, got: %v", err)
	}

	// 测试 ContainerLogs
	_, err = client.ContainerLogs(ctx, "test-id", LogOptions{})
	if err == nil {
		t.Error("Expected error for unimplemented ContainerLogs, got nil")
	}
	if err.Error() != "Docker 客户端未初始化" && err.Error() != "日志读取功能尚未实现" {
		t.Errorf("Expected '未初始化' or '尚未实现' error, got: %v", err)
	}

	// 测试 ExecShell
	err = client.ExecShell(ctx, "test-id", "/bin/bash")
	if err == nil {
		t.Error("Expected error for unimplemented ExecShell, got nil")
	}
	if err.Error() != "Docker 客户端未初始化" && err.Error() != "exec shell 功能尚未实现" {
		t.Errorf("Expected '未初始化' or '尚未实现' error, got: %v", err)
	}
}

// TestLogOptions_DefaultValues 测试 LogOptions 结构的默认值处理
func TestLogOptions_DefaultValues(t *testing.T) {
	opts := LogOptions{}

	if opts.Follow != false {
		t.Errorf("Expected Follow default to false, got %v", opts.Follow)
	}
	if opts.Tail != 0 {
		t.Errorf("Expected Tail default to 0, got %v", opts.Tail)
	}
	if opts.Timestamps != false {
		t.Errorf("Expected Timestamps default to false, got %v", opts.Timestamps)
	}
	if opts.Since != "" {
		t.Errorf("Expected Since default to empty string, got %v", opts.Since)
	}
}

// TestContainerDetails_DataStructure 测试 ContainerDetails 数据结构
func TestContainerDetails_DataStructure(t *testing.T) {
	now := time.Now()
	details := ContainerDetails{
		ID:      "abc123",
		Name:    "test-container",
		Image:   "nginx:latest",
		State:   "running",
		Status:  "Up 2 hours",
		Created: now,
		Ports: []PortMapping{
			{PrivatePort: 80, PublicPort: 8080, Type: "tcp", IP: "0.0.0.0"},
		},
		Mounts: []MountInfo{
			{Type: "volume", Source: "my-vol", Destination: "/data", Mode: "rw"},
		},
		Env:    []string{"ENV=production", "DEBUG=false"},
		Labels: map[string]string{"app": "web", "version": "1.0"},
	}

	if len(details.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(details.Ports))
	}
	if details.Ports[0].PublicPort != 8080 {
		t.Errorf("Expected PublicPort 8080, got %d", details.Ports[0].PublicPort)
	}

	if len(details.Mounts) != 1 {
		t.Errorf("Expected 1 mount, got %d", len(details.Mounts))
	}
	if details.Mounts[0].Type != "volume" {
		t.Errorf("Expected mount type 'volume', got '%s'", details.Mounts[0].Type)
	}

	if len(details.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(details.Env))
	}

	if len(details.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(details.Labels))
	}
	if details.Labels["app"] != "web" {
		t.Errorf("Expected label 'app'='web', got '%s'", details.Labels["app"])
	}
}

// 注意：以下是集成测试，需要真实的 Docker 环境
// 在 CI/CD 或本地开发时，可以通过环境变量控制是否跳过

// TestListContainers_Integration 是一个集成测试示例
// 需要设置 DOCKER_HOST 环境变量指向可用的 Docker 守护进程
// 使用 go test -short 可以跳过此测试
func TestListContainers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 尝试创建客户端
	client, err := NewLocalClientFromEnv()
	if err != nil {
		t.Skipf("Cannot create Docker client (Docker not available?): %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 测试 Ping
	if err := client.Ping(ctx); err != nil {
		t.Skipf("Cannot ping Docker daemon: %v", err)
		return
	}

	// 测试获取所有容器
	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		t.Fatalf("ListContainers(all=true) failed: %v", err)
	}

	t.Logf("Found %d containers (all)", len(containers))

	// 验证返回的数据结构
	for _, c := range containers {
		if c.ID == "" {
			t.Error("Container ID should not be empty")
		}
		if c.Image == "" {
			t.Error("Container Image should not be empty")
		}
		// State 和 Status 可能为空，取决于 Docker 版本
	}

	// 测试仅获取运行中的容器
	runningContainers, err := client.ListContainers(ctx, false)
	if err != nil {
		t.Fatalf("ListContainers(all=false) failed: %v", err)
	}

	t.Logf("Found %d running containers", len(runningContainers))

	// 运行中的容器数量应该 <= 所有容器数量
	if len(runningContainers) > len(containers) {
		t.Errorf("Running containers (%d) should not exceed total containers (%d)",
			len(runningContainers), len(containers))
	}
}

// TestContainerDetails_Integration 集成测试 ContainerDetails 方法
func TestContainerDetails_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建客户端
	client, err := NewLocalClientFromEnv()
	if err != nil {
		t.Skipf("Cannot create Docker client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 首先获取容器列表
	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		t.Skipf("Cannot list containers: %v", err)
		return
	}

	if len(containers) == 0 {
		t.Skip("No containers available for testing")
		return
	}

	// 测试获取第一个容器的详情
	containerID := containers[0].ID
	t.Logf("Testing container details for: %s (%s)", containers[0].Name, containerID[:12])

	details, err := client.ContainerDetails(ctx, containerID)
	if err != nil {
		t.Fatalf("ContainerDetails failed: %v", err)
	}

	// 验证返回的数据结构
	if details == nil {
		t.Fatal("Expected non-nil ContainerDetails")
	}

	if details.ID == "" {
		t.Error("Container ID should not be empty")
	}

	if details.Name == "" {
		t.Error("Container Name should not be empty")
	}

	if details.Image == "" {
		t.Error("Container Image should not be empty")
	}

	if details.State == "" {
		t.Error("Container State should not be empty")
	}

	// Ports, Mounts, Env, Labels 可能为空，但不应该是 nil
	if details.Ports == nil {
		t.Error("Ports slice should not be nil")
	}
	if details.Mounts == nil {
		t.Error("Mounts slice should not be nil")
	}
	if details.Env == nil {
		t.Error("Env slice should not be nil")
	}
	if details.Labels == nil {
		t.Error("Labels map should not be nil")
	}

	t.Logf("Container details retrieved successfully:")
	t.Logf("  Name: %s", details.Name)
	t.Logf("  Image: %s", details.Image)
	t.Logf("  State: %s", details.State)
	t.Logf("  Ports: %d", len(details.Ports))
	t.Logf("  Mounts: %d", len(details.Mounts))
	t.Logf("  Env vars: %d", len(details.Env))
	t.Logf("  Labels: %d", len(details.Labels))

	// 测试不存在的容器
	_, err = client.ContainerDetails(ctx, "nonexistent-container-id")
	if err == nil {
		t.Error("Expected error for nonexistent container, got nil")
	}
}

// TestContainerLogs_Integration 集成测试 ContainerLogs 方法
func TestContainerLogs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建客户端
	client, err := NewLocalClientFromEnv()
	if err != nil {
		t.Skipf("Cannot create Docker client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 获取运行中的容器
	containers, err := client.ListContainers(ctx, false)
	if err != nil {
		t.Skipf("Cannot list containers: %v", err)
		return
	}

	if len(containers) == 0 {
		t.Skip("No running containers available for testing")
		return
	}

	// 测试获取第一个容器的日志
	containerID := containers[0].ID
	t.Logf("Testing container logs for: %s (%s)", containers[0].Name, containerID[:12])

	// 测试 1: 获取最近 10 行日志
	t.Run("LastNLines", func(t *testing.T) {
		opts := LogOptions{
			Tail:       10,
			Timestamps: false,
		}

		logReader, err := client.ContainerLogs(ctx, containerID, opts)
		if err != nil {
			t.Fatalf("ContainerLogs failed: %v", err)
		}
		defer logReader.Close()

		// 读取日志
		logs, err := ReadAllLogs(logReader)
		if err != nil {
			t.Fatalf("Failed to read logs: %v", err)
		}

		t.Logf("Read %d lines of logs", len(logs))

		// 验证日志不为空（如果容器有输出）
		if len(logs) > 0 {
			t.Logf("First log line: %s", logs[0])
		}
	})

	// 测试 2: 带时间戳
	t.Run("WithTimestamps", func(t *testing.T) {
		opts := LogOptions{
			Tail:       5,
			Timestamps: true,
		}

		logReader, err := client.ContainerLogs(ctx, containerID, opts)
		if err != nil {
			t.Fatalf("ContainerLogs failed: %v", err)
		}
		defer logReader.Close()

		logs, err := ReadAllLogs(logReader)
		if err != nil {
			t.Fatalf("Failed to read logs: %v", err)
		}

		t.Logf("Read %d lines with timestamps", len(logs))

		// 验证日志包含时间戳（如果有日志）
		if len(logs) > 0 && len(logs[0]) > 20 {
			// 时间戳格式通常以年份开头，如 "2026-01-07..."
			t.Logf("First log with timestamp: %s", logs[0][:50])
		}
	})

	// 测试 3: Follow 模式（短时间）
	t.Run("FollowMode", func(t *testing.T) {
		// 创建带超时的上下文
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		opts := LogOptions{
			Follow:     true,
			Tail:       5,
			Timestamps: false,
		}

		logReader, err := client.ContainerLogs(ctxWithTimeout, containerID, opts)
		if err != nil {
			t.Fatalf("ContainerLogs failed: %v", err)
		}
		defer logReader.Close()

		// 读取日志（最多 2 秒）
		lineChan := make(chan string, 10)
		errChan := make(chan error, 1)

		go StreamLogs(logReader, lineChan, errChan)

		lineCount := 0
		timeout := time.After(2 * time.Second)

	loop:
		for {
			select {
			case line, ok := <-lineChan:
				if !ok {
					break loop
				}
				lineCount++
				if lineCount == 1 {
					t.Logf("First follow log: %s", line)
				}
			case err := <-errChan:
				if err != nil && err != context.DeadlineExceeded {
					t.Logf("Log read error: %v", err)
				}
				break loop
			case <-timeout:
				t.Log("Follow mode timeout (expected)")
				break loop
			}
		}

		t.Logf("Read %d lines in follow mode", lineCount)
	})

	// 测试不存在的容器
	_, err = client.ContainerLogs(ctx, "nonexistent-container-id", LogOptions{})
	if err == nil {
		t.Error("Expected error for nonexistent container, got nil")
	}
}

// TestExecCommand_Integration 集成测试 ExecCommand 方法
func TestExecCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建客户端
	client, err := NewLocalClientFromEnv()
	if err != nil {
		t.Skipf("Cannot create Docker client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// 获取运行中的容器
	containers, err := client.ListContainers(ctx, false)
	if err != nil {
		t.Skipf("Cannot list containers: %v", err)
		return
	}

	if len(containers) == 0 {
		t.Skip("No running containers available for testing")
		return
	}

	// 测试在第一个容器中执行命令
	containerID := containers[0].ID
	t.Logf("Testing exec in container: %s (%s)", containers[0].Name, containerID[:12])

	// 测试 1: 执行简单的 echo 命令
	t.Run("SimpleEcho", func(t *testing.T) {
		config := ExecConfig{
			Cmd:          []string{"echo", "test"},
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
		}

		result, err := client.ExecCommand(ctx, containerID, config)
		if err != nil {
			t.Fatalf("ExecCommand failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil ExecResult")
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		t.Logf("Echo command completed with exit code: %d", result.ExitCode)
	})

	// 测试 2: 执行返回非零退出码的命令
	t.Run("NonZeroExit", func(t *testing.T) {
		config := ExecConfig{
			Cmd:          []string{"sh", "-c", "exit 42"},
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
		}

		result, err := client.ExecCommand(ctx, containerID, config)
		if err != nil {
			t.Fatalf("ExecCommand failed: %v", err)
		}

		if result.ExitCode != 42 {
			t.Errorf("Expected exit code 42, got %d", result.ExitCode)
		}

		if result.Error == "" {
			t.Error("Expected error message for non-zero exit code")
		}

		t.Logf("Non-zero exit command completed: exit code=%d, error=%s", result.ExitCode, result.Error)
	})

	// 测试 3: 执行不存在的命令
	t.Run("InvalidCommand", func(t *testing.T) {
		config := ExecConfig{
			Cmd:          []string{"nonexistent-command-xyz"},
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
		}

		result, err := client.ExecCommand(ctx, containerID, config)
		// 命令不存在可能导致执行失败或返回非零退出码
		if err == nil && result.ExitCode == 0 {
			t.Error("Expected error or non-zero exit code for invalid command")
		}

		t.Logf("Invalid command test: err=%v, exitCode=%d", err, result.ExitCode)
	})
}
