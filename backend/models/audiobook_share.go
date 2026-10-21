package models

import "time"

// AudiobookShare 有声书分享（分享的是已生成完成的有声书任务）
type AudiobookShare struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint       `json:"-" gorm:"not null;default:0;index"` // 分享者
	TaskID    uint       `json:"task_id" gorm:"not null;index"`     // 被分享的有声书任务
	Token     string     `json:"token" gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}
