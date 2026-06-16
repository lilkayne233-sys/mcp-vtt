---
name: vtt
description: 总结视频内容。用户发来视频链接要求总结/摘要时，先判断链接类型（单视频/主页），再执行对应流程。
---

## 流程

### 1. 判断链接类型

收到链接后先分类：

| 类型 | 匹配规则 | 示例 |
|------|----------|------|
| **单视频** | 包含 `/video/` 或 `watch?v=` 等单视频标识 | `bilibili.com/video/BV1xx...`<br>`youtube.com/watch?v=...` |
| **UP主主页** | `space.bilibili.com/<数字>` 或 `space.bilibili.com/<数字>/video` | `space.bilibili.com/25876945` |
| **合集/列表** | 包含 `series`、`list`、`playlist`、`collection` 等 | `bilibili.com/medialist/...` |

- 若是**单视频** → 走"单视频流程"
- 若是**UP主主页** → 走"批量采集流程"
- 若是**合集/列表** → 走"批量采集流程"（但用列表 API 替代空间 API）
- 若无法判断 → 先尝试当单视频处理

---

### 2. 单视频流程

1. 调用 `mcp-vtt_transcribe_video`（timestamps: false）获取转录文本（会同时下载音频到 `~/.mcp-vtt/data/audio/<BV>.wav`）
2. 用 `ffprobe` 读取本地音频文件获取时长：`ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 <音频路径>`
3. 根据时长计算目标字数：大纲 `Math.ceil(时长秒 / 300) × 200` 字（每5分钟200字）+ 总结 `Math.ceil(时长秒 / 300) × 50` 字（每5分钟50字）
4. 生成**大纲**：按主题分章节，两级目录

   **大纲格式要求：**
   - 一级标题：`- **标题**`，概括该章节核心主题
   - 二级标题：缩进 `  - `，展开具体要点、关键数据、逻辑链条
   - 按视频内容自然分节，每节两级，覆盖核心事实
   ```
   - **一级标题**
     - 二级要点：关键数据/逻辑
     - 二级要点：关键数据/逻辑
   ```

5. 撰写**总结**：和大纲不同，不重复章节标题

   **总结格式要求：**
   - 也用两级目录，正文是连贯段落
   - 一级分类只有两个：`- **核心内容**`、`- **关键细节**`
   - 其下 `  - ` 接连贯正文，提炼视频的核心论点、反直觉结论、深层意义
   - 回答"看完这个视频最该带走什么"

   ```
   - **核心内容**
     - 正文段落：视频最核心的论点、反直觉结论…
   - **关键细节**
     - 正文段落：支撑该论点的关键数据、设计选择、行业信号…
   ```

---

### 3. 批量采集流程（UP主主页 / 合集）

#### 3.1 获取基本信息

- **UP主主页**：从 URL 提取 mid（`space.bilibili.com/<mid>`），询问用户要拉取多少个视频
- **合集/列表**：从 URL 提取 medialist/series ID，自动获取该列表下所有视频

#### 3.2 获取匿名 cookie（避免风控）

```
curl -s -c "%TEMP%\bili_cookies.txt" -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" "https://www.bilibili.com/" >nul
```

#### 3.3 拉取视频列表

- **UP主主页**：
  ```
  curl -s -b "%TEMP%\bili_cookies.txt" -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" "https://api.bilibili.com/x/space/arc/search?mid=<mid>&ps=<数量>&order=pubdate"
  ```
  - 翻页：`pn=2`、`pn=3` 类推，每页最多 `ps=30`
  - 每条视频有 `bvid`、`title`、`created`（发布时间戳）、`length`（时长）、`is_pay`、`is_charging_arc` 等字段

- **合集/列表**：使用对应的 Bilibili 列表 API 拉取视频列表

#### 3.4 过滤

- 充电视频：`is_pay != 0` 或 `is_charging_arc == true` → 跳过，标注 `[已跳过：充电视频]`
- 联合投稿（可选）：`is_union_video == 1` → 可询问用户是否排除

#### 3.5 逐个处理（按发布时间倒序）

每个视频执行以下步骤：

1. 调用 `mcp-vtt_transcribe_video`（timestamps: false）获取转录文本
2. 用 `ffprobe` 读取音频时长：
   ```
   ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "C:\Users\LIKAN\.mcp-vtt\data\audio\<BV>.wav"
   ```
3. **兜底检查**：如果音频时长 < 15 秒，判定为试听片段（疑似充电视频漏检），跳过
4. 计算目标字数：大纲 `Math.ceil(时长秒 / 300) × 200` 字，总结 `Math.ceil(时长秒 / 300) × 50` 字
5. 生成大纲（两级目录）和总结（核心内容 + 关键细节）
6. 每个视频之间用 `---` 分隔

---

## 通用规则

- 保留因果逻辑链，不添加原文没有的信息
- 用中文输出，先大纲再总结
- **直接以文本输出到对话框，不写入文件**
- 忽略口播广告（除非广告本身就是视频主题）
- 若整期视频为产品推广/商业合作，须在大纲和总结开头标注 `[⚠️ 商业合作/广告]`
- 直播类视频中，主播口播/回应的简短弹幕互动内容视为垃圾内容忽略（除非弹幕本身构成了视频的核心主题）
- 批量采集时 API 请求间隔建议 ≥ 3 秒，避免触发风控
