package models

type User struct {
	AutoID     int    `json:"auto_id"`
	UserName   string `json:"user_name"`
	UserPassword string `json:"-"`
	UserLevel  string `json:"user_level"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
