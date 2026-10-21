package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"

	"github.com/gin-gonic/gin"
)

// CreateReadingHistory 记录一次阅读会话（阅读器上报）
func CreateReadingHistory(c *gin.Context) {
	var input struct {
		BookID    uint      `json:"book_id"`
		Type      string    `json:"type"`  // read=阅读, tts=朗读
		Engine    string    `json:"engine"` // 朗读引擎，仅 tts
		Duration  int       `json:"duration"` // 秒
		StartPage int       `json:"start_page"`
		EndPage   int       `json:"end_page"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.BookID == 0 || input.Duration <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "book_id and duration required"})
		return
	}

	// 仅允许 read / tts 两种类型，缺省或非法值归为 read
	recType := input.Type
	if recType != "read" && recType != "tts" {
		recType = "read"
	}

	uid := middleware.CurrentUserID(c)
	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, input.BookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
		return
	}

	// 校验时间字段，缺省则回退（统一转为本地时区，保证按天聚合正确）
	start := input.StartTime
	end := input.EndTime
	if start.IsZero() {
		start = time.Now()
	}
	start = start.Local()
	if end.IsZero() {
		end = start.Add(time.Duration(input.Duration) * time.Second)
	}
	end = end.Local()

	record := models.ReadingHistory{
		UserID:    uid,
		BookID:    input.BookID,
		Type:      recType,
		Engine:    input.Engine,
		StartPage: input.StartPage,
		EndPage:   input.EndPage,
		Duration:  input.Duration,
		StartTime: start,
		EndTime:   &end,
	}
	if err := config.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// GetCalendarStats 按月返回每日阅读汇总
func GetCalendarStats(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year/month"})
		return
	}

	uid := middleware.CurrentUserID(c)

	// 基准每日目标（所有 active 计划中最大值）
	var goalMin int
	config.DB.Model(&models.ReadingPlan{}).Where("user_id = ? AND active = ?", uid, true).
		Select("COALESCE(MAX(daily_goal_min), 0)").Scan(&goalMin)

	startDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDay := startDay.AddDate(0, 1, 0)

	type row struct {
		Day      string
		TotalSec int
		Sessions int
	}
	var rows []row
	config.DB.Raw(`SELECT date(start_time) as day, SUM(duration) as total_sec, COUNT(*) as sessions
		FROM reading_histories
		WHERE user_id = ? AND start_time >= ? AND start_time < ?
		GROUP BY date(start_time)`, uid, startDay, endDay).Scan(&rows)

	dayMap := make(map[string]row)
	for _, r := range rows {
		dayMap[r.Day] = r
	}

	// 生成整月每一天
	days := []gin.H{}
	for d := startDay; d.Before(endDay); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		r, ok := dayMap[key]
		durationSec := 0
		sessions := 0
		if ok {
			durationSec = r.TotalSec
			sessions = r.Sessions
		}
		days = append(days, gin.H{
			"date":       key,
			"duration":   durationSec,
			"sessions":   sessions,
			"completed":  goalMin > 0 && durationSec >= goalMin*60,
			"daily_goal": goalMin,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"year":           year,
		"month":          month,
		"days":           days,
		"daily_goal_min": goalMin,
	})
}

// GetOverviewStats 获取阅读统计总览（含连续天数）
func GetOverviewStats(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var totalSec int64
	var totalSessions int64
	config.DB.Model(&models.ReadingHistory{}).Where("user_id = ?", uid).Select("COALESCE(SUM(duration), 0)").Scan(&totalSec)
	config.DB.Model(&models.ReadingHistory{}).Where("user_id = ?", uid).Count(&totalSessions)

	// 本月时长与达标天数
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	nextMonth := monthStart.AddDate(0, 1, 0)
	var monthSec int64
	config.DB.Model(&models.ReadingHistory{}).
		Where("user_id = ? AND start_time >= ? AND start_time < ?", uid, monthStart, nextMonth).
		Select("COALESCE(SUM(duration), 0)").Scan(&monthSec)

	// 今日时长
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	var todaySec int64
	config.DB.Model(&models.ReadingHistory{}).
		Where("user_id = ? AND start_time >= ? AND start_time < ?", uid, todayStart, todayStart.AddDate(0, 0, 1)).
		Select("COALESCE(SUM(duration), 0)").Scan(&todaySec)

	// 基准每日目标
	var goalMin int
	config.DB.Model(&models.ReadingPlan{}).Where("user_id = ? AND active = ?", uid, true).
		Select("COALESCE(MAX(daily_goal_min), 0)").Scan(&goalMin)

	// 本月达标天数
	var monthGoalDays int64
	if goalMin > 0 {
		config.DB.Raw(`SELECT COUNT(*) FROM (
			SELECT date(start_time) as day, SUM(duration) as total
			FROM reading_histories
			WHERE user_id = ? AND start_time >= ? AND start_time < ?
			GROUP BY date(start_time)
			HAVING total >= ?)`,
			uid, monthStart, nextMonth, goalMin*60).Scan(&monthGoalDays)
	}

	// 连续阅读天数（从今天或昨天开始向前数）
	streak := 0
	cur := todayStart
	if !hasReadingOn(uid, cur) {
		cur = cur.AddDate(0, 0, -1)
	}
	for hasReadingOn(uid, cur) {
		streak++
		cur = cur.AddDate(0, 0, -1)
	}

	// 阅读过的最早日期
	var firstRead string
	config.DB.Raw(`SELECT MIN(date(start_time)) FROM reading_histories WHERE user_id = ?`, uid).Scan(&firstRead)

	// 计划数量
	var planCount int64
	config.DB.Model(&models.ReadingPlan{}).Where("user_id = ? AND active = ?", uid, true).Count(&planCount)

	// 最近 7 天
	type drow struct {
		Day      string
		TotalSec int
	}
	var drows []drow
	config.DB.Raw(`SELECT date(start_time) as day, SUM(duration) as total_sec
		FROM reading_histories
		WHERE user_id = ? AND start_time >= ?
		GROUP BY date(start_time)
		ORDER BY day`, uid, todayStart.AddDate(0, 0, -6)).Scan(&drows)
	recent7 := []gin.H{}
	dMap := make(map[string]int)
	for _, r := range drows {
		dMap[r.Day] = r.TotalSec
	}
	for i := 6; i >= 0; i-- {
		d := todayStart.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		recent7 = append(recent7, gin.H{
			"date":      key,
			"duration":  dMap[key],
			"completed": goalMin > 0 && dMap[key] >= goalMin*60,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_duration":  totalSec,
		"total_sessions":  totalSessions,
		"month_duration":  monthSec,
		"today_duration":  todaySec,
		"current_streak":  streak,
		"month_goal_days": monthGoalDays,
		"daily_goal_min":  goalMin,
		"plan_count":      planCount,
		"first_read":      firstRead,
		"recent_7_days":   recent7,
	})
}

func hasReadingOn(uid uint, day time.Time) bool {
	var count int64
	config.DB.Model(&models.ReadingHistory{}).
		Where("user_id = ? AND start_time >= ? AND start_time < ?", uid, day, day.AddDate(0, 0, 1)).
		Count(&count)
	return count > 0
}

// GetDailyStats 获取某一天的阅读记录详情
func GetDailyStats(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date required (YYYY-MM-DD)"})
		return
	}
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
		return
	}

	uid := middleware.CurrentUserID(c)
	var records []struct {
		models.ReadingHistory
		BookTitle string  `json:"book_title"`
		BookCover string  `json:"book_cover"`
		BookType  string  `json:"book_type"`
		Progress  float64 `json:"progress"`
	}
	config.DB.Raw(`SELECT h.*, b.title as book_title, b.cover_path as book_cover,
		b.file_type as book_type, b.progress as progress
		FROM reading_histories h
		JOIN books b ON b.id = h.book_id
		WHERE h.user_id = ? AND h.start_time >= ? AND h.start_time < ?
		ORDER BY h.start_time DESC`,
		uid, day, day.AddDate(0, 0, 1)).Scan(&records)

	var totalSec int64
	var ttsSec int64
	for _, r := range records {
		totalSec += int64(r.Duration)
		if r.Type == "tts" {
			ttsSec += int64(r.Duration)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"date":      date,
		"total":     totalSec,
		"tts_total": ttsSec,
		"records":   records,
	})
}

// ========== 阅读计划 ==========

// ListPlans 获取全部阅读计划
func ListPlans(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var plans []models.ReadingPlan
	config.DB.Where("user_id = ?", uid).Order("active DESC, id DESC").Find(&plans)

	// 附带每本书信息
	bookIDs := []uint{}
	for _, p := range plans {
		if p.BookID != nil {
			bookIDs = append(bookIDs, *p.BookID)
		}
	}
	books := map[uint]models.Book{}
	if len(bookIDs) > 0 {
		var list []models.Book
		config.DB.Where("id IN ? AND user_id = ?", bookIDs, uid).Find(&list)
		for _, b := range list {
			books[b.ID] = b
		}
	}

	result := []gin.H{}
	for _, p := range plans {
		item := gin.H{
			"id":             p.ID,
			"name":           p.Name,
			"daily_goal_min": p.DailyGoalMin,
			"start_date":     p.StartDate,
			"end_date":       p.EndDate,
			"book_id":        p.BookID,
			"active":         p.Active,
			"created_at":     p.CreatedAt,
		}
		if p.BookID != nil {
			if b, ok := books[*p.BookID]; ok {
				item["book_title"] = b.Title
				item["book_cover"] = b.CoverPath
			}
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// CreatePlan 创建阅读计划
func CreatePlan(c *gin.Context) {
	var input struct {
		Name         string `json:"name"`
		DailyGoalMin int    `json:"daily_goal_min"`
		StartDate    string `json:"start_date"`
		EndDate      string `json:"end_date"`
		BookID       *uint  `json:"book_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Name == "" || input.DailyGoalMin <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and daily_goal_min required"})
		return
	}
	if input.StartDate == "" {
		input.StartDate = time.Now().Format("2006-01-02")
	}
	if input.EndDate == "" {
		input.EndDate = time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	}

	plan := models.ReadingPlan{
		UserID:       middleware.CurrentUserID(c),
		Name:         input.Name,
		DailyGoalMin: input.DailyGoalMin,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
		BookID:       input.BookID,
		Active:       true,
	}
	if err := config.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// UpdatePlan 更新阅读计划
