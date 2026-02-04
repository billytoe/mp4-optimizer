# 🚀 发布与部署指南 (Release Guide)

本文档将详细说明如何编译、打包并在 GitHub 上发布 **MP4 FastStart Inspector** 的新版本。

## 1. 发布前准备

在开始构建之前，请确保当前代码已准备就绪：

1.  **代码测试**：确保 `wails dev` 下所有功能正常。
2.  **更新依赖**：运行 `go mod tidy` 和 `pnpm install` 确保依赖最新。
3.  **确定版本号**：决定本次发布的版本号（例如 `v1.0.1`）。建议遵循 [Semantic Versioning](https://semver.org/) (语义化版本控制)。

---

## 2. 编译构建 (Build)

我们需要通过命令行构建 Windows 和 macOS 的可执行文件。构建时需**注入版本号**，这对自动更新功能至关重要。

### Windows 版本编译

在项目根目录下运行以下命令（支持在 macOS 上交叉编译）：

```bash
# 请将 v1.0.1 替换为你实际的版本号
wails build -platform windows/amd64 -ldflags "-X main.Version=v1.0.1"
```

*   **产出文件**：`build/bin/MP4 FastStart Inspector.exe`
*   **重命名建议**：为了清晰，建议将文件重命名为 `FastStartInspector_v1.0.1.exe`。

### macOS 版本编译

仅支持在 macOS 系统上执行：

```bash
# 请将 v1.0.1 替换为你实际的版本号
wails build -platform darwin/universal -ldflags "-X main.Version=v1.0.1"
```

*   **产出文件**：`build/bin/MP4 FastStart Inspector.app`
*   **注意**：GitHub Releases 不支持直接上传文件夹（`.app` 本质是文件夹），因此必须先将其**压缩为 ZIP**。

**压缩命令：**
```bash
cd build/bin
zip -r "FastStartInspector_v1.0.1_mac.zip" "MP4 FastStart Inspector.app"
```

---

## 3. 在 GitHub 上创建 Release

准备好构建产物后，我们可以发布 release。

1.  **访问 GitHub Releases 页面**：
    *   打开你的 GitHub 仓库主页。
    *   点击右侧边栏的 **"Releases"** 部分，或者点击 **"Create a new release"**。

2.  **打标签 (Tag version)**：
    *   点击 **"Choose a tag"** 下拉菜单。
    *   输入新的版本号（例如 `v1.0.1`）。
    *   点击 **"Create new tag: v1.0.1" on publish**。

3.  **填写标题和说明**：
    *   **Release title**: 填写版本号或简短描述，例如 `v1.0.1 - 修复拖拽问题`。
    *   **Describe this release**: 详细列出变更日志 (Changelog)。
    
    *模板示例：*
    ```markdown
    ## ✨ 新特性
    *   新增了深色模式支持。
    *   优化了 MP4 分析速度。

    ## 🐛 修复
    *   修复了 Windows 下无法拖拽文件夹的问题。
    ```

4.  **上传附件 (Assets)**：
    *   将之前准备好的文件拖入底部的 "Attach binaries by dropping them here..." 区域：
        1.  Windows `exe` 文件 (例如 `FastStartInspector_v1.0.1.exe`)
        2.  macOS `zip` 压缩包 (例如 `FastStartInspector_v1.0.1_mac.zip`)

5.  **发布**：
    *   确认无误后，点击绿色按钮 **"Publish release"**。

---

## 4. (可选) 配置 GitHub 仓库信息

为了让项目看起来更专业，建议更新仓库顶部的 **About** 信息。

*   **Description (简介)**: 
    > A cross-platform desktop tool to optimize MP4 files for fast network streaming (FastStart). Instantly detects and moves the 'moov' atom. Built with Wails (Golang) + React.
    
    *(中文版可选：一款基于 Wails + React 开发的跨平台 MP4 视频 FastStart 优化工具，支持秒开检测与一键修复。)*

*   **Website (官网)**: 
    *   如果你有官网可以填，没有的话可以留空或填仓库地址。

*   **Topics (标签)**:
    添加以下标签有助于被搜索到：
    `mp4`, `video-optimization`, `wails`, `golang`, `react`, `desktop-app`, `streaming`, `faststart`, `ffmpeg`

---

## 5. 自动更新注意事项 (Auto-Update)

如果你启用了应用的自动更新功能，发布 Release 后还需要更新你的 `latest.json` 静态文件。

1.  **发布 Release**：按上述步骤在 GitHub 发布新版本，并上传 `.exe` (Windows) 和 `.zip` (macOS)。
2.  **获取下载链接**：在 Release 页面右键点击附件 -> 复制链接地址。
3.  **更新 `latest.json`**：
    *   编辑项目根目录下的 `latest.json` 文件。
    *   填入新版本号、发布说明和下载链接。
    *   **提交并推送 (Push)** 代码到 GitHub。
    
    *`latest.json` 示例：*
    ```json
    {
      "version": "v1.0.1",
      "download_url_windows": "https://github.com/billytoe/mp4-optimizer/releases/download/v1.0.1/FastStartInspector_v1.0.1.exe",
      "download_url_mac": "https://github.com/billytoe/mp4-optimizer/releases/download/v1.0.1/FastStartInspector_v1.0.1_mac.zip",
      "release_notes": "1. 修复了拖拽问题\n2. 优化了性能"
    }
    ```
    
    *注意：App 默认配置为读取 GitHub Raw 地址 (`raw.githubusercontent.com/.../latest.json`)。因此，更新此文件并推送后，用户端即可检测到升级。*
