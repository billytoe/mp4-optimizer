# 🚀 发布与部署指南 (Release Guide)

本文档详细说明如何构建、打包并在 GitHub 上发布 **MP4 FastStart Inspector** 的新版本。

本项目已实现**全自动化**的一键构建和发布流程。

---

## 1. 发布前准备

在开始发布之前，请确保：

1.  **安装必要工具**：
    *   `gh` (GitHub CLI): 用于自动上传 Release。
        ```bash
        brew install gh
        gh auth login
        ```
2.  **更新依赖**：运行 `go mod tidy` 和 `pnpm install`。
3.  **确定版本号**：例如 `v1.0.6`。

---

## 2. 更新 Changelog (必须)

自动发布脚本会从 `CHANGELOG.md` 读取发布说明。如果未找到对应版本的说明，脚本会中止发布。

请在 `CHANGELOG.md` 顶部的 `[Unreleased]` 下方添加新版本记录：

```markdown
## [1.0.6] - 2026-02-08

### ✨ 新增
- ...

### 🐛 修复
- ...
```

---

## 3. 一键构建与发布 (推荐)

我们提供了一个全自动脚本 `scripts/release_github.sh`，它会：
1.  检查 `CHANGELOG.md` 是否包含该版本的更新日志。
2.  自动构建 Windows (Release/Debug) 和 macOS (Universal) 版本。
3.  自动打包为 Zip 文件。
4.  自动创建 GitHub Release 并上传所有附件。
5.  生成 `latest.json` 更新片段。

### 运行发布脚本

```bash
# 请替换为你实际的版本号
./scripts/release_github.sh v1.0.6
```

脚本执行成功后，你将看到 GitHub Release 的链接。

---

## 4. 启用自动更新 (Auto-Update)

发布成功后，脚本会在终端输出 `latest.json` 的更新内容。

1.  **复制** 终端输出的 JSON 内容。
2.  **覆盖** 项目根目录下的 `latest.json` 文件。
3.  **提交并推送**：
    ```bash
    git add latest.json
    git commit -m "chore: update latest.json to v1.0.6"
    git push origin main
    ```

用户重启应用后即可检测到新版本并自动更新。

---

## 5. 手动构建 (仅构建不发布)

如果你只想构建产物而不发布到 GitHub，可以使用：

```bash
./scripts/build_release.sh v1.0.6
```

产物将生成在 `dist/v1.0.6/` 目录下。

---

## 6. 自动更新原理

- **Windows**: 下载 zip -> 解压 -> 备份旧 exe -> 替换新 exe -> 自动重启。
- **macOS**: 下载 zip -> 解压 -> 定位 `.app` -> 替换整个 Bundle -> 自动重启。
