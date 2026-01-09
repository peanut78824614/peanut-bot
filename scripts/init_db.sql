-- 初始化数据库表结构

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL COMMENT '姓名',
  `email` VARCHAR(100) NOT NULL COMMENT '邮箱',
  `phone` VARCHAR(20) NOT NULL COMMENT '手机号',
  `created_at` DATETIME NOT NULL COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL COMMENT '更新时间'
);

-- 创建索引
CREATE INDEX IF NOT EXISTS `idx_email` ON `users`(`email`);
CREATE INDEX IF NOT EXISTS `idx_phone` ON `users`(`phone`);
