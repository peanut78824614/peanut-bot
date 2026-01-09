package controller

import (
	"data/internal/service"
	"data/internal/model"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// UserController 用户控制器
type UserController struct{}

// NewUserController 创建用户控制器
func NewUserController() *UserController {
	return &UserController{}
}

// GetUserList 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表接口
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} model.Response{data=[]model.User}
// @Router /api/v1/users [get]
func (c *UserController) GetUserList(r *ghttp.Request) {
	ctx := r.Context()
	
	page := r.Get("page", 1).Int()
	size := r.Get("size", 10).Int()

	users, total, err := service.User().GetList(ctx, page, size)
	if err != nil {
		g.Log().Error(ctx, "获取用户列表失败:", err)
		r.Response.WriteJson(model.Response{
			Code:    500,
			Message: "获取用户列表失败",
			Data:    nil,
		})
		return
	}

	r.Response.WriteJson(model.Response{
		Code:    200,
		Message: "success",
		Data: g.Map{
			"list":  users,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

// GetUserById 根据ID获取用户
// @Summary 获取用户详情
// @Description 根据ID获取用户详情
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} model.Response{data=model.User}
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetUserById(r *ghttp.Request) {
	ctx := r.Context()
	id := r.Get("id").Uint64()

	user, err := service.User().GetById(ctx, id)
	if err != nil {
		g.Log().Error(ctx, "获取用户失败:", err)
		r.Response.WriteJson(model.Response{
			Code:    500,
			Message: "获取用户失败",
			Data:    nil,
		})
		return
	}

	if user == nil {
		r.Response.WriteJson(model.Response{
			Code:    404,
			Message: "用户不存在",
			Data:    nil,
		})
		return
	}

	r.Response.WriteJson(model.Response{
		Code:    200,
		Message: "success",
		Data:    user,
	})
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body model.CreateUserReq true "用户信息"
// @Success 200 {object} model.Response{data=model.User}
// @Router /api/v1/users [post]
func (c *UserController) CreateUser(r *ghttp.Request) {
	ctx := r.Context()
	
	var req model.CreateUserReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(model.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
			Data:    nil,
		})
		return
	}

	user, err := service.User().Create(ctx, &req)
	if err != nil {
		g.Log().Error(ctx, "创建用户失败:", err)
		r.Response.WriteJson(model.Response{
			Code:    500,
			Message: "创建用户失败",
			Data:    nil,
		})
		return
	}

	r.Response.WriteJson(model.Response{
		Code:    200,
		Message: "创建成功",
		Data:    user,
	})
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param user body model.UpdateUserReq true "用户信息"
// @Success 200 {object} model.Response{data=model.User}
// @Router /api/v1/users/{id} [put]
func (c *UserController) UpdateUser(r *ghttp.Request) {
	ctx := r.Context()
	id := r.Get("id").Uint64()

	var req model.UpdateUserReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJson(model.Response{
			Code:    400,
			Message: "参数错误: " + err.Error(),
			Data:    nil,
		})
		return
	}

	user, err := service.User().Update(ctx, id, &req)
	if err != nil {
		g.Log().Error(ctx, "更新用户失败:", err)
		r.Response.WriteJson(model.Response{
			Code:    500,
			Message: "更新用户失败",
			Data:    nil,
		})
		return
	}

	r.Response.WriteJson(model.Response{
		Code:    200,
		Message: "更新成功",
		Data:    user,
	})
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} model.Response
// @Router /api/v1/users/{id} [delete]
func (c *UserController) DeleteUser(r *ghttp.Request) {
	ctx := r.Context()
	id := r.Get("id").Uint64()

	err := service.User().Delete(ctx, id)
	if err != nil {
		g.Log().Error(ctx, "删除用户失败:", err)
		r.Response.WriteJson(model.Response{
			Code:    500,
			Message: "删除用户失败",
			Data:    nil,
		})
		return
	}

	r.Response.WriteJson(model.Response{
		Code:    200,
		Message: "删除成功",
		Data:    nil,
	})
}
