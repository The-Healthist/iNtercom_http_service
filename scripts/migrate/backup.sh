#!/bin/bash
# intercom_http_service 数据备份脚本 - 优化版

# 源服务器设置
SOURCE_HOST="39.108.49.167"
SOURCE_PORT="22"
SOURCE_USERNAME="root"
SOURCE_PASSWORD="1090119your@"

# MySQL密码设置（从环境变量获取）
MYSQL_ROOT_PASSWORD="1090119your"

# 备份目录设置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/backup"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="intercom_backup_${TIMESTAMP}"

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
  # 清理临时文件
  if [ -d "$TEMP_DIR" ]; then
    rm -rf "$TEMP_DIR"
  fi
  # 重启服务
  ssh_cmd "cd /root/intercom_http_service && docker-compose up -d"
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

# 定义SSH命令的函数，自动使用密码
function ssh_cmd() {
  export SSHPASS="$SOURCE_PASSWORD"
  sshpass -e ssh -o StrictHostKeyChecking=no -p "$SOURCE_PORT" "$SOURCE_USERNAME@$SOURCE_HOST" "$@"
}

function scp_from_server() {
  export SSHPASS="$SOURCE_PASSWORD"
  sshpass -e scp -o StrictHostKeyChecking=no -P "$SOURCE_PORT" "$SOURCE_USERNAME@$SOURCE_HOST:$1" "$2"
}

# 检查磁盘空间
print_info "检查本地磁盘空间..."
FREE_SPACE=$(df -k . | tail -1 | awk '{print $4}')
REQUIRED_SPACE=2097152  # 2GB in KB
if [ "$FREE_SPACE" -lt "$REQUIRED_SPACE" ]; then
  print_warning "磁盘空间不足，建议至少预留2GB空间。可用空间: $(($FREE_SPACE/1024))MB"
  read -p "是否继续备份？(y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    handle_error "备份已取消"
  fi
fi

# 创建备份目录
mkdir -p "$BACKUP_DIR" || handle_error "无法创建备份目录: $BACKUP_DIR"
print_info "创建备份目录: $BACKUP_DIR"

# 检查备份目录权限
if [ ! -w "$BACKUP_DIR" ]; then
  print_error "备份目录没有写入权限: $BACKUP_DIR"
  print_info "尝试修复权限..."
  chmod -R 755 "$BACKUP_DIR" || handle_error "无法修复备份目录权限，请手动检查"
fi

# 创建临时工作目录
TEMP_DIR=$(mktemp -d) || handle_error "无法创建临时工作目录"
print_info "创建临时工作目录: $TEMP_DIR"

# 检查服务状态并优雅停止
print_info "检查服务状态..."
if ! ssh_cmd "cd /root/intercom_http_service && docker-compose ps" | grep -q "Up"; then
  print_warning "服务似乎未运行，将尝试启动服务..."
  ssh_cmd "cd /root/intercom_http_service && docker-compose up -d"
  sleep 30
fi

print_info "优雅停止服务..."
ssh_cmd "cd /root/intercom_http_service && docker-compose stop app" || print_warning "停止应用服务失败，继续备份..."
sleep 5

# 备份MySQL数据
print_info "备份MySQL数据..."
if ssh_cmd "cd /root/intercom_http_service && docker-compose ps db" | grep -q "Up"; then
  # 直接使用数据目录备份，避免密码问题
  print_info "使用数据目录备份MySQL数据..."
  ssh_cmd "cd /root/intercom_http_service && docker-compose stop db"
  sleep 5
  ssh_cmd "docker run --rm -v intercom_mysql_data:/dbdata:ro -v /tmp:/backup alpine sh -c 'cd /dbdata && tar czf /backup/mysql_data.tar.gz .'"
  scp_from_server "/tmp/mysql_data.tar.gz" "$TEMP_DIR/"
  # 重启数据库
  ssh_cmd "cd /root/intercom_http_service && docker-compose start db"
else
  print_warning "MySQL服务未运行，尝试数据目录备份..."
  ssh_cmd "docker run --rm -v intercom_mysql_data:/dbdata:ro -v /tmp:/backup alpine sh -c 'cd /dbdata && tar czf /backup/mysql_data.tar.gz .'"
  scp_from_server "/tmp/mysql_data.tar.gz" "$TEMP_DIR/"
