package models

import "time"

// AudiobookTask 有声书生成任务（一本书对应一条记录）
type AudiobookTask struct {
	ID            uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint       `json:"user_id" gorm:"index;not null"`
	BookID        uint       `json:"book_id" gorm:"index;not null"`
	BookTitle     string     `json:"book_title" gorm:"size:512"` // 冗余书名，便于列表展示
	CoverPath     string     `json:"cover_path" gorm:"size:512"` // 封面文件路径（接收的分享任务使用，普通任务为空时回退原书封面）
	Status        string     `json:"status" gorm:"size:32;index"` // pending/processing/done/failed/cancelled/paused
	Engine        string     `json:"engine" gorm:"size:32"`
	Voice         string     `json:"voice" gorm:"size:128"` // Edge TTS 音色 short name
	SpeedFactor   float64    `json:"speed_factor"`
	TextLang      string     `json:"text_lang" gorm:"size:16"`
	TotalSegments int        `json:"total_segments"`
	DoneSegments  int        `json:"done_segments"`
	Error         string     `json:"error" gorm:"size:1024"`
	AudioDir      string     `json:"-" gorm:"-"` // 音频文件目录（运行时计算）
	// 播放进度（用于下次继续播放）
	LastSegment   int        `json:"last_segment" gorm:"default:0"`  // 最后播放段索引（0 起）
	LastPosition  float64    `json:"last_position" gorm:"default:0"` // 段内播放位置（秒）
	LastPlayedAt  *time.Time `json:"last_played_at"`                 // 最后播放时间
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	FinishedAt    *time.Time `json:"finished_at"`
}

// SegmentInfo 有声书分段元数据（存于 segments.json，随生成进度实时刷新）
type SegmentInfo struct {
	Index      int    `json:"index"`   // 1-based
	Title      string `json:"title"`   // 章节标题/页码区间
	File       string `json:"file"`    // 音频文件名（segment_0001.mp3）
	Text       string `json:"text"`    // 文本摘要（前 120 字），用于展示
	CharCount  int    `json:"char_count"`
	Duration   int    `json:"duration"` // 估算时长（秒）
	Done       bool   `json:"done"`     // 是否已生成
	ChapterIdx int    `json:"chapter_idx"`
}
