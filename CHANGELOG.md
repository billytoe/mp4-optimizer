# Changelog

本文件记录了项目的所有重要变更。
格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且本项目遵循 [Semantic Versioning](https://semver.org/spec/v2.0.0.html)。

## [Unreleased]

## [0.0.3] - 2026-02-07

### 🐛 修复 (Fixed)
- **macOS 自动更新**: 修复了严重 Bug，之前的更新逻辑会将 `.zip` 文件直接覆盖可执行文件导致应用损坏。现在正确实现了：
  - 下载 `.zip` 到临时文件
  - 解压 `.zip` 到临时目录
  - 定位并替换整个 `.app` Bundle
  - 替换失败时自动回滚到旧版本

## [0.0.2] - 2026-02-07

### ✨ 新增 (Added)
- **Build System**: 构建脚本 (`scripts/build_release.sh`) 现支持同时生成 Release 和 Debug 双版本。
  - Release 版 (`FastStartInspector_vX.X.X.exe`): 无控制台窗口，适合最终用户。
  - Debug 版 (`FastStartInspector_Debug_vX.X.X.exe`): 保留控制台窗口，便于查看运行日志。
- **UI**: 界面拖拽区域提示文本更新，明确支持 "MP4 文件或文件夹"。
- **DevOps**: 新增 AI Skills 自动安装脚本 (`scripts/install_skills.sh`)。

### 🐛 修复 (Fixed)
- **Windows 拖拽 (Drag & Drop)**: 
  - 彻底修复了 Windows 平台下文件夹拖拽失效的问题。
  - 采用了原生 `drop` 事件处理机制，绕过了 WebView2 对文件夹拖拽的限制。
  - 增强了文件过滤逻辑，防止有效文件夹被误判。
- **元数据解析 (Metadata)**: 
  - 修复了部分未优化的 MP4 文件（`moov` 原子位于文件末尾，例如 OBS 录制文件）无法读取分辨率和编码信息的问题。
  - 重构了 `GetMetadata` 逻辑，引入健壮的 `pkg/atomic` 库进行原子定位。
- **版本号规范**: 修复了构建脚本中版本号处理不一致的问题。现在统一遵循 SemVer 规范（内部版本号纯数字，文件名带 `v` 前缀）。

### ⚡ 优化 (Changed)
- **Core**: 验证并确保了 MP4 优化逻辑的安全性。采用原位替换 (`Replace`) + `.bak` 备份机制，若优化失败会自动恢复原文件。

## [0.0.1] - 2026-02-01

### 🎉 初始发布 (Initial Release)
- 基础功能：MP4 FastStart 结构检测与优化。
- 支持单个文件和批量文件拖拽。
- 跨平台支持 (Windows / macOS)。
- 简单的状态列表与进度显示。
