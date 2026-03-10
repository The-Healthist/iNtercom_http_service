#!/bin/bash
# intercom_http_service 服务器部署脚本

# 版本设置
VERSION="2.3.0"

# Server settings
SSH_HOST="39.108.49.167"
SSH_PORT="22"
SSH_USERNAME="root"
SSH_PASSWORD="1090119your@"

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

# 检查sshpass是否安装
if ! command -v sshpass &> /dev/null; then
  print_warning "sshpass未安装，将尝试安装..."
  if [[ "$OSTYPE" == "darwin"* ]]; then
    brew install sshpass || { 
      print_error "sshpass安装失败！请手动安装: brew install sshpass"; 
      exit 1; 
    }
  elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    sudo apt-get update && sudo apt-get install -y sshpass || { print_error "sshpass安装失败！请手动安装: sudo apt-get install sshpass"; exit 1; }
  else
    print_error "无法识别的操作系统，请手动安装sshpass后重试"; 
    exit 1;
  fi
  print_success "sshpass安装成功"
fi

# 定义SSH和SCP命令的函数，自动使用密码
function ssh_cmd() {
  export SSHPASS="$SSH_PASSWORD"
  sshpass -e ssh -o StrictHostKeyChecking=no -p "$SSH_PORT" "$SSH_USERNAME@$SSH_HOST" "$@"
}

function scp_cmd() {
  export SSHPASS="$SSH_PASSWORD"
  sshpass -e scp -o StrictHostKeyChecking=no -P "$SSH_PORT" "$@" "$SSH_USERNAME@$SSH_HOST:/root/intercom_http_service/"
}

# 创建docker-compose.yml文件
cat > docker-compose.yml << EOF
services: 
  app: 
    image: stonesea/intercom-http-service:$VERSION
    container_name: intercom_http_service 
    restart: always 
    ports: 
      - '20033:20033'
    volumes: 
      - ./logs:/app/logs 
      - ./.env:/app/.env 
    environment: 
      - ENV_TYPE=SERVER 
      - ALIYUN_ACCESS_KEY=\${ALIYUN_ACCESS_KEY} 
      - ALIYUN_RTC_APP_ID=\${ALIYUN_RTC_APP_ID} 
      - ALIYUN_RTC_REGION=\${ALIYUN_RTC_REGION} 
      - DEFAULT_ADMIN_PASSWORD=\${DEFAULT_ADMIN_PASSWORD} 
      - LOCAL_DB_HOST=db
      - LOCAL_DB_USER=root
      - LOCAL_DB_PASSWORD=\${MYSQL_ROOT_PASSWORD}
      - LOCAL_DB_NAME=\${MYSQL_DATABASE}
      - LOCAL_DB_PORT=3306
      - SERVER_DB_HOST=db
      - SERVER_DB_USER=root
      - SERVER_DB_PASSWORD=\${MYSQL_ROOT_PASSWORD}
      - SERVER_DB_NAME=\${MYSQL_DATABASE}
      - SERVER_DB_PORT=3306
      - MQTT_BROKER_URL=tcp://mqtt:1883
    depends_on: 
      db: 
        condition: service_healthy 
      redis: 
        condition: service_healthy 
    networks: 
      - intercom_network 
    healthcheck: 
      test: ['CMD', 'curl', '-f', 'http://localhost:20033/api/ping']
      interval: 10s 
      timeout: 5s 
      retries: 3 
      start_period: 10s 
 
  db: 
    image: mysql:8.0 
    container_name: intercom_mysql 
    restart: always 
    ports: 
      - '3310:3306'
    volumes: 
      - mysql_data:/var/lib/mysql 
    environment: 
      - MYSQL_ROOT_PASSWORD=\${MYSQL_ROOT_PASSWORD} 
      - MYSQL_DATABASE=\${MYSQL_DATABASE} 
    command: --default-authentication-plugin=mysql_native_password 
    networks: 
      - intercom_network 
    healthcheck: 
      test: ['CMD', 'mysqladmin', 'ping', '-h', 'localhost']
      interval: 10s 
      timeout: 5s 
      retries: 3 
 
  redis: 
    image: redis:7.0-alpine 
    container_name: intercom_redis 
    restart: always 
    ports: 
      - '6380:6379'
    volumes: 
      - redis_data:/data 
    networks: 
      - intercom_network 
    healthcheck: 
      test: ['CMD', 'redis-cli', 'ping']
      interval: 10s 
      timeout: 5s 
      retries: 3 
       
  mqtt: 
    image: eclipse-mosquitto:2.0 
    container_name: intercom_mqtt 
    restart: always 
    ports: 
      - '1883:1883'
      - '8883:8883'
      - '9001:9001'
    volumes: 
      - ./mqtt/config:/mosquitto/config 
      - ./mqtt/data:/mosquitto/data 
      - ./mqtt/log:/mosquitto/log 
    networks: 
      - intercom_network 
    healthcheck: 
      test: ['CMD', 'mosquitto_sub', '-t', '\$SYS/#', '-C', '1', '-i', 'healthcheck', '-W', '3']
      interval: 10s 
      timeout: 5s 
      retries: 3 
 
