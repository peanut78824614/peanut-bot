package dao

import (
	"context"
	"data/internal/model"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

type IUserDao interface {
	GetList(ctx context.Context, page, size int) ([]*model.User, int, error)
	GetById(ctx context.Context, id uint64) (*model.User, error)
	Create(ctx context.Context, req *model.CreateUserReq) (*model.User, error)
	Update(ctx context.Context, id uint64, req *model.UpdateUserReq) (*model.User, error)
	Delete(ctx context.Context, id uint64) error
}

type userDaoImpl struct{}

var userDao = userDaoImpl{}

// User 获取用户DAO实例
func User() IUserDao {
	return &userDao
}

// GetList 获取用户列表
func (d *userDaoImpl) GetList(ctx context.Context, page, size int) ([]*model.User, int, error) {
	db := g.DB()
	
	// 获取总数
	count, err := db.Model("users").Count()
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	var users []*model.User
	err = db.Model("users").
		Page(page, size).
		OrderDesc("id").
		Scan(&users)
	
	if err != nil {
		return nil, 0, err
	}

	// 如果没有数据，返回空切片而不是 nil
	if users == nil {
		users = make([]*model.User, 0)
	}

	return users, count, nil
}

// GetById 根据ID获取用户
func (d *userDaoImpl) GetById(ctx context.Context, id uint64) (*model.User, error) {
	db := g.DB()
	
	var user model.User
	err := db.Model("users").Where("id", id).Scan(&user)
	if err != nil {
		return nil, err
	}

	// 如果 ID 为 0，说明没有找到记录
	if user.Id == 0 {
		return nil, nil
	}

	return &user, nil
}

// Create 创建用户
func (d *userDaoImpl) Create(ctx context.Context, req *model.CreateUserReq) (*model.User, error) {
	db := g.DB()
	
	data := gdb.Map{
		"name":      req.Name,
		"email":     req.Email,
		"phone":     req.Phone,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	result, err := db.Model("users").Data(data).Insert()
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return d.GetById(ctx, uint64(id))
}

// Update 更新用户
func (d *userDaoImpl) Update(ctx context.Context, id uint64, req *model.UpdateUserReq) (*model.User, error) {
	db := g.DB()
	
	data := gdb.Map{
		"updated_at": time.Now(),
	}
	
	if req.Name != nil {
		data["name"] = *req.Name
	}
	if req.Email != nil {
		data["email"] = *req.Email
	}
	if req.Phone != nil {
		data["phone"] = *req.Phone
	}

	_, err := db.Model("users").Where("id", id).Data(data).Update()
	if err != nil {
		return nil, err
	}

	return d.GetById(ctx, id)
}

// Delete 删除用户
func (d *userDaoImpl) Delete(ctx context.Context, id uint64) error {
	db := g.DB()
	
	_, err := db.Model("users").Where("id", id).Delete()
	return err
}
