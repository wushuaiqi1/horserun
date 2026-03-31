package biz

import (
	"encoding/json"
	"fmt"
	"time"
)

// ValidityType 授权码有效期类型
type ValidityType int

const (
	Validity3Days   ValidityType = iota + 1 // 3 天
	Validity1Month                          // 1 个月
	Validity3Months                         // 3 个月
	Validity1Year                           // 1 年
)

// String 返回有效期类型的中文描述
func (v ValidityType) String() string {
	switch v {
	case Validity3Days:
		return "3 天"
	case Validity1Month:
		return "1 个月"
	case Validity3Months:
		return "3 个月"
	case Validity1Year:
		return "1 年"
	default:
		return "未知"
	}
}

// Duration 返回有效期类型对应的时长
func (v ValidityType) Duration() time.Duration {
	switch v {
	case Validity3Days:
		return 72 * time.Hour
	case Validity1Month:
		return 720 * time.Hour // 30 天
	case Validity3Months:
		return 2160 * time.Hour // 90 天
	case Validity1Year:
		return 8760 * time.Hour // 365 天
	default:
		return 0
	}
}

// TimeFormat 时间格式化常量
const TimeFormat = "2006-01-02 15:04:05"

// AuthCode 授权码结构
type AuthCode struct {
	ID          uint         `gorm:"primaryKey" json:"id"`             // 主键
	Code        string       `gorm:"uniqueIndex;size:32" json:"code"`  // 授权码
	Type        ValidityType `gorm:"type:integer" json:"type"`         // 有效期类型
	ExpiryTime  time.Time    `json:"expiry_time"`                      // 过期时间
	IsActive    bool         `json:"is_active"`                        // 是否激活
	ActivatedAt *time.Time   `json:"activated_at,omitempty"`           // 激活时间
	CreatedAt   time.Time    `gorm:"autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt   time.Time    `gorm:"autoUpdateTime" json:"updated_at"` // 更新时间
}

// MarshalJSON 自定义 JSON 序列化
func (a *AuthCode) MarshalJSON() ([]byte, error) {
	type Alias AuthCode

	expiryTimeStr := ""
	if !a.ExpiryTime.IsZero() {
		expiryTimeStr = a.ExpiryTime.Format(TimeFormat)
	}

	var activatedAtStr *string
	if a.ActivatedAt != nil && !a.ActivatedAt.IsZero() {
		formatted := a.ActivatedAt.Format(TimeFormat)
		activatedAtStr = &formatted
	}

	return json.Marshal(&struct {
		*Alias
		ExpiryTime  string  `json:"expiry_time"`
		ActivatedAt *string `json:"activated_at,omitempty"`
	}{
		Alias:       (*Alias)(a),
		ExpiryTime:  expiryTimeStr,
		ActivatedAt: activatedAtStr,
	})
}

// UnmarshalJSON 自定义 JSON 反序列化
func (a *AuthCode) UnmarshalJSON(data []byte) error {
	type Alias AuthCode

	aux := &struct {
		ExpiryTime  string  `json:"expiry_time"`
		ActivatedAt *string `json:"activated_at,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.ExpiryTime != "" {
		expiryTime, err := time.Parse(TimeFormat, aux.ExpiryTime)
		if err != nil {
			return fmt.Errorf("解析过期时间失败：%w", err)
		}
		a.ExpiryTime = expiryTime
	}

	if aux.ActivatedAt != nil && *aux.ActivatedAt != "" {
		activatedAt, err := time.Parse(TimeFormat, *aux.ActivatedAt)
		if err != nil {
			return fmt.Errorf("解析激活时间失败：%w", err)
		}
		a.ActivatedAt = &activatedAt
	}

	return nil
}

// IsExpired 检查授权码是否过期
func (a *AuthCode) IsExpired() bool {
	return time.Now().After(a.ExpiryTime)
}

// IsValid 检查授权码是否有效
func (a *AuthCode) IsValid() bool {
	return a.IsActive && !a.IsExpired()
}

// RemainingTime 获取剩余有效时间
func (a *AuthCode) RemainingTime() time.Duration {
	if a.IsExpired() {
		return 0
	}
	return time.Until(a.ExpiryTime)
}
