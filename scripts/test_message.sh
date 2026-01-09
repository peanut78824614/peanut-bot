#!/bin/bash

# 测试消息格式脚本

echo "正在发送测试消息到 Telegram..."
echo ""

# 先尝试 POST，如果失败则使用 GET
response=$(curl -s -X POST http://localhost:8000/api/v1/telegram/test 2>&1)

# 如果 POST 失败，尝试 GET
if echo "$response" | grep -q "Not Found\|404\|Method Not Allowed"; then
    echo "POST 方法失败，尝试使用 GET 方法..."
    response=$(curl -s http://localhost:8000/api/v1/telegram/test 2>&1)
fi

echo "响应结果："
echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"

echo ""
echo "请查看 Telegram 群组查看消息格式效果！"