networks: 
  intercom_network: 
    driver: bridge 
 
volumes: 
  mysql_data: 
  redis_data:  
EOF

# 准备MQTT配置文件
mkdir -p mqtt/config
cat > mqtt/config/mosquitto.conf << 'EOF'
# 监听端口
listener 1883
listener 8883
listener 9001
protocol websockets

# 持久化设置
persistence true
persistence_location /mosquitto/data/
persistence_file mosquitto.db

# 日志设置
log_dest file /mosquitto/log/mosquitto.log
log_type all

# 默认允许匿名访问
allow_anonymous true
EOF

# 准备Docker镜像加速配置
cat > setup_docker_mirror.sh << 'EOF'
#!/bin/bash

# 创建或更新Docker配置目录
mkdir -p /etc/docker

# 创建daemon.json配置文件
cat > /etc/docker/daemon.json << 'INNEREOF'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://dockerproxy.net",
    "https://hub.rat.dev",
    "https://docker.1ms.run"
  ]
}
INNEREOF

# 重启Docker服务
systemctl daemon-reload
systemctl restart docker

echo "Docker镜像加速配置完成"
EOF

# 复制文件到服务器
print_info "复制部署文件到服务器..."
scp_cmd docker-compose.yml .env setup_docker_mirror.sh

# 创建MQTT所需目录
print_info "创建MQTT所需目录..."
ssh_cmd "cd /root/intercom_http_service && mkdir -p mqtt/config mqtt/data mqtt/log"

# 上传MQTT配置文件
print_info "上传MQTT配置文件..."
scp_cmd mqtt/config/mosquitto.conf mqtt/config/

# 配置Docker镜像加速
print_info "配置Docker镜像加速..."
ssh_cmd "cd /root/intercom_http_service && chmod +x setup_docker_mirror.sh && ./setup_docker_mirror.sh"

# 停止现有服务
print_info "停止现有服务..."
ssh_cmd "cd /root/intercom_http_service && docker-compose down"

# 拉取新镜像
print_info "拉取新镜像..."
ssh_cmd "cd /root/intercom_http_service && docker-compose pull"

# 启动服务
print_info "启动服务..."
ssh_cmd "cd /root/intercom_http_service && docker-compose up -d"

# 等待服务就绪
print_info "等待服务就绪..."
ssh_cmd "cd /root/intercom_http_service && for i in {1..30}; do if docker-compose ps | grep -q 'Up (healthy)'; then echo '所有服务已就绪！'; break; fi; if [ \$i -eq 30 ]; then echo '服务启动超时'; docker-compose logs; exit 1; fi; echo '等待服务就绪... (尝试 '\$i'/30)'; sleep 2; done"

# 检查服务状态
print_info "检查服务状态..."
ssh_cmd "cd /root/intercom_http_service && docker-compose ps"

# 清理临时文件
rm -f setup_docker_mirror.sh docker-compose.yml
rm -rf mqtt

print_success "部署完成！"
print_info "服务状态："
ssh_cmd "cd /root/intercom_http_service && docker-compose ps" 