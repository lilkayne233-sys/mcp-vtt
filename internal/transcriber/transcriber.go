package transcriber

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// modelsDir 默认 ~/.mcp-vtt/models，可通过 MODELS_DIR 环境变量覆盖。
func modelsDir() string {
	if d := os.Getenv("MODELS_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mcp-vtt", "models")
}

const modelName = "ggml-tiny.bin"

// ModelPath 返回 whisper 模型文件的完整路径。
func ModelPath() string {
	return filepath.Join(modelsDir(), modelName)
}

// TranscriptResult 转写结果。
type TranscriptResult struct {
	PlainText  string
	SRTContent string
}

// Transcribe 转写音频文件，可选生成 SRT 时间戳。
func Transcribe(audioPath string, includeTimestamps bool) (*TranscriptResult, error) {
	wavPath, err := convertToWav(audioPath)
	if err != nil {
		return nil, fmt.Errorf("convert to wav: %w", err)
	}
	defer os.Remove(wavPath)
	modelPath := ModelPath()
	args := []string{
		"-m", modelPath,
		"-f", wavPath,
		"-l", "zh",
		"-otxt",
	}
	if includeTimestamps {
		args = append(args, "-osrt")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	if _, err := runCmd(ctx, "whisper-cli", args...); err != nil {
		return nil, fmt.Errorf("whisper-cli: %w", err)
	}
	result := &TranscriptResult{}
	// whisper-cli 输出文件名因编译方式不同，可能是 input.txt 或 input.wav.txt，两者都试
	txtPath := wavPath + ".txt"
	if _, err := os.Stat(txtPath); err != nil {
		txtPath = strings.TrimSuffix(wavPath, ".wav") + ".txt"
	}
	if data, err := os.ReadFile(txtPath); err == nil {
		result.PlainText = strings.TrimSpace(string(data))
	}
	os.Remove(txtPath)
	os.Remove(strings.TrimSuffix(wavPath, ".wav") + ".txt") // 清理另一个可能的文件
	if includeTimestamps {
		srtPath := wavPath + ".srt"
		if _, err := os.Stat(srtPath); err != nil {
			srtPath = strings.TrimSuffix(wavPath, ".wav") + ".srt"
		}
		if data, err := os.ReadFile(srtPath); err == nil {
			result.SRTContent = strings.TrimSpace(string(data))
		}
		os.Remove(srtPath)
		os.Remove(strings.TrimSuffix(wavPath, ".wav") + ".srt")
	}
	return result, nil
}
func convertToWav(input string) (string, error) {
	wavPath := strings.TrimSuffix(input, filepath.Ext(input)) + ".wav"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	_, err := runCmd(ctx, "ffmpeg",
		"-i", input,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-y",
		wavPath,
	)
	if err != nil {
		return "", err
	}
	return wavPath, nil
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
