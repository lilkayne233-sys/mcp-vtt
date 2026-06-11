---
name: vtt
description: 总结视频内容。用户发来视频链接要求总结/摘要时，先提取转录文本，再生成大纲和按比例总结。
---

## 流程

1. 调用 `mcp-vtt_transcribe_video`（timestamps: false）获取转录文本（会同时下载音频到 `~/.mcp-vtt/data/audio/<BV>.mp3`）
2. 用 `ffprobe` 读取本地音频文件获取时长：`ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 <音频路径>`
3. 根据时长计算目标字数：`Math.ceil(时长秒 / 360) × 100` 字（即每6分钟100字）
4. 生成**大纲**：按主题分章节，每章一小标题 + 一句话核心
5. 按大纲比例撰写**连贯总结**，不写成要点列表
6. 忽略口播广告（除非广告本身就是视频主题）

## 规则

- 保留因果逻辑链，不添加原文没有的信息
- 用中文输出，先大纲再总结
