package transcriber

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/likan/mcp-vtt/internal/resolver"
)

const modelName = "ggml-tiny-q5_1.bin"

// ModelPath 返回 whisper 模型文件的完整路径。
func ModelPath() string {
	return filepath.Join(resolver.ModelDir(), modelName)
}

// TranscriptResult 转写结果。
type TranscriptResult struct {
	PlainText  string
	SRTContent string
}

// Transcribe 直接转写 whisper-cli 支持的音频文件，可选生成 SRT 时间戳。
func Transcribe(audioPath string, includeTimestamps bool) (*TranscriptResult, error) {
	outFile, err := os.CreateTemp("", "mcp-vtt-transcript-*")
	if err != nil {
		return nil, fmt.Errorf("create temp output: %w", err)
	}
	outBase := outFile.Name()
	outFile.Close()
	os.Remove(outBase)
	defer os.Remove(outBase)

	modelPath := ModelPath()
	args := []string{
		"-m", modelPath,
		"-f", audioPath,
		"-t", "8",
		"-l", "zh",
		"-otxt",
		"-of", outBase,
	}
	if includeTimestamps {
		args = append(args, "-osrt")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	if _, err := runCmd(ctx, resolver.Resolve("whisper-cli"), args...); err != nil {
		return nil, fmt.Errorf("whisper-cli: %w", err)
	}
	result := &TranscriptResult{}
	txtPath := outBase + ".txt"
	if data, err := os.ReadFile(txtPath); err == nil {
		result.PlainText = strings.TrimSpace(string(data))
	}
	os.Remove(txtPath)
	if includeTimestamps {
		srtPath := outBase + ".srt"
		if data, err := os.ReadFile(srtPath); err == nil {
			result.SRTContent = strings.TrimSpace(string(data))
		}
		os.Remove(srtPath)
	}
	return result, nil
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		tail := stderr.String()
		if len(tail) > 300 {
			tail = tail[len(tail)-300:]
		}
		return "", fmt.Errorf("%s: %w\n%s", name, err, tail)
	}
	return strings.TrimSpace(stdout.String()), nil
}
