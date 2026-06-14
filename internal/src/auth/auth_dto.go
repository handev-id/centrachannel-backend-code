package auth

import "encoding/json"

type RegisterRequest struct {
	FirstName string          `json:"first_name" validate:"required,max=255"`
	LastName  *string         `json:"last_name,omitempty" validate:"omitempty,max=255"`
	Username  string          `json:"username" validate:"required,min=3,max=50"`
	Email     string          `json:"email" validate:"required,email"`
	Phone     *string         `json:"phone,omitempty" validate:"omitempty,max=20"`
	Avatar    json.RawMessage `json:"avatar,omitempty"`
	Password  string          `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}
