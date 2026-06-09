# my-vtt

MCP Server — 视频/音频转文字。调用 yt-dlp 下载视频字幕或音频，使用 whisper.cpp tiny 模型本地转写。

## 快速开始

### 1. 安装系统依赖

```bash
# macOS
brew install ffmpeg whisper-cpp yt-dlp

# Debian/Ubuntu
sudo apt install ffmpeg
pip install yt-dlp
# whisper-cpp: 从源码编译或下载预编译二进制 https://github.com/ggml-org/whisper.cpp
```

### 2. 编译或下载

```bash
# 编译
git clone https://github.com/likan/my-vtt.git
cd my-vtt
make build

# 或直接下载预编译二进制
curl -L https://github.com/likan/my-vtt/releases/latest/download/my-vtt-darwin-arm64 \
  -o /usr/local/bin/my-vtt && chmod +x /usr/local/bin/my-vtt
```

### 3. 配置 MCP 客户端

添加到 Claude Desktop / Cursor / VS Code 的 MCP 配置：

```json
{
  "mcpServers": {
    "my-vtt": {
      "command": "/usr/local/bin/my-vtt"
    }
  }
}
```

首次运行时，**自动下载** whisper tiny 模型（~74MB），无需手动操作。

### 4. 使用

配置好 MCP 客户端后，直接对 AI 说：

> 帮我转写这个视频 https://www.bilibili.com/video/BV1JwEu6BEAU

AI 会自动调用 `transcribe_video` 工具，返回转录文字。

## 文件存放位置

| 内容 | 路径 | 说明 |
|---|---|---|
| whisper 模型 | `~/.my-vtt/models/ggml-tiny.bin` | 74MB，首次运行自动下载 |
| 下载的音频 | `~/.my-vtt/data/audio/` | 临时文件，可手动清理 |
| 下载的字幕 | `~/.my-vtt/data/subs/` | 临时文件 |
| 转录结果 | `~/.my-vtt/transcripts/{videoId}.md` | Markdown 格式永久保存 |
| 转录 SRT | `~/.my-vtt/transcripts/{videoId}.srt` | 可选，带时间戳 |

可通过环境变量覆盖默认路径：

| 变量 | 默认值 |
|---|---|
| `DATA_DIR` | `~/.my-vtt/data` |
| `MODELS_DIR` | `~/.my-vtt/models` |
| `TRANSCRIPTS_DIR` | `~/.my-vtt/transcripts` |

## 架构

```
my-vtt/
├── cmd/my-vtt/main.go            # MCP 服务入口
├── internal/
│   ├── downloader/downloader.go  # yt-dlp 封装
│   └── transcriber/transcriber.go # ffmpeg + whisper-cli 封装
├── go.mod / go.sum
├── Makefile
└── README.md
```

490 行 Go 代码，单第三方依赖 `mcp-go`。编译后 **7.4MB** 静态二进制。

## MCP 工具

### `transcribe_video`

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `url` | string | 是 | 视频链接（Bilibili、YouTube 等 yt-dlp 支持的所有平台） |
| `timestamps` | boolean | 否 | 是否生成 SRT 时间戳，默认 true |

处理流程：优先取已有字幕 → 无字幕时下载音频 → whisper.cpp 转写 → 保存 `.md` + `.srt`

## License

MIT
