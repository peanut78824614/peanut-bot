package database

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

// Init 初始化数据库
func Init(ctx context.Context) error {
	db := g.DB()
	
	// 检查表是否存在
	hasTable, err := db.GetValue(ctx, `
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='users'
	`)
	if err != nil {
		return err
	}

	// 如果表不存在，则创建表
	if hasTable == nil || hasTable.IsEmpty() {
		g.Log().Info(ctx, "初始化数据库表...")
		
		// 创建用户表
		_, err = db.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(100) NOT NULL,
				phone VARCHAR(20) NOT NULL,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			)
		`)
		if err != nil {
			return err
		}

		// 创建索引
		_, err = db.Exec(ctx, `
			CREATE INDEX IF NOT EXISTS idx_email ON users(email)
		`)
		if err != nil {
			return err
		}

		_, err = db.Exec(ctx, `
			CREATE INDEX IF NOT EXISTS idx_phone ON users(phone)
		`)
		if err != nil {
			return err
		}

		g.Log().Info(ctx, "数据库表初始化完成")
	}

	return nil
}
