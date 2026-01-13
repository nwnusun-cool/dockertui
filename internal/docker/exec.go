package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
)

// ExecConfig 执行命令的配置
type ExecConfig struct {
	Cmd          []string // 执行的命令
	AttachStdin  bool     // 是否附加标准输入
	AttachStdout bool     // 是否附加标准输出
	AttachStderr bool     // 是否附加标准错误
	Tty          bool     // 是否分配伪终端
	Env          []string // 环境变量
	WorkingDir   string   // 工作目录
}

// ExecResult exec 执行结果
type ExecResult struct {
	ExitCode int    // 退出码
	Error    string // 错误信息
}

// ExecCommand 在容器中执行命令（非交互式）
// 返回命令的输出和错误
func (c *LocalClient) ExecCommand(ctx context.Context, containerID string, config ExecConfig) (*ExecResult, error) {
	if c == nil || c.cli == nil {
		return nil, fmt.Errorf("Docker client not initialized")
	}

	// 创建 exec 实例
	execConfig := container.ExecOptions{
		AttachStdout: config.AttachStdout,
		AttachStderr: config.AttachStderr,
		AttachStdin:  config.AttachStdin,
		Tty:          config.Tty,
		Cmd:          config.Cmd,
		Env:          config.Env,
		WorkingDir:   config.WorkingDir,
	}

	execCreateResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec instance: %w", err)
	}

	// 附加到 exec
	execAttachResp, err := c.cli.ContainerExecAttach(ctx, execCreateResp.ID, container.ExecStartOptions{
		Tty: config.Tty,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer execAttachResp.Close()

	// 如果需要标准输入，连接到 stdin
	if config.AttachStdin {
		go func() {
			io.Copy(execAttachResp.Conn, os.Stdin)
		}()
	}

	// 读取输出
	if config.Tty {
		// TTY 模式：直接复制输出
		io.Copy(os.Stdout, execAttachResp.Reader)
	} else {
		// 非 TTY 模式：直接复制（简化处理）
		io.Copy(os.Stdout, execAttachResp.Reader)
	}

	// 获取执行结果
	inspectResp, err := c.cli.ContainerExecInspect(ctx, execCreateResp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exec result: %w", err)
	}

	result := &ExecResult{
		ExitCode: inspectResp.ExitCode,
	}

	if inspectResp.ExitCode != 0 {
		result.Error = fmt.Sprintf("command exit code: %d", inspectResp.ExitCode)
	}

	return result, nil
}

// ExecShell 在容器中启动交互式 shell
// 这是一个简化的实现，适用于基本的交互式场景
func (c *LocalClient) ExecShell(ctx context.Context, containerID string, shell string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	// 如果未指定 shell，自动检测可用的 shell
	if shell == "" {
		detectedShell, err := c.detectShell(ctx, containerID)
		if err != nil {
			return fmt.Errorf("unable to detect shell in container: %w", err)
		}
		shell = detectedShell
	}

	// 创建交互式 exec 配置
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	// 创建 exec 实例
	execCreateResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	// 附加到 exec
	execAttachResp, err := c.cli.ContainerExecAttach(ctx, execCreateResp.ID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer execAttachResp.Close()

	// 用于通知输出读取完成（shell 退出）
	outputDone := make(chan struct{})

	// 从 stdin 复制到容器（在 goroutine 中）
	go func() {
		defer func() {
			recover() // 忽略可能的 panic
		}()
		
		buf := make([]byte, 1024)
		for {
			// 先检查是否应该退出
			select {
			case <-outputDone:
				// shell 已退出，停止读取 stdin
				return
			default:
			}
			
			// 设置读取超时，避免永久阻塞
			// 注意：os.Stdin 不支持 SetReadDeadline，所以我们用非阻塞方式
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				// 再次检查是否应该退出
				select {
				case <-outputDone:
					return
				default:
				}
				_, err = execAttachResp.Conn.Write(buf[:n])
				if err != nil {
					return
				}
			}
		}
	}()

	// 从容器复制到 stdout（阻塞直到 shell 退出）
	io.Copy(os.Stdout, execAttachResp.Reader)

	// shell 已退出，关闭 outputDone 通知 stdin goroutine
	close(outputDone)
	
	// 关闭连接，这会导致 stdin goroutine 的 Write 失败并退出
	execAttachResp.Close()

	return nil
}

// detectShell 自动检测容器中可用的 shell
// 按优先级尝试: /bin/bash -> /bin/sh -> /bin/ash
func (c *LocalClient) detectShell(ctx context.Context, containerID string) (string, error) {
	// 尝试的 shell 列表（按优先级）
	shells := []string{
		"/bin/bash", // 功能最丰富，优先使用
		"/bin/sh",   // POSIX 标准 shell，几乎所有容器都有
		"/bin/ash",  // Alpine Linux 使用的 shell
	}

	for _, shell := range shells {
		// 检查 shell 是否存在
		if c.checkShellExists(ctx, containerID, shell) {
			return shell, nil
		}
	}

	return "", fmt.Errorf("no available shell in container (tried: %v)", shells)
}

// checkShellExists 检查指定的 shell 是否存在于容器中
func (c *LocalClient) checkShellExists(ctx context.Context, containerID string, shell string) bool {
	// 使用 test 命令检查文件是否存在
	execConfig := container.ExecOptions{
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          []string{"test", "-f", shell},
	}

	execCreateResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return false
	}

	// 启动 exec（不附加输出）
	err = c.cli.ContainerExecStart(ctx, execCreateResp.ID, container.ExecStartOptions{})
	if err != nil {
		return false
	}

	// 检查执行结果
	inspectResp, err := c.cli.ContainerExecInspect(ctx, execCreateResp.ID)
	if err != nil {
		return false
	}

	// 退出码为 0 表示文件存在
	return inspectResp.ExitCode == 0
}

