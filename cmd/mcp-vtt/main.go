package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/likan/mcp-vtt/internal/downloader"
	"github.com/likan/mcp-vtt/internal/transcriber"
)

const modelURL = "https://hf-mirror.com/ggerganov/whisper.cpp/resolve/main/ggml-tiny-q5_1.bin"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("mcp-vtt 0.1.0")
		return
	}

	s := server.NewMCPServer(
		"mcp-vtt",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	s.AddTool(mcp.NewTool("transcribe_video",
		mcp.WithDescription("下载视频音频并用 whisper.cpp tiny 模型转写为文字。Bilibili 已适配请求头；其他 yt-dlp 支持的网站可尝试使用。"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("视频链接，已测试 Bilibili，其他 yt-dlp 可处理的平台不做专门适配"),
		),
		mcp.WithBoolean("timestamps",
			mcp.Description("是否在输出中包含时间戳 (SRT 格式)"),
		),
	), transcribeVideoHandler)

	// 启动前确保模型文件存在
	if err := ensureModel(); err != nil {
		fmt.Fprintf(os.Stderr, "model setup failed: %v\n", err)
		os.Exit(1)
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func ensureModel() error {
	modelPath := transcriber.ModelPath()
	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}

	// silently download model, stderr would pollute MCP stdio
	os.MkdirAll(filepath.Dir(modelPath), 0o755)

	resp, err := http.Get(modelURL)
	if err != nil {
		return fmt.Errorf("download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download model: HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp(filepath.Dir(modelPath), ".model-*.tmp")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	cleanup := true
	defer func() {
		f.Close()
		if cleanup {
			os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("save model: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpName, modelPath); err != nil {
		return fmt.Errorf("install model: %w", err)
	}
	cleanup = false

	_, _ = os.Stat(modelPath)
	// model download complete
	return nil
}

func transcribeVideoHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, _ := req.RequireString("url")
	timestamps := true
	args := req.GetArguments()
	if b, ok := args["timestamps"].(bool); ok {
		timestamps = b
	}

	// 1. 下载音频
	dl, err := downloader.DownloadAudio(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("下载失败: %v", err)), nil
	}

	// 2. 转写
	result, err := transcriber.Transcribe(dl.FilePath, timestamps)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("转写失败: %v", err)), nil
	}

	// 3. 保存 transcripts
	transcriptsDir := os.Getenv("TRANSCRIPTS_DIR")
	if transcriptsDir == "" {
		home, _ := os.UserHomeDir()
		transcriptsDir = filepath.Join(home, ".mcp-vtt", "transcripts")
	}
	os.MkdirAll(transcriptsDir, 0o755)
	now := time.Now().Format(time.RFC3339)
	md := fmt.Sprintf("# %s\n> 来源: %s\n> 转写时间: %s\n\n%s",
		dl.Title, url, now, result.PlainText)
	mdPath := filepath.Join(transcriptsDir, dl.VideoID+".md")
	os.WriteFile(mdPath, []byte(md), 0o644)
	var info string
	if timestamps && result.SRTContent != "" {
		srtPath := filepath.Join(transcriptsDir, dl.VideoID+".srt")
		os.WriteFile(srtPath, []byte(result.SRTContent), 0o644)
		info = fmt.Sprintf("> 已保存到 %s/%s.md 和 %s.srt\n\n",
			transcriptsDir, dl.VideoID, dl.VideoID)
	} else {
		info = fmt.Sprintf("> 已保存到 %s/%s.md\n\n", transcriptsDir, dl.VideoID)
	}
	return mcp.NewToolResultText(info + result.PlainText), nil
}
