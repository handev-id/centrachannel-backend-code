package user

import "encoding/json"

type CreateUserRequest struct {
	FirstName string          `json:"first_name" validate:"required,max=255"`
	LastName  *string         `json:"last_name,omitempty" validate:"omitempty,max=255"`
	Username  string          `json:"username" validate:"required,min=3,max=50"`
	Email     string          `json:"email" validate:"required,email"`
	Phone     *string         `json:"phone,omitempty" validate:"omitempty,max=20"`
	Password  string          `json:"password" validate:"required,min=8"`
	Avatar    json.RawMessage `json:"avatar,omitempty"`
	Roles     []int           `json:"roles" validate:"required,min=1"`
}

type UpdateUserRequest struct {
	FirstName string          `json:"first_name" validate:"required,max=255"`
	LastName  *string         `json:"last_name,omitempty" validate:"omitempty,max=255"`
	Username  string          `json:"username" validate:"required,min=3,max=50"`
	Email     string          `json:"email" validate:"required,email"`
	Phone     *string         `json:"phone,omitempty" validate:"omitempty,max=20"`
	Password  *string         `json:"password,omitempty" validate:"omitempty,min=8"`
	Avatar    json.RawMessage `json:"avatar,omitempty"`
	Roles     []int           `json:"roles" validate:"required,min=1"`
}

type ListUserQuery struct {
	Page   int    `query:"page" validate:"omitempty,min=1"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Search string `query:"search"`
	RoleID *int   `query:"role_id"`
	SortBy string `query:"sort_by"`
}

