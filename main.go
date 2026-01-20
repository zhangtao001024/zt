package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const deepSeekAPIURL = "https://api.deepseek.com/v1/chat/completions"

var (
	model           = getEnv("DEEPSEEK_MODEL", "deepseek-chat")
	apiKey          = os.Getenv("DEEPSEEK_API_KEY")
	promptPath      = getEnv("DSHELL_SYSTEM_PROMPT", defaultPromptPath())
	blockedKeywords = []string{"rm -rf", "mkfs", "dd ", "shutdown", "reboot"}
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type requestPayload struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func defaultPromptPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "system_prompt.txt")
	}
	return filepath.Join(filepath.Dir(execPath), "system_prompt.txt")
}

func loadSystemPrompt() (string, error) {
	data, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("read system prompt: %w", err)
	}
	return string(data), nil
}

func streamDeepSeek(messages []message, streamOutput bool) (string, error) {
	payload := requestPayload{
		Model:       model,
		Messages:    messages,
		Temperature: 0.1,
		MaxTokens:   512,
		Stream:      true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, deepSeekAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed with status %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	var fullText strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk streamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		if streamOutput {
			fmt.Print(delta)
		}
		fullText.WriteString(delta)
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream read error: %w", err)
	}
	if streamOutput {
		fmt.Println()
	}
	return strings.TrimSpace(fullText.String()), nil
}

func isDangerous(command string) bool {
	for _, keyword := range blockedKeywords {
		if strings.Contains(command, keyword) {
			return true
		}
	}
	return false
}

func streamExecute(command string) (int, error) {
	cmd := exec.Command("bash", "-c", command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 1, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start command: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stdout, stdout)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stderr, stderr)
	}()

	wg.Wait()
	err = cmd.Wait()
	if err == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 1, fmt.Errorf("command execution failed: %w", err)
}

func main() {
	if apiKey == "" {
		fmt.Println("❌ DEEPSEEK_API_KEY 未设置")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("用法: dshell \"自然语言指令\"")
		os.Exit(1)
	}

	prompt, err := loadSystemPrompt()
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}

	userInput := strings.Join(os.Args[1:], " ")

	fmt.Println("\n▶ 正在生成命令…\n")
	command, err := streamDeepSeek([]message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: userInput},
	}, true)
	if err != nil {
		fmt.Printf("❌ 命令生成失败: %v\n", err)
		os.Exit(1)
	}
	if command == "" {
		fmt.Println("❌ 命令生成为空")
		os.Exit(1)
	}

	fmt.Println("\n▶ 生成完成\n")

	if isDangerous(command) {
		fmt.Println("❌ 命令包含高危关键词，已阻止执行")
		os.Exit(2)
	}

	fmt.Println("▶ 正在执行命令…\n")
	code, err := streamExecute(command)
	if err != nil {
		fmt.Printf("❌ 命令执行失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n▶ 执行结束，退出码: %d\n\n", code)

	fmt.Println("▶ 正在分析执行结果…\n")
	analysisPrompt := fmt.Sprintf(`以下是 Linux 命令的真实执行情况，请进行运维分析。

命令：
%s

退出码：
%d

请输出：
- 关键发现
- 是否异常
- 可能原因
- 运维建议
`, command, code)

	_, err = streamDeepSeek([]message{
		{Role: "system", Content: "你是一个资深 Linux 运维工程师，只分析真实执行结果。"},
		{Role: "user", Content: analysisPrompt},
	}, true)
	if err != nil {
		fmt.Printf("❌ 分析失败: %v\n", err)
		os.Exit(1)
	}
}
