#!/bin/bash

# API 测试脚本

BASE_URL="http://localhost:8000/api/v1"

echo "=== 测试健康检查接口 ==="
curl -X GET "${BASE_URL}/health"
echo -e "\n\n"

echo "=== 测试创建用户 ==="
curl -X POST "${BASE_URL}/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三",
    "email": "zhangsan@example.com",
    "phone": "13800138000"
  }'
echo -e "\n\n"

echo "=== 测试获取用户列表 ==="
curl -X GET "${BASE_URL}/users?page=1&size=10"
echo -e "\n\n"

echo "=== 测试获取用户详情（ID=1）==="
curl -X GET "${BASE_URL}/users/1"
echo -e "\n\n"

echo "=== 测试更新用户（ID=1）==="
curl -X PUT "${BASE_URL}/users/1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "李四",
    "email": "lisi@example.com"
  }'
echo -e "\n\n"

echo "=== 测试完成 ==="