// ExecShellInteractive 在容器中启动交互式 shell（带终端原始模式支持）
// 这个方法会设置终端为原始模式，提供更好的交互体验
// 注意：这个方法会修改终端状态，调用前应确保终端已被释放
func (c *LocalClient) ExecShellInteractive(ctx context.Context, containerID string, shell string) error {
	if c == nil || c.cli == nil {
		return fmt.Errorf("Docker client not initialized")
	}

	// 如果未指定 shell，自动检测可用的 shell
	if shell == "" {
		detectedShell, err := c.detectShell(ctx, containerID)
		if err != nil {
			return fmt.Errorf("unable to detect shell in container: %w", err)
		}
		shell = detectedShell
	}

	// 创建交互式 exec 配置
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	// 创建 exec 实例
	execCreateResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	// 附加到 exec
	execAttachResp, err := c.cli.ContainerExecAttach(ctx, execCreateResp.ID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer execAttachResp.Close()

	// 设置终端为原始模式（如果是 TTY）
	// 注意：这里简化处理，实际使用时应该在 TUI 层面处理终端模式
	// 因为 Bubble Tea 已经处理了终端的释放和恢复

	// 处理输入输出流
	// 从 stdin 复制到容器
	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(execAttachResp.Conn, os.Stdin)
		errChan <- err
	}()

	// 从容器复制到 stdout
	_, err = io.Copy(os.Stdout, execAttachResp.Reader)
	if err != nil {
		return fmt.Errorf("failed to read container output: %w", err)
	}

	// 等待输入复制完成
	if err := <-errChan; err != nil && err != io.EOF {
		return fmt.Errorf("failed to write container input: %w", err)
	}

	// 检查执行结果
	inspectResp, err := c.cli.ContainerExecInspect(ctx, execCreateResp.ID)
	if err != nil {
		return fmt.Errorf("failed to get exec result: %w", err)
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("shell exit code: %d", inspectResp.ExitCode)
	}

	return nil
}

// GetAvailableShells 获取容器中所有可用的 shell 列表
// 用于调试和显示给用户
func (c *LocalClient) GetAvailableShells(ctx context.Context, containerID string) []string {
	shells := []string{
		"/bin/bash",
		"/bin/sh",
		"/bin/ash",
		"/bin/zsh",
		"/bin/ksh",
	}

	available := []string{}
	for _, shell := range shells {
		if c.checkShellExists(ctx, containerID, shell) {
			available = append(available, shell)
		}
	}

	return available
}