func UpdatePlan(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var plan models.ReadingPlan
	if err := config.DB.Where("user_id = ?", uid).First(&plan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	var input struct {
		Name         *string `json:"name"`
		DailyGoalMin *int    `json:"daily_goal_min"`
		StartDate    *string `json:"start_date"`
		EndDate      *string `json:"end_date"`
		BookID       *uint   `json:"book_id"`
		Active       *bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != nil && *input.Name != "" {
		updates["name"] = *input.Name
	}
	if input.DailyGoalMin != nil && *input.DailyGoalMin > 0 {
		updates["daily_goal_min"] = *input.DailyGoalMin
	}
	if input.StartDate != nil {
		updates["start_date"] = *input.StartDate
	}
	if input.EndDate != nil {
		updates["end_date"] = *input.EndDate
	}
	if input.BookID != nil {
		updates["book_id"] = *input.BookID
	}
	if input.Active != nil {
		updates["active"] = *input.Active
	}
	if len(updates) > 0 {
		config.DB.Model(&plan).Updates(updates)
	}
	c.JSON(http.StatusOK, plan)
}

// DeletePlan 删除阅读计划
func DeletePlan(c *gin.Context) {
	id := c.Param("id")
	uid := middleware.CurrentUserID(c)
	var plan models.ReadingPlan
	if err := config.DB.Where("user_id = ?", uid).First(&plan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	config.DB.Delete(&plan)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// 辅助：格式化时长
func FormatDuration(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("%d秒", sec)
	}
	if sec < 3600 {
		return fmt.Sprintf("%d分钟", sec/60)
	}
	return fmt.Sprintf("%d小时%d分", sec/3600, (sec%3600)/60)
}
