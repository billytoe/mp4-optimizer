#!/bin/bash

# 检查是否提供了版本号参数
if [ -z "$1" ]; then
  echo "请提供版本号，例如: ./scripts/build_release.sh v1.0.1"
  exit 1
fi

VERSION=$1

# 标准化版本号：移除开头的 'v' (例如 v0.0.1 -> 0.0.1)
VERSION=${VERSION#v}

BUILD_DIR="build/bin"
DIST_DIR="dist/v$VERSION"

# 固定的发布文件名（不带版本号，简化版本管理）
WIN_EXE_NAME="FastStartInspector.exe"
WIN_DEBUG_EXE="FastStartInspector_Debug.exe"
WIN_ZIP_NAME="FastStartInspector_windows.zip"
WIN_DEBUG_ZIP="FastStartInspector_Debug_windows.zip"
MAC_ZIP_NAME="FastStartInspector_darwin_universal.zip"

echo "🚀 开始构建版本: v$VERSION"
echo "-----------------------------------"

# 清理旧的构建和分发目录
echo "🧹 清理旧文件..."
rm -rf "$BUILD_DIR"
mkdir -p "$DIST_DIR"

# 1. 编译 Windows 版本
echo "🪟 正在编译 Windows 版本..."

# 1.1 Release Build (Hidden Console)
echo "  • Building Release version..."
wails build -platform windows/amd64 -clean -o "$WIN_EXE_NAME" -ldflags "-X main.Version=${VERSION} -H windowsgui"

if [ $? -eq 0 ]; then
  echo "  ✅ Release 版本构建成功!"
  # 打包成 zip（绕过浏览器安全警告）
  cd "$BUILD_DIR"
  zip "$WIN_ZIP_NAME" "$WIN_EXE_NAME"
  mv "$WIN_ZIP_NAME" "../../$DIST_DIR/"
  cd - > /dev/null
else
  echo "  ❌ Release 版本构建失败!"
  exit 1
fi

# 1.2 Debug Build (Console Visible)
echo "  • Building Debug version (Console)..."
wails build -platform windows/amd64 -clean -o "$WIN_DEBUG_EXE" -ldflags "-X main.Version=${VERSION}"

if [ $? -eq 0 ]; then
  echo "  ✅ Debug 版本构建成功!"
  cd "$BUILD_DIR"
  zip "$WIN_DEBUG_ZIP" "$WIN_DEBUG_EXE"
  mv "$WIN_DEBUG_ZIP" "../../$DIST_DIR/"
  cd - > /dev/null
else
  echo "  ❌ Debug 版本构建失败!"
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
  zip -r "$MAC_ZIP_NAME" "mp4-optimizer.app"
  
  if [ $? -eq 0 ]; then
    echo "✅ 压缩成功!"
    mv "$MAC_ZIP_NAME" "../../$DIST_DIR/"
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
echo "📄 发布文件（全部为 zip 格式）:"
echo "   • Windows: $WIN_ZIP_NAME, $WIN_DEBUG_ZIP"
echo "   • macOS:   $MAC_ZIP_NAME"
echo ""
echo "📝 更新 latest.json 示例:"
cat << EOF
{
    "version": "$VERSION",
    "download_url_windows": "https://github.com/billytoe/mp4-optimizer/releases/download/v$VERSION/$WIN_ZIP_NAME",
    "download_url_mac": "https://github.com/billytoe/mp4-optimizer/releases/download/v$VERSION/$MAC_ZIP_NAME",
    "release_notes": "更新说明..."
}
EOF
