package models

import "time"

// Category 用户自定义分类（支持不选图书创建的空分类，支持父级分类形成层级）
type Category struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint      `json:"-" gorm:"not null;default:0;uniqueIndex:idx_user_category_name"`
	Name      string    `json:"name" gorm:"size:128;not null;uniqueIndex:idx_user_category_name"`
	Parent    string    `json:"parent" gorm:"size:128;default:''"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
