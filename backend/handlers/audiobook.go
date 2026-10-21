package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"my-books/backend/config"
	"my-books/backend/middleware"
	"my-books/backend/models"
	"my-books/backend/services"
	"my-books/backend/utils"

	"github.com/gin-gonic/gin"
)

// audiobookSupportedTypes 支持生成有声书的格式
var audiobookSupportedTypes = map[string]bool{
	"epub": true, "kepub": true, "azw3": true, "mobi": true,
	"pdf": true, "md": true, "txt": true,
}

// ListAudiobooks 获取当前用户全部有声书任务（含图书信息），按最后播放时间倒序（未播放过的按更新时间排后）
// GET /api/audiobooks
func ListAudiobooks(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var tasks []models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).
		Order("last_played_at IS NULL, last_played_at DESC, updated_at DESC").
		Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	bookIDs := make([]uint, 0, len(tasks))
	for _, t := range tasks {
		bookIDs = append(bookIDs, t.BookID)
	}
	books := map[uint]models.Book{}
	if len(bookIDs) > 0 {
		var list []models.Book
		// 仅关联当前用户自己的书；接收的分享任务指向分享者的书，查不到则 book 为空
		config.DB.Where("user_id = ? AND id IN ?", uid, bookIDs).Find(&list)
		for _, b := range list {
			books[b.ID] = b
		}
	}

	// 兜底：任务自身无封面且原书不属于当前用户时（如接收的分享任务），
	// 从全库按 id 查封面路径，仅内存填充用于展示，不落库
	missingIDs := make([]uint, 0, len(tasks))
	for _, t := range tasks {
		if b, ok := books[t.BookID]; !ok || b.CoverPath == "" {
			missingIDs = append(missingIDs, t.BookID)
		}
	}
	globalCovers := map[uint]string{}
	if len(missingIDs) > 0 {
		var global []models.Book
		config.DB.Where("id IN ?", missingIDs).Select("id", "cover_path").Find(&global)
		for _, b := range global {
			if b.CoverPath != "" {
				globalCovers[b.ID] = b.CoverPath
			}
		}
	}

	type item struct {
		models.AudiobookTask
		Book *models.Book `json:"book"`
	}
	items := make([]item, 0, len(tasks))
	for _, t := range tasks {
		b := books[t.BookID]
		// 内存填充封面路径：任务自身 > 当前用户原书 > 全库原书（接收的分享任务）
		if t.CoverPath == "" {
			if b.CoverPath != "" {
				t.CoverPath = b.CoverPath
			} else if p, ok := globalCovers[t.BookID]; ok {
				t.CoverPath = p
			}
		}
		items = append(items, item{AudiobookTask: t, Book: &b})
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// CreateAudiobook 创建有声书生成任务（异步生成）
// POST /api/audiobooks  body: {book_id, engine?, voice?, speed_factor?, text_lang?}
func CreateAudiobook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	var req struct {
		BookID      uint    `json:"book_id" binding:"required"`
		Engine      string  `json:"engine"`
		Voice       string  `json:"voice"`
		SpeedFactor float64 `json:"speed_factor"`
		TextLang    string  `json:"text_lang"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}

	var book models.Book
	if err := config.DB.Where("user_id = ?", uid).First(&book, req.BookID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图书不存在"})
		return
	}
	ft := strings.ToLower(book.FileType)
	if !audiobookSupportedTypes[ft] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该格式暂不支持生成有声书（支持 epub/azw3/pdf/md/txt）"})
		return
	}

	// 音色与语速缺省值（与 TTS 朗读保持一致）
	cfg := services.GetTTSConfig(uid)
	engine := req.Engine
	if engine == "" {
		engine = cfg.Engine
	}
	if engine == "" {
		engine = services.EngineEdgeTTS
	}
	voice := req.Voice
	if voice == "" {
		voice = cfg.Voice
	}
	speed := req.SpeedFactor
	if speed <= 0 {
		speed = cfg.SpeedFactor
	}
	if speed <= 0 {
		speed = 1.0
	}
	textLang := req.TextLang
	if textLang == "" {
		textLang = cfg.TextLang
	}
	if textLang == "" {
		textLang = "zh"
	}

	dir := services.AudiobookDirFor(uid, req.BookID)
	now := time.Now()

	// 同一本书同时只允许一个进行中的任务（含已暂停）
	var active models.AudiobookTask
	if err := config.DB.Where("user_id = ? AND book_id = ? AND status IN ?",
		uid, req.BookID, []string{"pending", "processing", "paused"}).First(&active).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该图书已有生成任务进行中", "task": active})
		return
	}

	var task models.AudiobookTask
	err := config.DB.Where("user_id = ? AND book_id = ?", uid, req.BookID).
		Order("id DESC").First(&task).Error
	reuse := err == nil
	if reuse {
		// 参数与旧任务不一致时全量重新生成（先清空旧音频，避免新旧音色混杂）
		if task.Voice != voice || task.Engine != engine ||
			task.SpeedFactor != speed || task.TextLang != textLang {
			_ = services.DeleteAudiobookFiles(uid, req.BookID)
			task.Engine = engine
			task.Voice = voice
			task.SpeedFactor = speed
			task.TextLang = textLang
		}
		task.Status = "pending"
		task.Error = ""
		task.TotalSegments = 0
		task.DoneSegments = 0
		task.FinishedAt = nil
		task.UpdatedAt = now
		config.DB.Save(&task)
	} else {
		task = models.AudiobookTask{
			UserID: uid, BookID: req.BookID, BookTitle: book.Title,
			Engine: engine, Voice: voice, SpeedFactor: speed, TextLang: textLang,
			Status: "pending",
		}
		if err := config.DB.Create(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败"})
			return
		}
	}

	// 同步构建分段元数据：前端能立即拿到总段数（PDF 提取全文可能耗时数秒）
	segs := services.LoadAudiobookSegments(dir)
	if len(segs) == 0 {
		segs, err = services.BuildAudiobookSegments(&book, speed)
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			task.FinishedAt = &now
			config.DB.Save(&task)
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "无法生成有声书：" + err.Error()})
			return
		}
		if err := services.SaveAudiobookSegments(dir, segs); err != nil {
			task.Status = "failed"
			task.Error = "分段元数据写入失败"
			task.FinishedAt = &now
			config.DB.Save(&task)
			c.JSON(http.StatusInternalServerError, gin.H{"error": task.Error})
			return
		}
	}
	task.TotalSegments = len(segs)
	// 复用任务时恢复已完成段数（断点续传：元数据标记 done 且音频文件存在的段）
	doneCount := 0
	for i := range segs {
		if segs[i].Done {
			if _, err := os.Stat(filepath.Join(dir, segs[i].File)); err == nil {
				doneCount++
			}
		}
	}
	task.DoneSegments = doneCount
	config.DB.Model(&task).Updates(map[string]interface{}{
		"total_segments": len(segs), "done_segments": doneCount,
	})

	// 异步启动服务端合成
	services.StartAudiobookGeneration(task.ID)

	c.JSON(http.StatusOK, gin.H{"data": task})
}

// GetAudiobook 获取任务详情与分段列表
// GET /api/audiobooks/:id
func GetAudiobook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	segs := services.LoadAudiobookSegments(services.AudiobookDirFor(uid, task.BookID))
	if segs == nil {
		segs = []models.SegmentInfo{}
	}

	// 兜底填充封面：任务自身 > 原书封面（全库，兼容接收的分享任务）
	if task.CoverPath == "" {
		var book models.Book
		if err := config.DB.First(&book, task.BookID).Error; err == nil && book.CoverPath != "" {
			task.CoverPath = book.CoverPath
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"task": task, "segments": segs}})
}

// GetAudiobookCover 获取有声书封面（优先任务自身封面，接收的分享任务无原书记录时可用）
// GET /api/audiobooks/:id/cover
func GetAudiobookCover(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	cover := task.CoverPath
	if cover == "" {
		// 回退到原书封面（普通任务）
		var book models.Book
		if err := config.DB.First(&book, task.BookID).Error; err != nil || book.CoverPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "no cover"})
			return
		}
		cover = book.CoverPath
	}

	data, err := os.ReadFile(utils.ResolvePath(cover))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cover not found"})
		return
	}
	contentType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(cover), ".png") {
		contentType = "image/png"
	}
	c.Data(http.StatusOK, contentType, data)
}

// CancelAudiobook 取消生成任务（保留已生成的分段，可续传）
// POST /api/audiobooks/:id/cancel
func CancelAudiobook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if task.Status == "pending" || task.Status == "processing" {
		services.RequestAudiobookCancel(task.ID)
		c.JSON(http.StatusOK, gin.H{"data": "已请求取消"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "任务不在生成中"})
}

// PauseAudiobook 暂停生成任务（保留已生成分段，可恢复）
// POST /api/audiobooks/:id/pause
func PauseAudiobook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if task.Status == "pending" || task.Status == "processing" {
		services.RequestAudiobookPause(task.ID)
		c.JSON(http.StatusOK, gin.H{"data": "已请求暂停"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "任务不在生成中"})
}

// ResumeAudiobook 恢复已暂停的生成任务（续传未完成分段）
// POST /api/audiobooks/:id/resume
func ResumeAudiobook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if task.Status != "paused" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务不在暂停状态"})
		return
	}
	if err := services.ResumeAudiobookTask(task.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "已恢复生成"})
}

// UpdateAudiobookProgress 保存有声书播放进度（用于下次继续播放）
// PUT /api/audiobooks/:id/progress  body: {segment, position}
func UpdateAudiobookProgress(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	var input struct {
		Segment  int     `json:"segment"`
		Position float64 `json:"position"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if input.Segment < 0 {
		input.Segment = 0
	}
	if input.Position < 0 {
		input.Position = 0
	}
	now := time.Now()
	config.DB.Model(&task).Updates(map[string]interface{}{
		"last_segment":   input.Segment,
		"last_position":  input.Position,
		"last_played_at": now,
	})
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// DeleteAudiobook 取消任务并删除全部音频文件与记录
// DELETE /api/audiobooks/:id
func DeleteAudiobook(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if task.Status == "pending" || task.Status == "processing" {
		services.RequestAudiobookCancel(task.ID)
	}
	_ = services.DeleteAudiobookFiles(uid, task.BookID)
	config.DB.Delete(&task)
	c.JSON(http.StatusOK, gin.H{"data": "已删除"})
}

// ServeAudiobookAudio 流式返回分段音频（支持 Range，浏览器可拖动进度条）
// GET /api/audiobooks/:id/audio/:file
func ServeAudiobookAudio(c *gin.Context) {
	uid := middleware.CurrentUserID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var task models.AudiobookTask
	if err := config.DB.Where("user_id = ?", uid).First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	file := c.Param("file")
	if file == "" || strings.Contains(file, "..") || strings.ContainsAny(file, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法文件名"})
		return
	}
	dir := services.AudiobookDirFor(uid, task.BookID)
	audioPath := filepath.Join(dir, filepath.Base(file))
	f, err := os.Open(audioPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "音频不存在"})
		return
	}
	// MediaRecorder 产出格式因浏览器而异（webm/ogg/mp4），按扩展名显式声明类型，
	// 避免 Go mime 表把 .webm 识别为 video/webm 导致 <audio> 拒绝播放。
	// http.ServeContent 支持 Range（进度条拖动/断点续传），且不会覆盖已设置的 Content-Type。
	c.Header("Content-Type", audiobookAudioContentType(file))
	http.ServeContent(c.Writer, c.Request, filepath.Base(file), fi.ModTime(), f)
}

// audiobookAudioContentType 按扩展名返回音频 Content-Type
func audiobookAudioContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".webm":
		return "audio/webm"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".mp4", ".m4a":
		return "audio/mp4"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".aac":
		return "audio/aac"
	default:
		return "application/octet-stream"
	}
}
