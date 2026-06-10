package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/likan/mcp-vtt/internal/resolver"
)

func defaultDir(envKey, fallback string) string {
	if d := os.Getenv(envKey); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mcp-vtt", fallback)
}

// DataDir 默认 ~/.mcp-vtt/data，可通过 DATA_DIR 环境变量覆盖。
var DataDir = defaultDir("DATA_DIR", "data")

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

func runCmdTimeout(name string, args []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return runCmd(ctx, name, args...)
}

// ResolveShortUrl 解析 b23.tv 短链，非短链原样返回。
func ResolveShortUrl(u string) (string, error) {
	if !strings.Contains(u, "b23.tv") {
		return u, nil
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(u)
	if err != nil {
		return "", fmt.Errorf("resolve b23.tv: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("b23.tv redirect not found: %s", u)
	}
	return loc, nil
}

// ytBaseArgs 根据 URL 域名返回 yt-dlp 的通用 header 参数。
func ytBaseArgs(targetURL string) []string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}
	switch {
	case strings.Contains(u.Host, "bilibili.com"):
		// Bilibili requires Referer + Origin to avoid HTTP 412.
		return []string{
			"--add-header", "Referer:https://www.bilibili.com",
			"--add-header", "Origin:https://www.bilibili.com",
		}
	default:
		return nil
	}
}

// videoIDRegex 提取 BVxxx 或常见 11 位视频 ID。
var videoIDRegex = regexp.MustCompile(`(BV\w+|[\w-]{11})`)

// DownloadResult 下载结果。
type DownloadResult struct {
	FilePath string
	Title    string
	VideoID  string
}

// DownloadAudio 下载视频最佳音频，转 mp3 64K。返回文件路径、标题和视频 ID。
// Bilibili 已适配 Referer + Origin；其他 yt-dlp 支持的网站走通用参数。
func DownloadAudio(url string) (*DownloadResult, error) {
	url, _ = ResolveShortUrl(url)

	// 获取标题
	titleArgs := append(ytBaseArgs(url), "--print", "%(title)s", "--no-playlist", url)
	ytdlpPath := resolver.Resolve("yt-dlp")
	title, err := runCmdTimeout(ytdlpPath, titleArgs, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("get title: %w", err)
	}

	outDir := filepath.Join(DataDir, "audio")
	os.MkdirAll(outDir, 0o755)
	output := filepath.Join(outDir, "%(id)s.%(ext)s")

	dlArgs := append(ytBaseArgs(url),
		"-f", "bestaudio[ext=m4a]/bestaudio/best",
		"-x", "--audio-format", "mp3",
		"--audio-quality", "64K",
		"--output", output,
		"--no-playlist",
		"--print", "after_move:filepath",
		url,
	)
	out, err := runCmdTimeout(ytdlpPath, dlArgs, 300*time.Second)
	if err != nil {
		return nil, fmt.Errorf("download audio: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	fp := strings.TrimSpace(lines[len(lines)-1])
	vid := "unknown"
	m := videoIDRegex.FindStringSubmatch(fp)
	if len(m) >= 2 {
		vid = m[1]
	}
	if title == "" {
		title = "untitled"
	}

	return &DownloadResult{FilePath: fp, Title: title, VideoID: vid}, nil
}
