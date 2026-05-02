# yt-dlp 批量下载工具

基于 yt-dlp 原生配置的批量下载脚本，支持高清视频和字幕下载。

## 目录结构

```
toolkits/yt-dlp/
├── urls.txt        # 下载 URL 列表（每次只维护这个文件）
├── yt-dlp.conf     # 所有下载选项（一次配置永久生效）
├── download.bat    # Windows 双击运行
├── 2download.sh    # Linux/macOS/Git Bash 运行
└── README.md       # 本文档
```

下载文件保存到：`D:\yt-dlpDown\`

## 前置依赖

- [yt-dlp](https://github.com/yt-dlp/yt-dlp)：`choco install yt-dlp`
- [ffmpeg](https://ffmpeg.org/)：`choco install ffmpeg`（合并视频+音频必须）
- [Node.js 20+](https://nodejs.org/)：`choco install nodejs`（解决 YouTube n-challenge 必须）

## 快速使用

### 1. 编辑下载列表

打开 `urls.txt`，每行填写一个视频 URL：

```
# 注释行（忽略）
https://www.youtube.com/watch?v=7XQdp3hItX4
https://www.youtube.com/watch?v=xxxxx
```

### 2. 运行下载

**Windows**：双击 `download.bat` 即可，或命令行运行：
```bat
cd D:\softIns\Dropbox\vscode\toolkits\yt-dlp
download.bat
```

**Git Bash / WSL / Linux / macOS**：
```bash
cd /d/softIns/Dropbox/vscode/toolkits/yt-dlp
bash 2download.sh
```

## Cookies 说明

两个脚本均按以下优先级自动读取浏览器 Cookies，无需手动导出文件：

1. **Chrome**（优先）：Chrome 未运行时自动使用
2. **Firefox**（后备）：Chrome 正在运行（数据库被锁定）时自动切换
   > 建议在 Firefox 中登录一次 YouTube，避免因 Chrome 运行导致回退
3. **Edge**（兜底）：以上均不可用时使用

> 使用浏览器 Cookies 时，若提示数据库被锁定，请关闭对应浏览器后重试。

## 下载说明

| 功能 | 说明 |
|------|------|
| 画质 | 自动选择最高画质，合并为 mp4 |
| 音频 | 与视频流合并为单个 mp4 文件 |
| 字幕 | 优先中文（简/繁），其次英文；下载后自动转换为双语 ASS 格式 |
| 播放列表 | 默认只下载单个视频，不展开列表 |
| 重试 | 网络失败自动重试 5 次 |

## 常见问题

**Q: 提示 `n challenge solving failed`**  
A: 未安装 Node.js，运行 `choco install nodejs` 后重试。

**Q: 视频下载完是 .webm 格式**  
A: 未安装 ffmpeg，运行 `choco install ffmpeg` 后重试。

**Q: 字幕文件没有下载**  
A: 该视频没有提供字幕，属于正常情况，不影响视频下载。

**Q: 提示 cookies 数据库被锁定**  
A: 关闭 Chrome/Edge 浏览器后重新运行；或在 Firefox 中登录 YouTube 后再试。