fi

# 备份Redis数据
print_info "备份Redis数据..."
if ssh_cmd "cd /root/intercom_http_service && docker-compose ps redis" | grep -q "Up"; then
  # 触发Redis数据持久化
  ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T redis redis-cli SAVE"
  sleep 5
fi
ssh_cmd "docker run --rm -v intercom_redis_data:/dbdata:ro -v /tmp:/backup alpine sh -c 'cd /dbdata && tar czf /backup/redis_data.tar.gz .'"
scp_from_server "/tmp/redis_data.tar.gz" "$TEMP_DIR/"

# 备份MQTT数据和配置
print_info "备份MQTT数据和配置..."
# 检查MQTT目录是否存在
if ssh_cmd "test -d /root/intercom_http_service/internal/infrastructure/mqtt && echo 'exists'" | grep -q "exists"; then
  ssh_cmd "cd /root/intercom_http_service && tar czf /tmp/mqtt_backup.tar.gz internal/infrastructure/mqtt/"
  scp_from_server "/tmp/mqtt_backup.tar.gz" "$TEMP_DIR/"
else
  print_warning "MQTT目录不存在，创建默认配置..."
  mkdir -p "$TEMP_DIR/internal/infrastructure/mqtt/config"
  cat > "$TEMP_DIR/internal/infrastructure/mqtt/config/mosquitto.conf" << 'EOF'
listener 1883
allow_anonymous true
persistence true
persistence_location /mosquitto/data/
log_dest file /mosquitto/log/mosquitto.log
log_type all
connection_messages true
log_timestamp true
EOF
  # 在临时目录中创建MQTT备份
  cd "$TEMP_DIR" || handle_error "无法进入临时目录"
  tar czf "mqtt_backup.tar.gz" internal/
  cd - > /dev/null
fi

# 备份环境配置文件
print_info "备份环境配置文件..."
scp_from_server "/root/intercom_http_service/.env" "$TEMP_DIR/" || print_warning ".env文件不存在，将在迁移时创建默认配置"
scp_from_server "/root/intercom_http_service/docker-compose.yml" "$TEMP_DIR/" || print_warning "docker-compose.yml文件不存在或无法访问"

# 创建备份包
print_info "创建备份包..."
cd "$TEMP_DIR" || handle_error "无法进入临时目录"
tar czf "$BACKUP_DIR/${BACKUP_NAME}.tar.gz" * || handle_error "创建备份包失败"

# 验证备份
if [ -f "$BACKUP_DIR/${BACKUP_NAME}.tar.gz" ]; then
  print_success "备份完成！"
  print_info "备份文件位置: $BACKUP_DIR/${BACKUP_NAME}.tar.gz"
  
  # 显示备份文件大小
  BACKUP_SIZE=$(du -h "$BACKUP_DIR/${BACKUP_NAME}.tar.gz" | cut -f1)
  print_info "备份文件大小: $BACKUP_SIZE"
  
  # 验证备份文件完整性
  print_info "验证备份文件完整性..."
  if tar tzf "$BACKUP_DIR/${BACKUP_NAME}.tar.gz" &>/dev/null; then
    print_success "备份文件完整性验证通过！"
  else
    print_error "备份文件可能已损坏，请重新执行备份！"
  fi
else
  handle_error "备份失败！"
fi

# 清理临时文件
print_info "清理临时文件..."
ssh_cmd "rm -f /tmp/mysql_backup.sql /tmp/mysql_data.tar.gz /tmp/redis_data.tar.gz /tmp/mqtt_backup.tar.gz"
rm -rf "$TEMP_DIR"

# 重启服务
print_info "重启服务..."
ssh_cmd "cd /root/intercom_http_service && docker-compose up -d"

# 等待服务启动
print_info "等待服务启动..."
for i in {1..30}; do
  if ssh_cmd "cd /root/intercom_http_service && docker-compose ps" | grep -q "Up (healthy)"; then
    print_success "所有服务已就绪！"
    break
  fi
  if [ $i -eq 30 ]; then
    print_warning "服务启动超时，请手动检查服务状态"
  else
    echo "等待服务就绪... (尝试 $i/30)"
    sleep 2
  fi
done

print_success "备份流程完成！"