#!/bin/bash
# intercom_http_service 配置文件上传脚本

# 目标服务器设置
TARGET_HOST="117.72.193.54"
TARGET_PORT="22"
TARGET_USERNAME="root"
TARGET_PASSWORD="1090119your@"

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

# 错误处理函数
function handle_error() {
  print_error "$1"
  exit 1
}

# 检查sshpass是否安装
if ! command -v sshpass &> /dev/null; then
  print_warning "sshpass未安装，将尝试安装..."
  if [[ "$OSTYPE" == "darwin"* ]]; then
    brew install sshpass || handle_error "sshpass安装失败！请手动安装: brew install sshpass"
  elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sudo apt-get update && sudo apt-get install -y sshpass || handle_error "sshpass安装失败！请手动安装: sudo apt-get install sshpass"
  else
    handle_error "无法识别的操作系统，请手动安装sshpass后重试"
  fi
  print_success "sshpass安装成功"
fi

# 定义SSH和SCP命令的函数
function ssh_cmd() {
  export SSHPASS="$TARGET_PASSWORD"
  sshpass -e ssh -o StrictHostKeyChecking=no -p "$TARGET_PORT" "$TARGET_USERNAME@$TARGET_HOST" "$@"
}

function scp_cmd() {
  export SSHPASS="$TARGET_PASSWORD"
  sshpass -e scp -o StrictHostKeyChecking=no -P "$TARGET_PORT" "$@" "$TARGET_USERNAME@$TARGET_HOST:/root/intercom_http_service/"
}

function scp_dir_cmd() {
  export SSHPASS="$TARGET_PASSWORD"
  sshpass -e scp -o StrictHostKeyChecking=no -r -P "$TARGET_PORT" "$@" "$TARGET_USERNAME@$TARGET_HOST:/root/intercom_http_service/"
}

# 获取脚本所在目录的项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

print_info "项目根目录: $PROJECT_ROOT"
print_info "开始上传配置文件到目标服务器: $TARGET_HOST"

# 创建目标服务器目录结构
print_info "创建目标服务器目录结构..."
ssh_cmd "mkdir -p /root/intercom_http_service/internal/infrastructure/mqtt/{config,data,log,certs} /root/intercom_http_service/logs"

# 1. 上传docker-compose.yml文件
print_info "上传docker-compose.yml文件..."
if [ -f "$PROJECT_ROOT/docker-compose.yml" ]; then
  scp_cmd "$PROJECT_ROOT/docker-compose.yml"
  print_success "docker-compose.yml上传成功"
else
  handle_error "docker-compose.yml文件不存在: $PROJECT_ROOT/docker-compose.yml"
fi

# 2. 上传.env文件
print_info "上传.env文件..."
if [ -f "$PROJECT_ROOT/.env" ]; then
  scp_cmd "$PROJECT_ROOT/.env"
  print_success ".env文件上传成功"
else
  print_warning ".env文件不存在: $PROJECT_ROOT/.env"
  print_info "将创建示例.env文件，请根据实际情况修改..."
  
  # 创建示例.env文件
  cat > "/tmp/example.env" << 'EOF'
# 环境配置
ENV_TYPE=SERVER

# 数据库配置
MYSQL_ROOT_PASSWORD=1090119your
MYSQL_DATABASE=intercom

# 阿里云配置
ALIYUN_ACCESS_KEY=your_access_key
ALIYUN_RTC_APP_ID=your_app_id
ALIYUN_RTC_REGION=cn-shanghai

# 默认管理员密码
DEFAULT_ADMIN_PASSWORD=admin123

# MQTT配置
MQTT_BROKER_URL=tcp://mqtt:1883
EOF
  
  scp_cmd "/tmp/example.env"
  ssh_cmd "mv /root/intercom_http_service/example.env /root/intercom_http_service/.env"
  rm -f "/tmp/example.env"
  
  print_warning "已创建示例.env文件，请登录服务器修改配置:"
  print_info "ssh root@$TARGET_HOST"
  print_info "vi /root/intercom_http_service/.env"
fi

# 3. 上传MQTT配置文件
print_info "上传MQTT配置文件..."
if [ -f "$PROJECT_ROOT/internal/infrastructure/mqtt/config/mosquitto.conf" ]; then
  scp_dir_cmd "$PROJECT_ROOT/internal/infrastructure/mqtt/"
  print_success "MQTT配置文件上传成功"
else
  print_warning "MQTT配置文件不存在，将在迁移时创建默认配置"
fi

# 4. 设置正确的权限
print_info "设置文件权限..."
ssh_cmd "chmod 644 /root/intercom_http_service/docker-compose.yml"
ssh_cmd "chmod 600 /root/intercom_http_service/.env"
ssh_cmd "chmod -R 755 /root/intercom_http_service/internal/infrastructure/mqtt/ 2>/dev/null || true"
ssh_cmd "chmod 644 /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf 2>/dev/null || true"

# 验证上传的文件
print_info "验证上传的文件..."
print_info "目标服务器文件列表:"
ssh_cmd "ls -la /root/intercom_http_service/"
ssh_cmd "ls -la /root/intercom_http_service/internal/infrastructure/mqtt/config/ 2>/dev/null || echo 'MQTT配置目录不存在'"

print_success "配置文件上传完成！"
print_info "接下来可以执行迁移脚本:"
print_info "cd $SCRIPT_DIR && ./migrate.sh"

# 提醒用户检查配置
print_warning "重要提醒："
print_info "1. 请确认.env文件中的配置是否正确"
print_info "2. 请确认MQTT配置是否符合要求"
print_info "3. 如有需要，可以手动修改服务器上的配置文件"
print_info "4. 确认无误后，执行迁移脚本进行数据迁移"
