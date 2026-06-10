# mcp-vtt

MCP Server — 视频/音频转文字。下载解压即用，无需安装任何依赖。

## 快速开始

### macOS / Windows

从 [GitHub Releases](https://github.com/likan/mcp-vtt/releases) 下载对应平台的压缩包：

| 平台 | 文件 |
|---|---|
| macOS ARM64 | `mcp-vtt-darwin-arm64.tar.gz` |
| Windows AMD64 | `mcp-vtt-windows-amd64.zip` |

#### macOS

```bash
# 解压到任意目录（如 ~/Applications/）
tar -xzf mcp-vtt-darwin-arm64.tar.gz
cd ~/Applications/mcp-vtt
./mcp-vtt --version  # 验证安装成功
```

#### Windows

```powershell
# 解压到任意目录（如 C:\Users\you\Downloads\mcp-vtt）
Expand-Archive mcp-vtt-windows-amd64.zip -DestinationPath C:\Users\you\Downloads\mcp-vtt
cd C:\Users\you\Downloads\mcp-vtt
.\mcp-vtt.exe --version  # 验证安装成功
```

压缩包内含 ffmpeg、whisper-cli、yt-dlp 和 whisper 模型，无需安装任何系统依赖。

> macOS 的 yt-dlp 是 Python 脚本，如果系统没有 Python，可通过 `xcode-select --install` 或 `brew install python` 安装。

### 配置 MCP 客户端

#### Claude Desktop / Cursor / VS Code

指向 mcp-vtt 目录下的主程序（写你解压后的实际路径）：

如果已把二进制放到 PATH 中：

```json
{
  "mcpServers": {
    "mcp-vtt": {
      "command": "/path/to/mcp-vtt/mcp-vtt"
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
      "command": "C:\\Users\\you\\Downloads\\mcp-vtt\\mcp-vtt.exe"
    }
  }
}
```

#### OpenCode

在项目 `.opencode/opencode.jsonc` 或全局 `~/.config/opencode/opencode.jsonc` 中添加：

```jsonc
{
  "mcp": {
    "mcp-vtt": {
      "type": "local",
      "command": ["/path/to/mcp-vtt/mcp-vtt"],
      "timeout": 600000,
      "enabled": true
    }
  }
}
```

### 使用

配置好 MCP 客户端后，直接对 AI 说：

> 帮我转写这个视频 https://www.bilibili.com/video/BV1JwEu6BEAU

AI 会自动调用 `transcribe_video` 工具，返回转录文字。

## 文件存放位置

| 内容 | macOS | Windows | 说明 |
|---|---|---|---|
| whisper 模型 | 内嵌 `models/ggml-tiny-q5_1.bin` | 内嵌 `models\ggml-tiny-q5_1.bin` | 32MB 量化版，已内嵌 |
| 下载的音频 | `~/.mcp-vtt/data/audio/` | `%USERPROFILE%\.mcp-vtt\data\audio\` | 临时文件，可手动清理 |
| 转录结果 | `~/.mcp-vtt/transcripts/{id}.md` | `%USERPROFILE%\.mcp-vtt\transcripts\{id}.md` | Markdown 格式永久保存 |
| 转录 SRT | `~/.mcp-vtt/transcripts/{id}.srt` | `%USERPROFILE%\.mcp-vtt\transcripts\{id}.srt` | 可选，带时间戳 |

可通过环境变量覆盖默认路径：

| 变量 | 默认值 |
|---|---|
| `DATA_DIR` | `~/.mcp-vtt/data` |
| `MODELS_DIR` | exe 同目录 `models/` 或 `~/.mcp-vtt/models` |
| `TRANSCRIPTS_DIR` | `~/.mcp-vtt/transcripts` |

## 架构

```
mcp-vtt/
├── mcp-vtt              # 主程序（7.4MB）
├── bin/
│   ├── ffmpeg           # 音频处理
│   ├── ffprobe
│   ├── whisper-cli      # 语音转写
│   └── yt-dlp           # 视频下载
└── models/
    └── ggml-tiny-q5_1.bin  # whisper 量化模型（32MB）
```

510 行 Go 代码，单第三方依赖 `mcp-go`。发布包约 **118MB**（含所有依赖和模型）。

## 从源码构建发布包

```bash
# macOS arm64（需要 cmake，可通过 brew install cmake 安装）
make dist-darwin-arm64

# Windows amd64（交叉编译，不需要额外工具）
make dist-windows-amd64

# 同时构建两个平台
make dist
```

## MCP 工具

### `transcribe_video`

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `url` | string | 是 | 视频链接，已测试 Bilibili；其他 yt-dlp 支持的网站可尝试 |
| `timestamps` | boolean | 否 | 是否生成 SRT 时间戳，默认 true |

处理流程：下载音频并转为 mp3 → whisper.cpp 直接转写 mp3 → 保存 `.md` + `.srt`

## OpenCode Skill (视频总结)

本项目还附带了一个 **OpenCode Skill**，在转写的基础上自动生成视频大纲和总结。

### 安装

```bash
# 将 skill 目录拷贝到 OpenCode 的 skills 目录
cp -r <mcp-vtt>/skill ~/.config/opencode/skills/vtt
```

或手动创建 `~/.config/opencode/skills/vtt/SKILL.md`，内容见同目录下的 `skill/SKILL.md`。

### 流程

1. `yt-dlp --print duration <url>` 获取视频时长
2. 调用 `transcribe_video`（timestamps: false）获取转录文本
3. 按 `ceil(时长秒 / 360) × 100` 字的目标长度生成大纲和连贯总结
4. 忽略口播广告（除非广告就是主题）

### 使用

对 OpenCode 说：

> 总结这个视频 https://www.bilibili.com/video/XXXXX

AI 会自动调用 skill 流程，先转录、再生成大纲和总结。

## License

MIT
