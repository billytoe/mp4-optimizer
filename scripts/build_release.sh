#!/bin/bash

# 检查是否提供了版本号参数
if [ -z "$1" ]; then
  echo "请提供版本号，例如: ./scripts/build_release.sh v1.0.1"
  exit 1
fi

VERSION=$1
BUILD_DIR="build/bin"
DIST_DIR="dist/$VERSION"

echo "🚀 开始构建版本: $VERSION"
echo "-----------------------------------"

# 清理旧的构建和分发目录
echo "🧹 清理旧文件..."
rm -rf "$BUILD_DIR"
mkdir -p "$DIST_DIR"

# 1. 编译 Windows 版本
# Build options matching user requirement for console debug
echo "🪟 正在编译 Windows 版本 (Console Debug Mode)..."
# Removed -H windowsgui to show console
wails build -platform windows/amd64 -clean -o "FastStartInspector_Debug_${VERSION}.exe" -ldflags "-X main.Version=${VERSION}"

if [ $? -eq 0 ]; then
  echo "✅ Windows 版本构建成功!"
  # 移动到分发目录
  mv "$BUILD_DIR/FastStartInspector_Debug_${VERSION}.exe" "$DIST_DIR/"
else
  echo "❌ Windows 版本构建失败!"
  exit 1
fi

# 2. 编译 macOS 版本
echo "🍎 正在编译 macOS 版本..."
wails build -platform darwin/universal -ldflags "-X main.Version=$VERSION"

if [ $? -eq 0 ]; then
  echo "✅ macOS 版本构建成功!"
  
  # 3. 压缩 macOS 版本 (Zip)
  echo "📦 正在压缩 macOS 应用..."
  cd "$BUILD_DIR"
  zip -r "FastStartInspector_${VERSION}_mac.zip" "mp4-optimizer.app"
  
  if [ $? -eq 0 ]; then
    echo "✅ 压缩成功!"
    # 移动到分发目录 (注意我们需要返回上一级目录结构来定位)
    mv "FastStartInspector_${VERSION}_mac.zip" "../../$DIST_DIR/"
  else
    echo "❌ 压缩失败!"
    exit 1
  fi
  
  cd - > /dev/null
else
  echo "❌ macOS 版本构建失败!"
  exit 1
fi

echo "-----------------------------------"
echo "🎉 构建打包完成！"
echo "📂 输出目录: $DIST_DIR"
echo ""
echo "📝 接下来的步骤:"
echo "1. 打开 Dist 目录: open $DIST_DIR"
echo "2. 将该目录下的 .exe 和 .zip 文件拖拽到 GitHub Release 的 Assets 区域。"
echo "3. 获取下载链接后，更新根目录下的 latest.json 并提交代码。"
