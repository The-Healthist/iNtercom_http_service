#!/bin/bash
# intercom_http_service 对讲机后端服务迁移脚本 - 优化版

# 版本设置
VERSION="2.3.0"

# 目标服务器设置
TARGET_HOST="117.72.193.54"
TARGET_PORT="22"
TARGET_USERNAME="root"
TARGET_PASSWORD="1090119your@"

# MySQL密码设置（从环境变量获取）
MYSQL_ROOT_PASSWORD="1090119your"

# 备份目录设置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/backup"
LATEST_BACKUP=$(ls -t "$BACKUP_DIR"/*.tar.gz 2>/dev/null | head -n1)

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
  if [ -d "$TEMP_DIR" ]; then
    rm -rf "$TEMP_DIR"
  fi
  exit 1
}

# 显示帮助信息
function show_help() {
  echo "用法: $0 [选项]"
  echo "选项:"
  echo "  -c, --clean     清理目标服务器上的所有容器和数据卷"
  echo "  -h, --help      显示此帮助信息"
  echo "  -f, --force     强制执行，不提示确认"
  echo "示例:"
  echo "  $0              # 正常迁移"
  echo "  $0 -c          # 清理并迁移"
  echo "  $0 -c -f       # 强制清理并迁移"
}

# 解析命令行参数
CLEAN_MODE=false
FORCE_MODE=false

while [[ "$#" -gt 0 ]]; do
  case $1 in
    -c|--clean) CLEAN_MODE=true ;;
    -f|--force) FORCE_MODE=true ;;
    -h|--help) show_help; exit 0 ;;
    *) handle_error "未知参数: $1" ;;
  esac
  shift
done

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

# 检查备份文件
if [ ! -f "$LATEST_BACKUP" ]; then
  handle_error "未找到备份文件！请先运行 backup.sh"
fi

print_info "使用备份文件: $LATEST_BACKUP"

# 如果是清理模式，确认用户意图
if [ "$CLEAN_MODE" = true ] && [ "$FORCE_MODE" = false ]; then
  print_warning "警告：清理模式将删除目标服务器上的所有容器和数据！"
  read -p "确定要继续吗？(y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    handle_error "操作已取消"
  fi
fi

# 创建临时工作目录
TEMP_DIR=$(mktemp -d) || handle_error "无法创建临时工作目录"
print_info "创建临时工作目录: $TEMP_DIR"

# 解压备份文件到临时目录
print_info "解压备份文件到临时目录..."
tar xzf "$LATEST_BACKUP" -C "$TEMP_DIR" || handle_error "备份文件解压失败！请检查备份文件是否完整"

# 如果是清理模式，执行清理
if [ "$CLEAN_MODE" = true ]; then
  print_info "执行清理操作..."
  ssh_cmd "cd /root/intercom_http_service && docker-compose down -v || true"
  ssh_cmd "docker system prune -af || true"
  ssh_cmd "docker volume rm intercom_mysql_data intercom_redis_data || true"
  ssh_cmd "rm -rf /root/intercom_http_service/* || true"
fi

# 创建目标服务器目录
print_info "创建目标服务器目录..."
ssh_cmd "mkdir -p /root/intercom_http_service/internal/infrastructure/mqtt/{config,data,log,certs} /root/intercom_http_service/logs"

# 检查必要文件是否已手动上传到目标服务器
print_info "检查必要文件是否已手动上传..."

# 检查docker-compose.yml文件
if ! ssh_cmd "test -f /root/intercom_http_service/docker-compose.yml && echo 'exists'" | grep -q "exists"; then
  print_error "docker-compose.yml文件不存在于目标服务器上！"
  print_info "请先手动上传docker-compose.yml文件到目标服务器："
  print_info "scp docker-compose.yml root@${TARGET_HOST}:/root/intercom_http_service/"
  handle_error "缺少必要的配置文件"
else
  print_success "docker-compose.yml文件已存在"
fi

# 检查.env文件
if ! ssh_cmd "test -f /root/intercom_http_service/.env && echo 'exists'" | grep -q "exists"; then
  print_error ".env文件不存在于目标服务器上！"
  print_info "请先手动上传.env文件到目标服务器："
  print_info "scp .env root@${TARGET_HOST}:/root/intercom_http_service/"
  handle_error "缺少必要的环境配置文件"
else
  print_success ".env文件已存在"
fi

# 检查MQTT配置是否已上传
if ! ssh_cmd "test -f /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf && echo 'exists'" | grep -q "exists"; then
  print_warning "MQTT配置文件不存在，将创建默认配置"
else
  print_success "MQTT配置文件已存在"
fi

# 上传现有的MQTT配置文件（如果存在）
print_info "配置MQTT服务..."
ssh_cmd "mkdir -p /root/intercom_http_service/internal/infrastructure/mqtt/{config,data,log,certs}"

# 检查是否已有MQTT配置文件，如果没有则创建默认配置
if ! ssh_cmd "test -f /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf && echo 'exists'" | grep -q "exists"; then
  print_info "创建默认MQTT配置文件..."
  ssh_cmd "cat > /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf << 'EOF'
# MQTT Configuration for intercom_http_service
# 基本监听器
listener 1883
allow_anonymous true

# WebSocket监听器  
listener 9001
protocol websockets

# 持久化设置
persistence true
persistence_location /mosquitto/data/

# 日志设置
log_dest file /mosquitto/log/mosquitto.log
log_dest stdout
log_type error
log_type warning
log_type notice
log_type information
connection_messages true
log_timestamp true

# 系统设置
sys_interval 10
max_inflight_messages 40
max_queued_messages 500
message_size_limit 0
allow_zero_length_clientid true
persistent_client_expiration 2m
EOF"
else
  print_success "使用已上传的MQTT配置文件"
fi

# 创建初始日志文件
print_info "初始化MQTT日志文件..."
ssh_cmd "touch /root/intercom_http_service/internal/infrastructure/mqtt/log/mosquitto.log"

# 设置正确的权限（关键步骤）
print_info "设置MQTT目录权限..."
ssh_cmd "chown -R root:root /root/intercom_http_service/internal/infrastructure/mqtt"
ssh_cmd "chmod -R 755 /root/intercom_http_service/internal/infrastructure/mqtt"
ssh_cmd "chmod 644 /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf"
ssh_cmd "chmod 666 /root/intercom_http_service/internal/infrastructure/mqtt/log/mosquitto.log"

# 确保数据目录可写
ssh_cmd "chmod 777 /root/intercom_http_service/internal/infrastructure/mqtt/data"
ssh_cmd "chmod 777 /root/intercom_http_service/internal/infrastructure/mqtt/log"

# 先启动MySQL容器
print_info "启动MySQL容器..."
ssh_cmd "cd /root/intercom_http_service && docker-compose up -d db"
print_info "等待MySQL启动..."
sleep 15

# 确保intercom数据库存在
print_info "确保intercom数据库存在..."
ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root -e \"CREATE DATABASE IF NOT EXISTS intercom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\"'"

# 恢复MySQL数据
print_info "恢复MySQL数据..."
if [ -f "$TEMP_DIR/mysql_backup.sql" ]; then
  print_info "使用SQL备份恢复MySQL数据..."
  scp_cmd "$TEMP_DIR/mysql_backup.sql"
  
  # 恢复数据
  ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root < /var/lib/mysql/mysql_backup.sql'" || print_warning "MySQL数据恢复可能失败，请检查日志"
elif [ -f "$TEMP_DIR/mysql_data.tar.gz" ]; then
  print_info "使用数据目录备份恢复MySQL数据..."
  # 停止MySQL容器
  ssh_cmd "cd /root/intercom_http_service && docker-compose stop db"
  sleep 5
  
  # 创建并恢复数据卷
  ssh_cmd "docker volume create intercom_mysql_data || true"
  scp_cmd "$TEMP_DIR/mysql_data.tar.gz"
  ssh_cmd "cd /root/intercom_http_service && docker run --rm -v intercom_mysql_data:/dbdata -v /root/intercom_http_service:/backup alpine sh -c 'cd /dbdata && rm -rf * && tar xzf /backup/mysql_data.tar.gz'"
  
  # 重启MySQL容器
  ssh_cmd "cd /root/intercom_http_service && docker-compose up -d db"
  print_info "等待MySQL重启..."
  sleep 20
  
  # 再次确保intercom数据库存在
  print_info "再次确保intercom数据库存在..."
  ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root -e \"CREATE DATABASE IF NOT EXISTS intercom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\"'"
  
  # 如果需要，可以创建初始表结构
  print_info "检查是否需要创建初始表结构..."
  TABLE_COUNT=$(ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root -e \"USE intercom; SHOW TABLES;\" | wc -l'")
  if [ "$TABLE_COUNT" -le 1 ]; then
    print_warning "数据库表可能不存在，创建初始表结构..."
    # 上传初始SQL文件（如果有的话）
    if [ -f "$SCRIPT_DIR/init.sql" ]; then
      print_info "上传初始SQL文件..."
      scp_cmd "$SCRIPT_DIR/init.sql"
      ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root intercom < /var/lib/mysql/init.sql'"
    else
      print_warning "未找到初始SQL文件，将使用空数据库"
      # 创建最基本的表结构
      cat > "$TEMP_DIR/basic_schema.sql" << 'EOF'
-- 基本表结构
CREATE TABLE IF NOT EXISTS `admin` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 添加默认管理员
INSERT INTO `admin` (`username`, `password`) VALUES ('admin', '$2a$10$QZCGj2aBA6ZpDU1i6HQEWuqgBJZXXsj.bzMQbx2/3wb0zOqw5gvAS');
EOF
      scp_cmd "$TEMP_DIR/basic_schema.sql"
      ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root intercom < /var/lib/mysql/basic_schema.sql'"
    fi
  fi
else
  print_warning "未找到MySQL备份，创建空数据库和基本表结构..."
  # 创建初始数据库
  ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root -e \"CREATE DATABASE IF NOT EXISTS intercom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;\"'"
  
  # 创建最基本的表结构
  cat > "$TEMP_DIR/basic_schema.sql" << 'EOF'
-- 基本表结构
CREATE TABLE IF NOT EXISTS `admin` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(255) NOT NULL,
  `password` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 添加默认管理员
INSERT INTO `admin` (`username`, `password`) VALUES ('admin', '$2a$10$QZCGj2aBA6ZpDU1i6HQEWuqgBJZXXsj.bzMQbx2/3wb0zOqw5gvAS');
EOF
  scp_cmd "$TEMP_DIR/basic_schema.sql"
  ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T db sh -c 'MYSQL_PWD=${MYSQL_ROOT_PASSWORD} mysql -u root intercom < /var/lib/mysql/basic_schema.sql'"
fi

# 恢复Redis数据
print_info "恢复Redis数据..."
if [ -f "$TEMP_DIR/redis_data.tar.gz" ]; then
  ssh_cmd "docker volume create intercom_redis_data || true"
  scp_cmd "$TEMP_DIR/redis_data.tar.gz"
  ssh_cmd "cd /root/intercom_http_service && docker-compose stop redis || true"
  ssh_cmd "cd /root/intercom_http_service && docker run --rm -v intercom_redis_data:/dbdata -v /root/intercom_http_service:/backup alpine sh -c 'cd /dbdata && rm -rf * && tar xzf /backup/redis_data.tar.gz'"
  ssh_cmd "cd /root/intercom_http_service && docker-compose up -d redis"
else
  print_warning "未找到Redis备份，将使用空缓存"
  ssh_cmd "cd /root/intercom_http_service && docker-compose up -d redis"
fi

# 启动MQTT服务
print_info "启动MQTT服务..."

# 首先停止任何现有的MQTT容器
ssh_cmd "cd /root/intercom_http_service && docker-compose stop mqtt"
ssh_cmd "cd /root/intercom_http_service && docker-compose rm -f mqtt"

# 验证配置文件
print_info "验证MQTT配置文件..."
ssh_cmd "ls -la /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf"
ssh_cmd "head -5 /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf"

# 启动MQTT服务
print_info "启动新的MQTT服务..."
ssh_cmd "cd /root/intercom_http_service && docker-compose up -d mqtt"

# 等待服务启动并检查
print_info "等待MQTT服务启动..."
sleep 15

# 检查MQTT服务状态
print_info "检查MQTT服务状态..."
if ssh_cmd "cd /root/intercom_http_service && docker-compose ps mqtt" | grep -q "Up"; then
  print_success "MQTT服务已成功启动！"
  
  # 验证容器内配置文件
  print_info "验证容器内配置文件..."
  if ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T mqtt cat /mosquitto/config/mosquitto.conf | head -3" 2>/dev/null | grep -q "MQTT Configuration"; then
    print_success "容器内配置文件验证成功！"
  else
    print_warning "容器内配置文件可能有问题"
    ssh_cmd "cd /root/intercom_http_service && docker-compose exec -T mqtt ls -la /mosquitto/config/" 2>/dev/null || print_warning "无法访问容器内配置目录"
  fi
  
  # 测试MQTT连接
  print_info "测试MQTT连接..."
  if ssh_cmd "cd /root/intercom_http_service && timeout 10 docker-compose exec -T mqtt mosquitto_sub -t test -C 1 >/dev/null 2>&1"; then
    print_success "MQTT连接测试成功！"
  else
    print_warning "MQTT连接测试失败，但服务已启动"
  fi
else
  print_warning "MQTT服务启动失败，查看详细日志..."
  ssh_cmd "cd /root/intercom_http_service && docker-compose logs --tail=20 mqtt"
  
  print_info "尝试使用最基础配置..."
  
  # 停止服务
  ssh_cmd "cd /root/intercom_http_service && docker-compose stop mqtt"
  
  # 创建最基础的配置
  ssh_cmd "cat > /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf << 'EOF'
listener 1883
allow_anonymous true
persistence false
log_dest stdout
EOF"
  
  # 设置权限
  ssh_cmd "chmod 644 /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf"
  
  # 重新启动
  print_info "使用基础配置重启MQTT服务..."
  ssh_cmd "cd /root/intercom_http_service && docker-compose up -d mqtt"
  sleep 10
  
  # 再次检查
  if ssh_cmd "cd /root/intercom_http_service && docker-compose ps mqtt" | grep -q "Up"; then
    print_success "MQTT服务使用基础配置启动成功！"
  else
    print_error "MQTT服务仍然无法启动"
    ssh_cmd "cd /root/intercom_http_service && docker-compose logs --tail=30 mqtt"
  fi
fi

# 清理临时文件
print_info "清理临时文件..."
ssh_cmd "cd /root/intercom_http_service && rm -f mysql_backup.sql mysql_data.tar.gz redis_data.tar.gz basic_schema.sql"
rm -rf "$TEMP_DIR"

# 最终验证所有基础服务状态
print_info "验证所有基础服务状态..."
print_info "等待所有服务完全就绪..."
sleep 5

# 检查数据库服务
if ssh_cmd "cd /root/intercom_http_service && docker-compose ps db" | grep -q "Up"; then
  print_success "MySQL数据库服务正常运行"
else
  print_error "MySQL数据库服务异常"
fi

# 检查Redis服务  
if ssh_cmd "cd /root/intercom_http_service && docker-compose ps redis" | grep -q "Up"; then
  print_success "Redis缓存服务正常运行"
else
  print_error "Redis缓存服务异常"
fi

# 检查MQTT服务
if ssh_cmd "cd /root/intercom_http_service && docker-compose ps mqtt" | grep -q "Up"; then
  print_success "MQTT消息服务正常运行"
else
  print_error "MQTT消息服务异常"
fi

# 启动应用服务
print_info "启动应用服务..."
ssh_cmd "cd /root/intercom_http_service && docker-compose up -d app"

# 等待服务启动
print_info "等待服务启动..."
for i in {1..60}; do
  if ssh_cmd "cd /root/intercom_http_service && docker-compose ps app" | grep -q "Up"; then
    print_success "应用服务已启动！"
    break
  fi
  if [ $i -eq 60 ]; then
    print_warning "应用服务启动超时，请检查服务状态"
    ssh_cmd "cd /root/intercom_http_service && docker-compose ps"
    print_info "查看应用日志..."
    ssh_cmd "cd /root/intercom_http_service && docker-compose logs app"
  else
    echo "等待应用服务就绪... (尝试 $i/60)"
    sleep 5
  fi
done

# 验证服务状态
print_info "验证服务状态..."
if ssh_cmd "cd /root/intercom_http_service && curl -s -o /dev/null -w '%{http_code}' http://localhost:20033/api/ping" | grep -q "200"; then
  print_success "服务健康检查通过！"
else
  print_warning "服务可能未正常运行，请检查日志"
  ssh_cmd "cd /root/intercom_http_service && docker-compose logs app"
fi

print_success "迁移完成！"
print_info "服务状态："
ssh_cmd "cd /root/intercom_http_service && docker-compose ps"

# 最后提醒
print_warning "重要提示："
print_info "1. 请确保已手动上传.env文件到服务器的/root/intercom_http_service/目录"
print_info "2. 检查各服务是否正常运行"
print_info "3. 验证API是否可以正常访问"
print_info "4. 如需查看日志，请使用: docker-compose logs"
print_info "5. 如果应用服务未正常启动，可尝试重启: docker-compose restart app"

# MQTT故障排除指南
if ! ssh_cmd "cd /root/intercom_http_service && docker-compose ps mqtt" | grep -q "Up"; then
  print_error "MQTT服务未正常启动，请执行以下步骤："
  print_info "步骤1: 手动上传MQTT配置文件"
  print_info "   scp internal/infrastructure/mqtt/config/mosquitto.conf root@${TARGET_HOST}:/root/intercom_http_service/internal/infrastructure/mqtt/config/"
  print_info "步骤2: 设置正确权限"
  print_info "   ssh root@${TARGET_HOST} 'chmod 644 /root/intercom_http_service/internal/infrastructure/mqtt/config/mosquitto.conf'"
  print_info "步骤3: 重启MQTT服务"
  print_info "   ssh root@${TARGET_HOST} 'cd /root/intercom_http_service && docker-compose restart mqtt'"
  print_info "步骤4: 检查服务状态"
  print_info "   ssh root@${TARGET_HOST} 'cd /root/intercom_http_service && docker-compose ps mqtt'"
fi 