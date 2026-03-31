package biz

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"horserun/internal/di"
	"horserun/internal/model"
)

var (
	ErrCodeNotFound    = errors.New("授权码不存在")
	ErrCodeExpired     = errors.New("授权码已过期")
	ErrCodeAlreadyUsed = errors.New("授权码已被使用")
	ErrInvalidCode     = errors.New("无效的授权码")
)

// Manager 授权码管理器
type Manager struct {
	mu    sync.RWMutex
	codes map[string]*model.AuthCode
}

// NewManager 创建授权码管理器
func NewManager() *Manager {
	m := &Manager{
		codes: make(map[string]*model.AuthCode),
	}

	// 从数据库加载授权码
	if err := m.LoadCodes(); err != nil {
		log.Printf("警告：加载授权码失败：%v\n", err)
	}

	return m
}

// LoadCodes 从数据库加载授权码
func (m *Manager) LoadCodes() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var codes []*model.AuthCode
	result := di.DB.Find(&codes)
	if result.Error != nil {
		return fmt.Errorf("从数据库加载授权码失败：%w", result.Error)
	}

	// 将加载的授权码存入内存
	for _, code := range codes {
		m.codes[code.Code] = code
	}

	log.Printf("成功加载 %d 个授权码\n", len(codes))
	return nil
}

// SaveCodes 保存授权码到数据库
func (m *Manager) SaveCodes() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, code := range m.codes {
		result := di.DB.Save(code)
		if result.Error != nil {
			return fmt.Errorf("保存授权码到数据库失败：%w", result.Error)
		}
	}

	log.Printf("成功保存 %d 个授权码到数据库\n", len(m.codes))
	return nil
}

// saveIfNeeded 检查是否需要保存（在关键操作后自动保存）
func (m *Manager) saveIfNeeded() {
	if err := m.SaveCodes(); err != nil {
		log.Printf("错误：保存授权码失败：%v\n", err)
	}
}

// GenerateCode 生成授权码
func (m *Manager) GenerateCode(validityType model.ValidityType) (*model.AuthCode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成唯一授权码
	code, err := generateUniqueCode()
	if err != nil {
		return nil, fmt.Errorf("生成授权码失败：%w", err)
	}

	// 计算过期时间
	expiryTime := time.Now().Add(validityType.Duration())

	authCode := &model.AuthCode{
		Code:       code,
		Type:       validityType,
		ExpiryTime: expiryTime,
		IsActive:   false,
	}

	// 保存到数据库
	result := di.DB.Create(authCode)
	if result.Error != nil {
		return nil, fmt.Errorf("保存授权码到数据库失败：%w", result.Error)
	}

	m.codes[code] = authCode

	return authCode, nil
}

// ActivateCode 激活授权码
func (m *Manager) ActivateCode(code string) (*model.AuthCode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	authCode, exists := m.codes[code]
	if !exists {
		return nil, ErrCodeNotFound
	}

	if authCode.IsActive {
		return nil, ErrCodeAlreadyUsed
	}

	if authCode.IsExpired() {
		return nil, ErrCodeExpired
	}

	now := time.Now()
	authCode.IsActive = true
	authCode.ActivatedAt = &now

	// 保存到数据库
	result := di.DB.Save(authCode)
	if result.Error != nil {
		return nil, fmt.Errorf("保存授权码到数据库失败：%w", result.Error)
	}

	return authCode, nil
}

// ValidateCode 验证授权码
func (m *Manager) ValidateCode(code string) (*model.AuthCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	authCode, exists := m.codes[code]
	if !exists {
		return nil, ErrCodeNotFound
	}

	if !authCode.IsActive {
		return nil, ErrInvalidCode
	}

	if authCode.IsExpired() {
		return nil, ErrCodeExpired
	}

	return authCode, nil
}

// GetCode 获取授权码信息
func (m *Manager) GetCode(code string) (*model.AuthCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	authCode, exists := m.codes[code]
	if !exists {
		return nil, ErrCodeNotFound
	}

	return authCode, nil
}

// ListCodes 列出所有授权码
func (m *Manager) ListCodes() []*model.AuthCode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	codes := make([]*model.AuthCode, 0, len(m.codes))
	for _, code := range m.codes {
		codes = append(codes, code)
	}

	return codes
}

// DeleteCode 删除授权码
func (m *Manager) DeleteCode(code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	authCode, exists := m.codes[code]
	if !exists {
		return ErrCodeNotFound
	}

	// 从数据库删除
	result := di.DB.Delete(authCode)
	if result.Error != nil {
		return fmt.Errorf("从数据库删除授权码失败：%w", result.Error)
	}

	delete(m.codes, code)

	return nil
}

// generateUniqueCode 生成唯一的授权码
func generateUniqueCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 生成格式：HR-XXXXXXXX-XXXXXXXX-XXXX
	code := fmt.Sprintf("HR-%s-%s",
		hex.EncodeToString(bytes[:8]),
		hex.EncodeToString(bytes[8:12]))

	return code, nil
}
