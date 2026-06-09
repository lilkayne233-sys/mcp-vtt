# mcp-vtt

MCP Server — 视频/音频转文字。调用 yt-dlp 下载视频字幕或音频，使用 whisper.cpp tiny 模型本地转写。

## 快速开始

### macOS

```bash
# 1. 安装系统依赖
brew install ffmpeg whisper-cpp yt-dlp

# 2. 编译或下载二进制
git clone https://github.com/likan/mcp-vtt.git
cd mcp-vtt && make build

# 或直接下载预编译二进制
curl -L https://github.com/likan/mcp-vtt/releases/latest/download/mcp-vtt-darwin-arm64 \
  -o /usr/local/bin/mcp-vtt && chmod +x /usr/local/bin/mcp-vtt
```

### Windows

```powershell
# 1. 安装系统依赖
# ffmpeg: 从 https://www.gyan.dev/ffmpeg/builds/ 下载 ffmpeg-release-essentials.zip
#   解压后把 bin 目录加到 PATH

# yt-dlp: 从 https://github.com/yt-dlp/yt-dlp/releases 下载 yt-dlp.exe
#   放到 PATH 中的任意目录（如 C:\Tools\）

# whisper-cli: 从 https://github.com/ggml-org/whisper.cpp/releases 下载 main.zip
#   解压后把 whisper-cli.exe 放到 PATH 中的任意目录

# 2. 下载二进制
# 从 GitHub Releases 页面下载 mcp-vtt-windows-amd64.exe
#   重命名为 mcp-vtt.exe 放到 PATH 中
```

### Linux (Debian/Ubuntu)

```bash
# 1. 安装系统依赖
sudo apt install ffmpeg
pip install yt-dlp
# whisper-cpp: 从源码编译或下载预编译二进制 https://github.com/ggml-org/whisper.cpp

# 2. 编译或下载二进制
git clone https://github.com/likan/mcp-vtt.git
cd mcp-vtt && make build

# 或直接下载预编译二进制
curl -L https://github.com/likan/mcp-vtt/releases/latest/download/mcp-vtt-linux-amd64 \
  -o /usr/local/bin/mcp-vtt && chmod +x /usr/local/bin/mcp-vtt
```

### 3. 配置 MCP 客户端

添加到 Claude Desktop / Cursor / VS Code 的 MCP 配置：

如果已把二进制放到 PATH 中：

```json
{
  "mcpServers": {
    "mcp-vtt": {
      "command": "mcp-vtt"
    }
  }
}
```

macOS 用户写完整路径：

```json
{
  "mcpServers": {
    "mcp-vtt": {
      "command": "/usr/local/bin/mcp-vtt"
    }
  }
}
```

Windows 用户写完整路径：

```json
{
  "mcpServers": {
    "mcp-vtt": {
      "command": "C:\\Tools\\mcp-vtt.exe"
    }
  }
}
```

opencode 配置（含 10 分钟超时）：

```json
{
  "mcp": {
    "mcp-vtt": {
      "type": "local",
      "command": ["/path/to/mcp-vtt"],
      "timeout": 600000,
      "enabled": true
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

| 内容 | macOS/Linux | Windows | 说明 |
|---|---|---|---|
| whisper 模型 | `~/.mcp-vtt/models/ggml-tiny.bin` | `%USERPROFILE%\.mcp-vtt\models\ggml-tiny.bin` | 74MB，首次运行自动下载 |
| 下载的音频 | `~/.mcp-vtt/data/audio/` | `%USERPROFILE%\.mcp-vtt\data\audio\` | 临时文件，可手动清理 |
| 下载的字幕 | `~/.mcp-vtt/data/subs/` | `%USERPROFILE%\.mcp-vtt\data\subs\` | 临时文件 |
| 转录结果 | `~/.mcp-vtt/transcripts/{id}.md` | `%USERPROFILE%\.mcp-vtt\transcripts\{id}.md` | Markdown 格式永久保存 |
| 转录 SRT | `~/.mcp-vtt/transcripts/{id}.srt` | `%USERPROFILE%\.mcp-vtt\transcripts\{id}.srt` | 可选，带时间戳 |

可通过环境变量覆盖默认路径：

| 变量 | 默认值 |
|---|---|
| `DATA_DIR` | `~/.mcp-vtt/data` |
| `MODELS_DIR` | `~/.mcp-vtt/models` |
| `TRANSCRIPTS_DIR` | `~/.mcp-vtt/transcripts` |

## 架构

```
mcp-vtt/
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
