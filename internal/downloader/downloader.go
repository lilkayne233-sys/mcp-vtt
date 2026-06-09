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
)

func defaultDir(envKey, fallback string) string {
	if d := os.Getenv(envKey); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".my-vtt", fallback)
}

// DataDir 默认 ~/.my-vtt/data，可通过 DATA_DIR 环境变量覆盖。
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
		return []string{"--add-header", "Referer:https://www.bilibili.com"}
	default:
		return nil
	}
}

// videoIDRegex 提取 BVxxx 或 11 位 YouTube ID。
var videoIDRegex = regexp.MustCompile(`(BV\w+|[\w-]{11})`)

// DownloadResult 下载结果。
type DownloadResult struct {
	FilePath string
	Title    string
	VideoID  string
}

// DownloadAudio 下载视频最佳音频，转 mp3 64K。返回文件路径、标题和视频 ID。
func DownloadAudio(url string) (*DownloadResult, error) {
	url, _ = ResolveShortUrl(url)

	// 获取标题
	titleArgs := append(ytBaseArgs(url), "--print", "%(title)s", "--no-playlist", url)
	title, err := runCmdTimeout("yt-dlp", titleArgs, 30*time.Second)
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
	out, err := runCmdTimeout("yt-dlp", dlArgs, 300*time.Second)
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

// DownloadSubtitle 下载平台已有字幕，清理时间戳后返回纯文本。无字幕返回空字符串。
func DownloadSubtitle(url string) (string, error) {
	url, _ = ResolveShortUrl(url)

	outDir := filepath.Join(DataDir, "subs")
	os.MkdirAll(outDir, 0o755)

	// 获取 video ID
	vidArgs := append(ytBaseArgs(url), "--print", "%(id)s", "--no-playlist", url)
	vid, err := runCmdTimeout("yt-dlp", vidArgs, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("get video id: %w", err)
	}

	subArgs := append(ytBaseArgs(url),
		"--write-auto-subs", "--write-subs",
		"--sub-lang", "zh-Hans,zh-CN,zh,en",
		"--sub-format", "srt/vtt/best",
		"--skip-download",
		"--output", filepath.Join(outDir, "%(id)s.%(ext)s"),
		"--no-playlist",
		url,
	)
	_, err = runCmdTimeout("yt-dlp", subArgs, 60*time.Second)
	if err != nil {
		return "", nil // no subtitles, not an error
	}

	for _, ext := range []string{"zh-Hans.srt", "zh-Hans.vtt"} {
		p := filepath.Join(outDir, vid+"."+ext)
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			continue
		}
		cleaned := cleanSubtitle(string(data))
		if len(cleaned) > 50 {
			os.Remove(p)
			return cleaned, nil
		}
	}
	return "", nil
}

// cleanSubtitle 去除 SRT/VTT 时间戳、序号和元数据行。
func cleanSubtitle(raw string) string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, trimmed)
			continue
		}
		if regexp.MustCompile(`^\d+$`).MatchString(trimmed) {
			continue
		}
		if strings.Contains(trimmed, "-->") {
			continue
		}
		switch trimmed {
		case "WEBVTT":
			continue
		}
		if strings.HasPrefix(strings.ToUpper(trimmed), "NOTE") ||
			strings.HasPrefix(strings.ToUpper(trimmed), "STYLE") ||
			strings.HasPrefix(strings.ToUpper(trimmed), "REGION") {
			continue
		}
		out = append(out, line)
	}
	result := strings.Join(out, "\n")
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}
