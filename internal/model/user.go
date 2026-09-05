package model

import "time"

// User 表示一个注册用户。所有业务数据（Agent 及其下的会话、消息）
// 都通过 Agent.UserID 归属到某个 User，实现按用户隔离。
type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"` // bcrypt hash，绝不下发到前端
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session 是服务端会话记录。Token 为随机不透明串（非 JWT），
// 通过 HttpOnly cookie 下发给浏览器；登出即删除，天然可撤销。
type Session struct {
	Token     string    `gorm:"primaryKey" json:"-"`
	UserID    string    `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
