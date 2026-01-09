package model

import "time"

// User 用户模型
type User struct {
	Id        uint64    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Name  string `json:"name" v:"required|length:2,50#姓名不能为空|姓名长度应在2-50之间"`
	Email string `json:"email" v:"required|email#邮箱不能为空|邮箱格式不正确"`
	Phone string `json:"phone" v:"required|phone#手机号不能为空|手机号格式不正确"`
}

// UpdateUserReq 更新用户请求
type UpdateUserReq struct {
	Name  *string `json:"name" v:"length:2,50#姓名长度应在2-50之间"`
	Email *string `json:"email" v:"email#邮箱格式不正确"`
	Phone *string `json:"phone" v:"phone#手机号格式不正确"`
}

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
