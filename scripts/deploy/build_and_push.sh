#!/bin/bash
# intercom_http_service 镜像构建和推送脚本

# 版本设置
VERSION="2.3.0"

# Docker Hub settings
DOCKER_USERNAME="stonesea"
DOCKER_PASSWORD="1090119your"

# 颜色输出函数
function print_info() {
  echo -e "\033[0;34m[INFO] $1\033[0m"
}

function print_success() {
  echo -e "\033[0;32m[SUCCESS] $1\033[0m"
}

function print_warning() {
  echo -e "\033[0;33m[WARNING] $1\033[0m"
}

function print_error() {
  echo -e "\033[0;31m[ERROR] $1\033[0m"
}

# 切换到项目根目录
cd "$(dirname "$0")/../.." || { print_error "无法切换到项目根目录"; exit 1; }
print_info "Working from project root: $(pwd)"

# 检查命令是否存在
command -v docker >/dev/null 2>&1 || { print_error "需要安装Docker！"; exit 1; }

# 检查Docker版本是否支持buildx
if ! docker buildx version >/dev/null 2>&1; then
  print_error "您的Docker版本不支持buildx，请升级到最新版本的Docker Desktop"
  exit 1
fi

# 配置Docker镜像加速器
function configure_docker_mirrors() {
  print_info "配置Docker镜像加速器..."
  
  # 创建或修改Docker配置目录
  mkdir -p ~/.docker
  
  # 配置多个镜像源，按优先级排序
  cat > ~/.docker/config.json << EOF
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://dockerproxy.net",
    "https://hub.rat.dev",
    "https://docker.1ms.run"
  ]
}
EOF
  
  print_info "Docker镜像加速器配置完成"
}

# 配置Docker镜像加速器
configure_docker_mirrors

# 登录到Docker Hub
print_info "登录到Docker Hub..."
echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

# 使用buildx构建多平台镜像
print_info "Building Docker image version $VERSION with buildx for multiple platforms..."
docker buildx create --use --name multi-platform-builder || true
docker buildx inspect --bootstrap

# 构建并推送多平台镜像
print_info "Building and pushing multi-platform image for version $VERSION..."
docker buildx build --platform linux/amd64 \
  -t "$DOCKER_USERNAME/intercom-http-service:$VERSION" \
  -t "$DOCKER_USERNAME/intercom-http-service:latest" \
  --push .

print_success "镜像构建和推送完成！"
print_info "镜像地址: $DOCKER_USERNAME/intercom-http-service:$VERSION" 