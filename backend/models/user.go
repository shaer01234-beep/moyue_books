package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string    `json:"username" gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Nickname     string    `json:"nickname" gorm:"size:64"`
	Avatar       string    `json:"avatar" gorm:"size:255"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PublicUser 返回给前端的用户信息（不含密码）
func (u *User) PublicUser() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"username":   u.Username,
		"nickname":   u.Nickname,
		"avatar":     u.Avatar,
		"created_at": u.CreatedAt,
	}
}

// UserScopedModels 所有参与“用户数据隔离”的表：首个用户（注册或网关自动建号）接管旧数据（user_id=0）
var UserScopedModels = []interface{}{
	&Book{},
	&Annotation{},
	&Bookmark{},
	&Collection{},
	&CollectionBook{},
	&ReadingHistory{},
	&ReadingPlan{},
	&Setting{},
	&AudiobookTask{},
}

// AdoptLegacyData 把历史遗留（user_id=0）的数据分配给指定用户。
// 定义在 models 包以便 handlers（注册首个用户）复用。
func AdoptLegacyData(db *gorm.DB, uid uint) {
	for _, m := range UserScopedModels {
		db.Model(m).
			Where("user_id = ? OR user_id IS NULL", 0).
			Update("user_id", uid)
	}
}
