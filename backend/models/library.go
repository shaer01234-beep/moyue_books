package models

import "time"

// Library 媒体库：用户指定一个目录，系统监听其中电子书文件的变动并自动重新扫描入库
// （不迁移文件）。文件新增 / 修改 / 删除 / 移动后，经过防抖自动重新扫描。
type Library struct {
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint   `json:"-" gorm:"not null;default:0;index"`
	Name      string `json:"name" gorm:"size:256;not null"`  // 媒体库名称
	Path      string `json:"path" gorm:"size:1024;not null"` // 监听目录（绝对路径）
	Recursive bool   `json:"recursive" gorm:"default:true"`  // 是否递归子目录
	AutoScan  bool   `json:"auto_scan" gorm:"default:true"`  // 是否监听目录变动并自动重新扫描
	Category  string `json:"category" gorm:"size:128"`       // 入库图书归属的分类（空表示不指定）
	LastScanAt   *time.Time `json:"last_scan_at"`                      // 上次扫描时间
	LastResult   string     `json:"last_result" gorm:"type:text"`      // 上次扫描结果 JSON 摘要
	Status       string     `json:"status" gorm:"size:32;default:'idle'"` // idle / scanning / error
	BookCount    int        `json:"book_count"`                        // 当前入库图书数
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// LibraryScanLog 媒体库扫描记录：每次扫描（手动或定时）都会留痕，记录新增与删除的文件。
type LibraryScanLog struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     uint      `json:"-" gorm:"not null;default:0;index"`
	LibraryID  uint      `json:"library_id" gorm:"not null;index"`
	ScanAt     time.Time `json:"scan_at"` // 扫描时间
	TotalFound int       `json:"total_found"`   // 本次扫描发现的文件总数
	Added      int       `json:"added"`         // 新增入库数量
	Updated    int       `json:"updated"`       // 内容变更数量
	Deleted    int       `json:"deleted"`       // 源文件消失（标记为缺失）数量
	Detail     string    `json:"detail" gorm:"type:text"` // JSON：{added:[], updated:[], deleted:[]}
}
