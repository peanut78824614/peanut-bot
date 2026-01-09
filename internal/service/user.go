package service

import (
	"context"
	"data/internal/model"
	"data/internal/dao"

	"github.com/gogf/gf/v2/frame/g"
)

type IUser interface {
	GetList(ctx context.Context, page, size int) ([]*model.User, int, error)
	GetById(ctx context.Context, id uint64) (*model.User, error)
	Create(ctx context.Context, req *model.CreateUserReq) (*model.User, error)
	Update(ctx context.Context, id uint64, req *model.UpdateUserReq) (*model.User, error)
	Delete(ctx context.Context, id uint64) error
}

type userImpl struct{}

var userService = userImpl{}

// User 获取用户服务实例
func User() IUser {
	return &userService
}

// GetList 获取用户列表
func (s *userImpl) GetList(ctx context.Context, page, size int) ([]*model.User, int, error) {
	return dao.User().GetList(ctx, page, size)
}

// GetById 根据ID获取用户
func (s *userImpl) GetById(ctx context.Context, id uint64) (*model.User, error) {
	return dao.User().GetById(ctx, id)
}

// Create 创建用户
func (s *userImpl) Create(ctx context.Context, req *model.CreateUserReq) (*model.User, error) {
	// 这里可以添加业务逻辑，比如验证、加密密码等
	return dao.User().Create(ctx, req)
}

// Update 更新用户
func (s *userImpl) Update(ctx context.Context, id uint64, req *model.UpdateUserReq) (*model.User, error) {
	return dao.User().Update(ctx, id, req)
}

// Delete 删除用户
func (s *userImpl) Delete(ctx context.Context, id uint64) error {
	return dao.User().Delete(ctx, id)
}

// Init 初始化服务
func Init(ctx context.Context) {
	g.Log().Info(ctx, "服务初始化完成")
}
