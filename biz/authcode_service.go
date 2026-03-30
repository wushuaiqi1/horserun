package biz

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrCodeNotFound    = errors.New("授权码不存在")
	ErrCodeExpired     = errors.New("授权码已过期")
	ErrCodeAlreadyUsed = errors.New("授权码已被使用")
	ErrInvalidCode     = errors.New("无效的授权码")
)

// Manager 授权码管理器
type Manager struct {
	mu      sync.RWMutex
	codes   map[string]*AuthCode
	dataDir string
}

// NewManager 创建授权码管理器
func NewManager(dataDir string) *Manager {
	m := &Manager{
		codes:   make(map[string]*AuthCode),
		dataDir: dataDir,
	}

	// 初始化数据目录并加载已有数据
	if err := m.initDataDir(); err != nil {
		log.Printf("警告：初始化数据目录失败：%v\n", err)
	}
	if err := m.LoadCodes(); err != nil {
		log.Printf("警告：加载授权码失败：%v\n", err)
	}

	return m
}

// initDataDir 初始化数据目录
func (m *Manager) initDataDir() error {
	if _, err := os.Stat(m.dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(m.dataDir, 0755); err != nil {
			return fmt.Errorf("创建数据目录失败：%w", err)
		}
	}
	return nil
}

// getDataFilePath 获取数据文件路径
func (m *Manager) getDataFilePath() string {
	return filepath.Join(m.dataDir, "authcodes.json")
}

// LoadCodes 从文件加载授权码
func (m *Manager) LoadCodes() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := m.getDataFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在时返回空，不报错
		}
		return fmt.Errorf("读取文件失败：%w", err)
	}

	var codes []*AuthCode
	if err := json.Unmarshal(data, &codes); err != nil {
		return fmt.Errorf("解析 JSON 失败：%w", err)
	}

	// 将加载的授权码存入内存
	for _, code := range codes {
		m.codes[code.Code] = code
	}

	log.Printf("成功加载 %d 个授权码\n", len(codes))
	return nil
}

// SaveCodes 保存授权码到文件
func (m *Manager) SaveCodes() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	codes := make([]*AuthCode, 0, len(m.codes))
	for _, code := range m.codes {
		codes = append(codes, code)
	}

	data, err := json.MarshalIndent(codes, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败：%w", err)
	}

	filePath := m.getDataFilePath()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败：%w", err)
	}

	log.Printf("成功保存 %d 个授权码到 %s\n", len(codes), filePath)
	return nil
}

// saveIfNeeded 检查是否需要保存（在关键操作后自动保存）
func (m *Manager) saveIfNeeded() {
	if err := m.SaveCodes(); err != nil {
		log.Printf("错误：保存授权码失败：%v\n", err)
	}
}

// GenerateCode 生成授权码
func (m *Manager) GenerateCode(validityType ValidityType) (*AuthCode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成唯一授权码
	code, err := generateUniqueCode()
	if err != nil {
		return nil, fmt.Errorf("生成授权码失败：%w", err)
	}

	// 计算过期时间
	expiryTime := time.Now().Add(validityType.Duration())

	authCode := &AuthCode{
		Code:       code,
		Type:       validityType,
		ExpiryTime: expiryTime,
		IsActive:   false,
	}

	m.codes[code] = authCode

	// 保存到文件
	go m.saveIfNeeded()

	return authCode, nil
}

// ActivateCode 激活授权码
func (m *Manager) ActivateCode(code string) (*AuthCode, error) {
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

	// 保存到文件
	go m.saveIfNeeded()

	return authCode, nil
}

// ValidateCode 验证授权码
func (m *Manager) ValidateCode(code string) (*AuthCode, error) {
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
func (m *Manager) GetCode(code string) (*AuthCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	authCode, exists := m.codes[code]
	if !exists {
		return nil, ErrCodeNotFound
	}

	return authCode, nil
}

// ListCodes 列出所有授权码
func (m *Manager) ListCodes() []*AuthCode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	codes := make([]*AuthCode, 0, len(m.codes))
	for _, code := range m.codes {
		codes = append(codes, code)
	}

	return codes
}

// DeleteCode 删除授权码
func (m *Manager) DeleteCode(code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.codes[code]; !exists {
		return ErrCodeNotFound
	}

	delete(m.codes, code)

	// 保存到文件
	go m.saveIfNeeded()

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
