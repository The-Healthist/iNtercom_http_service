#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 打印标题
echo -e "${YELLOW}===================================================${NC}"
echo -e "${YELLOW}     intercom_http_service 对讲机后端服务静态代码检查     ${NC}"
echo -e "${YELLOW}===================================================${NC}"

# 检查golangci-lint是否安装
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${RED}错误: golangci-lint 未安装${NC}"
    echo -e "请执行以下命令安装:"
    echo -e "${GREEN}go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2${NC}"
    exit 1
fi

# 显示版本信息
echo -e "${GREEN}使用 golangci-lint 版本:${NC}"
golangci-lint version
echo ""

# 检查是否有.golangci.yaml文件
if [ -f ".golangci.yaml" ]; then
    echo -e "${GREEN}发现配置文件: .golangci.yaml${NC}"
else
    echo -e "${YELLOW}警告: 未找到 .golangci.yaml 配置文件，将使用默认配置${NC}"
fi

echo -e "${GREEN}开始执行静态代码检查...${NC}"
echo -e "${YELLOW}===================================================${NC}"

# 执行golangci-lint
if [ "$1" == "--fix" ]; then
    echo -e "${YELLOW}正在执行代码检查并尝试自动修复...${NC}"
    golangci-lint run --fix ./...
elif [ "$1" == "--new" ]; then
    echo -e "${YELLOW}仅检查自上次提交以来的新代码...${NC}"
    golangci-lint run --new-from-rev=HEAD~1 ./...
else
    golangci-lint run ./...
fi

# 检查执行结果
EXIT_CODE=$?

echo -e "${YELLOW}===================================================${NC}"
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}恭喜! 代码检查通过，未发现问题。${NC}"
else
    echo -e "${RED}发现代码问题，请根据上述提示修复。${NC}"
    echo -e "${YELLOW}提示:${NC}"
    echo -e "  1. 使用 ${GREEN}./scripts/lint.sh --fix${NC} 尝试自动修复部分问题"
    echo -e "  2. 使用 ${GREEN}//nolint:linter名称${NC} 在代码中忽略特定问题"
    echo -e "  3. 修改 ${GREEN}.golangci.yaml${NC} 配置文件调整检查规则"
fi

exit $EXIT_CODE 