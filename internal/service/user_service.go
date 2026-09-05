package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"nexus/internal/model"
)

const minPasswordLen = 6

// UserService 处理用户注册、登录鉴权与会话（session）管理。
type UserService struct {
	db         *gorm.DB
	sessionTTL time.Duration
}

// NewUserService 构造一个 UserService。sessionTTL 为会话有效期。
func NewUserService(db *gorm.DB, sessionTTL time.Duration) *UserService {
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}
	return &UserService{db: db, sessionTTL: sessionTTL}
}

// Register 创建新用户。用户名唯一，密码用 bcrypt 加盐哈希存储。
func (s *UserService) Register(username, password, displayName string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(password) < minPasswordLen {
		return nil, ErrInvalidInput
	}

	// 唯一性预检查（DB 层 uniqueIndex 兜底，防并发穿透）。
	var count int64
	if err := s.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  strings.TrimSpace(displayName),
	}
	if err := s.db.Create(user).Error; err != nil {
		// 并发下唯一索引冲突也归为「用户名已存在」。
		return nil, ErrUserExists
	}
	return user, nil
}

// Authenticate 校验用户名+密码，成功返回用户。
func (s *UserService) Authenticate(username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)

	var user model.User
	if err := s.db.First(&user, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

// CreateSession 为用户新建一个会话，返回含随机 token 的 Session。
func (s *UserService) CreateSession(userID string) (*model.Session, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	session := &model.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.sessionTTL),
	}
	if err := s.db.Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

// LookupSession 按 token 查会话并校验有效期，返回对应用户。
// token 无效或已过期均返回 ErrUnauthorized（过期会话顺带清理）。
func (s *UserService) LookupSession(token string) (*model.User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}

	var session model.Session
	if err := s.db.First(&session, "token = ?", token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		s.db.Delete(&model.Session{}, "token = ?", token)
		return nil, ErrUnauthorized
	}

	var user model.User
	if err := s.db.First(&user, "id = ?", session.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	return &user, nil
}

// DeleteSession 删除一个会话（登出）。不存在也视为成功。
func (s *UserService) DeleteSession(token string) error {
	if token == "" {
		return nil
	}
	return s.db.Delete(&model.Session{}, "token = ?", token).Error
}

// randomToken 生成 32 字节的随机会话 token（hex 编码）。
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
