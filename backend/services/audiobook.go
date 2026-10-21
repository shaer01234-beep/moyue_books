package services

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"my-books/backend/config"
	"my-books/backend/models"
	"my-books/backend/utils"

	"github.com/ledongthuc/pdf"
)

// ============ 常量与目录 ============

const (
	// audiobookMaxCharsBase 基准语速（1.0x）下每段音频文本的最大字符数。
	// Edge TTS 单次合成上限为 10 分钟音频（600000ms）：按中文约 4.5 字/秒、
	// 目标时长 9 分钟（对硬上限留约 10% 余量）估算 = 2430 字符；
	// 其余引擎经 HTTP 转发（180s 超时），2430 字符也可在超时内完成合成。
	audiobookMaxCharsBase = 2430
	// audiobookMaxCharsCap 每段最大字符数上限（语速较快时不再放大，
	// 避免单段音频过长影响 HTTP 转发超时）
	audiobookMaxCharsCap = 3000
	// audiobookMinMaxChars 每段最小字符数下限（语速过慢时防止切得过于零碎）
	audiobookMinMaxChars = 800
	// audiobookMaxSegments 单本有声书最大分段数（防止超长书籍导致任务失控）
	audiobookMaxSegments = 3000
	// audiobookMaxConsecutiveFails 连续失败段数超过该值则中止任务
	audiobookMaxConsecutiveFails = 3
	// audiobookRetryTimes 单段合成失败后的最大重试次数（首次尝试不计，单段最多
	// 共尝试 audiobookRetryTimes+1 次）。Edge TTS 及 HTTP 转发的失败多为瞬时
	// 网络抖动或上游限流，失败后重试可显著降低偶发失败率。
	audiobookRetryTimes = 3
	// audiobookRetryBackoffBase 重试基础退避间隔：每次失败翻倍；
	// 命中限流/过载类错误时再放大 audiobookRetryBackoffRateLimit 倍等待。
	audiobookRetryBackoffBase      = time.Second
	audiobookRetryBackoffRateLimit = 4
	// audiobookRecoverGrace 降级后持续稳定该时长（期间未再命中限流强信号）即试探
	// 恢复一级并发；恢复动作同样重置计时，使并发逐级回升、每级再观察一个稳定窗口。
	audiobookRecoverGrace = 60 * time.Second
)

// AudiobookMaxCharsFor 根据朗读语速计算每段音频文本的最大字符数。
// Edge TTS 单次合成有 600s 时长硬上限：语速越慢，同等字符的朗读时间越长，
// 固定 3000 字符在慢速下必然超限。按中文基准 4.5 字/秒、目标时长 540s
// （9 分钟，对上限留 10% 余量）折算：maxChars = 2430 × speed。
func AudiobookMaxCharsFor(speed float64) int {
	if speed <= 0 {
		speed = 1.0
	}
	n := int(float64(audiobookMaxCharsBase) * speed)
	if n > audiobookMaxCharsCap {
		return audiobookMaxCharsCap
	}
	if n < audiobookMinMaxChars {
		return audiobookMinMaxChars
	}
	return n
}

// isEdgeTTSTooLongError 判断错误是否为 Edge TTS 单次合成音频超长（10 分钟上限）
func isEdgeTTSTooLongError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "1007") ||
		strings.Contains(msg, "600000") ||
		strings.Contains(msg, "最大媒体时长")
}

// isTTSServerThrottleError 判断合成失败是否为"服务端明确限流/过载/断连"类错误
// （强信号）。Edge TTS 为微软共享在线服务，并发连接过高时会被显式限流或直接
// 掐断 WS 连接；本地引擎经 HTTP 转发过载时会返回 429/502/503/504，或拒绝/
// 重置连接。命中此类错误说明上游确实撑不住当前并发，调用方应降低并发并拉长
// 退避后重试。
func isTTSServerThrottleError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// 服务端显式限流/过载类状态码与文案
	for _, k := range []string{
		"too many request", "rate limit", "ratelimit", "throttl",
		"quota", "busy", "overload", "unavailable", "限流",
		"429", "502", "503", "504",
	} {
		if strings.Contains(msg, k) {
			return true
		}
	}
	// 服务端断连/拒绝类：微软对过高并发常直接断开/重置 WS 连接（websocket close、
	// connection reset/broken pipe），本地服务过载时也会拒绝建连（connection
	// refused）或会话被掐断后无音频产出（no audio received）
	for _, k := range []string{
		"connection reset", "connection refused", "broken pipe",
		"no audio received", "websocket: close", "bad handshake",
	} {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

// isTTSTimeoutError 判断合成失败是否为"客户端侧超时/网络抖动"类错误（弱信号）。
// 例如连接建立超时、读超时、流提前 EOF 等。这类错误多为本地网络瞬时问题，而非
// 上游过载——Edge 直连路径的合成是边生成边推流，连接层从不因"合成耗时长"而
// 超时，故报超时几乎都发生在建连阶段，与文本长短无关。命中时只做普通退避重试，
// 不降并发（避免把正常网络抖动误判成上游限流而过度降级）。
func isTTSTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, k := range []string{
		"context deadline exceeded", "deadline exceeded",
		"i/o timeout", "timed out", "timeout", "超时", "eof",
	} {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

// audiobookLimiter 单任务内"实际同时执行的合成请求数"限流器。
// 上限初始为 audiobookSegmentWorkers()；命中限流/过载类错误时调用 reduce()
// 将上限折半（最低 1），使后续段自动排队、降低对上游的请求并发压力；待持续
// 稳定 audiobookRecoverGrace 后由 maybeRecover() 逐级回升至初始上限。
type audiobookLimiter struct {
	mu           sync.Mutex
	cond         *sync.Cond
	active       int       // 当前执行中的合成请求数
	limit        int       // 当前并发上限（限流降级后逐级回升，封顶 baseLimit）
	baseLimit    int       // 初始并发上限，也是自适应回升的上限
	lastThrottle time.Time // 最近一次命中限流强信号（降级/恢复）的时间
}

func newAudiobookLimiter(limit int) *audiobookLimiter {
	l := &audiobookLimiter{limit: limit, baseLimit: limit}
	l.cond = sync.NewCond(&l.mu)
	return l
}

// acquire 占用一个并发名额；活跃请求数达到上限时阻塞等待释放。
func (l *audiobookLimiter) acquire() {
	l.mu.Lock()
	for l.active >= l.limit {
		l.cond.Wait()
	}
	l.active++
	l.mu.Unlock()
}

// release 释放一个并发名额并唤醒一个等待者。
func (l *audiobookLimiter) release() {
	l.mu.Lock()
	if l.active > 0 {
		l.active--
	}
	l.cond.Signal()
	l.mu.Unlock()
}

// reduce 将并发上限折半（最低 1），返回调整后的上限，用于限流降级。
// 同时记录降级时刻：maybeRecover 只有在持续稳定 audiobookRecoverGrace
// 未再降级后才会尝试回升。
func (l *audiobookLimiter) reduce() int {
	l.mu.Lock()
	if l.limit > 1 {
		l.limit /= 2
	}
	l.lastThrottle = time.Now()
	n := l.limit
	l.mu.Unlock()
	return n
}

// maybeRecover 降级后自适应回升并发：若距上次降级/恢复已持续稳定满
// audiobookRecoverGrace 且未达初始上限，则试探性恢复一级（折半的逆向 ×2，
// 封顶 baseLimit）。恢复动作同时刷新计时，使回升逐级进行、每级之间再观察一个
// 稳定窗口；若回升后再次命中限流，reduce() 会立即重新降级，不会并发震荡。
// 返回当前并发上限与是否发生了恢复。
func (l *audiobookLimiter) maybeRecover() (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.limit >= l.baseLimit {
		return l.limit, false
	}
	if time.Since(l.lastThrottle) < audiobookRecoverGrace {
		return l.limit, false
	}
	if l.limit*2 >= l.baseLimit {
		l.limit = l.baseLimit
	} else {
		l.limit *= 2
	}
	l.lastThrottle = time.Now()
	return l.limit, true
}

// sleepInterruptible 退避期间分段睡眠，每 200ms 检查一次取消/暂停标记；
// 任务被暂停/取消时提前返回 true（当前段退出，不计失败，续传时自动重做）。
func sleepInterruptible(taskID uint, d time.Duration) bool {
	const step = 200 * time.Millisecond
	for waited := time.Duration(0); waited < d; waited += step {
		if isAudiobookCancelled(taskID) || isAudiobookPaused(taskID) {
			return true
		}
		time.Sleep(step)
	}
	return false
}

// synthesizeSplitHalf 将文本在中间附近句子边界处拆成两半，分别合成后拼接。
// 作为 Edge TTS 单次合成时长超限的兜底重试（分段已按语速压缩，此路径极少触发）。
func synthesizeSplitHalf(cfg TTSConfig, text string) ([]byte, error) {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) < 200 {
		return nil, fmt.Errorf("文本过短，无法拆半重试")
	}
	half := len(runes) / 2
	cut := half
	for i := half; i < len(runes) && i-half <= 40; i++ {
		switch runes[i-1] {
		case '。', '！', '？', '；', '…', '：', '”', '’', '"', '!', '?':
			cut = i
		}
	}
	p1 := strings.TrimSpace(string(runes[:cut]))
	p2 := strings.TrimSpace(string(runes[cut:]))
	if p1 == "" || p2 == "" {
		return nil, fmt.Errorf("拆半后存在空片段")
	}
	d1, _, err := SynthesizeWithConfig(cfg, p1, "", cfg.SpeedFactor, cfg.TextLang, "mp3")
	if err != nil {
		return nil, err
	}
	d2, _, err := SynthesizeWithConfig(cfg, p2, "", cfg.SpeedFactor, cfg.TextLang, "mp3")
	if err != nil {
		return nil, err
	}
	// MP3 为帧流格式，可直接二进制拼接（播放器按帧解析）
	return append(d1, d2...), nil
}

// audiobookMaxRunning 同时执行的有声书生成任务数。
// 环境变量 AUDIOBOOK_MAX_RUNNING（1~4），默认 1：单个任务内已可并行合成多个段，
// 一般无需多任务并行；如需同时生成多本可调大（注意 Edge TTS 为共享在线服务，可能限流）。
func audiobookMaxRunning() int {
	n, _ := strconv.Atoi(os.Getenv("AUDIOBOOK_MAX_RUNNING"))
	if n <= 0 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return n
}

// audiobookSem 全局信号量：限制同时执行的有声书生成任务数。
// 默认 1 个任务；任务内部按 audiobookSegmentWorkers 并行合成多个段。
var audiobookSem = make(chan struct{}, audiobookMaxRunning())

// audiobookSegmentWorkers 单个任务内并行合成的段数。
// Edge TTS 为网络 IO 型，多路并发 WS 可大幅提升吞吐；微软对单 IP 并发连接
// 有一定容忍度，默认 4 路。环境变量 AUDIOBOOK_CONCURRENCY（1~16）可调整。
func audiobookSegmentWorkers() int {
	n, _ := strconv.Atoi(os.Getenv("AUDIOBOOK_CONCURRENCY"))
	if n <= 0 {
		n = 4
	}
	if n > 16 {
		n = 16
	}
	return n
}

// audiobookCancelMap 内存取消/暂停标记（CancelRequest 不落库，跨 goroutine 用 map 传递）
// 值为 "cancel" 或 "pause"，由生成协程轮询生效
var audiobookCancelMap sync.Map

// RequestAudiobookCancel 请求取消指定任务（异步生效）
func RequestAudiobookCancel(taskID uint) {
	audiobookCancelMap.Store(taskID, "cancel")
}

// RequestAudiobookPause 请求暂停指定任务（异步生效，已生成分段保留，可恢复）
func RequestAudiobookPause(taskID uint) {
	audiobookCancelMap.Store(taskID, "pause")
}

func clearAudiobookCancel(taskID uint) {
	audiobookCancelMap.Delete(taskID)
}

func isAudiobookCancelled(taskID uint) bool {
	v, ok := audiobookCancelMap.Load(taskID)
	return ok && v == "cancel"
}

func isAudiobookPaused(taskID uint) bool {
	v, ok := audiobookCancelMap.Load(taskID)
	return ok && v == "pause"
}

// AudiobookRoot 有声书根目录（数据目录/audiobooks）
func AudiobookRoot() string {
	return filepath.Join(config.DataDir(), "audiobooks")
}

// AudiobookDirFor 某本书的音频目录（按用户隔离）
func AudiobookDirFor(uid, bookID uint) string {
	return filepath.Join(AudiobookRoot(), strconv.FormatUint(uint64(uid), 10), strconv.FormatUint(uint64(bookID), 10))
}

func audiobookSegmentsPath(dir string) string { return filepath.Join(dir, "segments.json") }

// ============ 文本提取与分段 ============

var (
	abHTMLBlockRe = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(?:script|style)>`)
	abHTMLTagRe   = regexp.MustCompile(`<[^>]+>`)
	abHTMLSpaceRe = regexp.MustCompile(`[ \t\r\f\v]+`)
	abHTMLEmptyRe = regexp.MustCompile(`\n{3,}`)
	abTitleRe     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
)

// htmlToText 将 EPUB/AZW3 章节 HTML 转纯文本（与朗读接口逻辑一致，保留段落结构）
func htmlToText(content []byte) string {
	text := string(content)
	text = abHTMLBlockRe.ReplaceAllString(text, "\n")
	text = abHTMLTagRe.ReplaceAllString(text, "\n")
	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = abHTMLSpaceRe.ReplaceAllString(text, " ")

	lines := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "。") &&
			!strings.HasSuffix(lines[len(lines)-1], "！") &&
			!strings.HasSuffix(lines[len(lines)-1], "？") {
			lines[len(lines)-1] += line
		} else {
			lines = append(lines, line)
		}
	}
	text = strings.Join(lines, "\n")
	return abHTMLEmptyRe.ReplaceAllString(text, "\n")
}

// pdfPageText 顺序提取 PDF 各页纯文本（用于生成音频）
func pdfPageText(filePath string) ([]string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	np := r.NumPage()
	if np <= 0 {
		return nil, fmt.Errorf("PDF 无页面内容")
	}
	pages := make([]string, np+1) // 1-based
	for pg := 1; pg <= np; pg++ {
		p := r.Page(pg)
		if p.V.IsNull() {
			continue
		}
		t, _ := p.GetPlainText(nil)
		pages[pg] = strings.TrimSpace(t)
	}
	return pages, nil
}

// splitTextChunks 按段落累积切分：优先保留段落完整性，多个小段落合并至接近
// maxChars 再切分；单个超长段落内部按句子边界断句（回看最多 60 字符）。
func splitTextChunks(text string, maxChars int) []string {
	var chunks []string
	buf := strings.Builder{}
	bufLen := 0
	flush := func() {
		if bufLen == 0 {
			return
		}
		chunks = append(chunks, strings.TrimSpace(buf.String()))
		buf.Reset()
		bufLen = 0
	}

	for _, p := range strings.Split(text, "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		pl := len([]rune(p))
		// 已累积段落 + 换行 + 新段落 将超出上限：先切出当前累积
		if bufLen > 0 && bufLen+1+pl > maxChars {
			flush()
		}
		if pl > maxChars {
			// 单个超长段落：内部按句子边界切分
			flush()
			for _, part := range splitLongText(p, maxChars) {
				chunks = append(chunks, part)
			}
			continue
		}
		if bufLen > 0 {
			buf.WriteString("\n")
			bufLen++
		}
		buf.WriteString(p)
		bufLen += pl
	}
	flush()
	return chunks
}

// splitLongText 将超长文本按句子边界切成不超过 maxChars 的片段（回看最多 60 字符）
func splitLongText(text string, maxChars int) []string {
	runes := []rune(text)
	var parts []string
	for len(runes) > 0 {
		if len(runes) <= maxChars {
			parts = append(parts, strings.TrimSpace(string(runes)))
			break
		}
		cut := maxChars
	cutLoop:
		for i := maxChars; i > 0 && maxChars-i <= 60; i-- {
			switch runes[i-1] {
			case '。', '！', '？', '；', '…', '：', '”', '’', '"', '!', '?':
				cut = i
				break cutLoop
			}
		}
		parts = append(parts, strings.TrimSpace(string(runes[:cut])))
		runes = runes[cut:]
	}
	return parts
}

// isTOCHeading 标题是否为目录类标题（"目录"、"Contents"、"Table of Contents" 等）。
// 判定时先合并空白（含全角空格），再做关键词比较。
func isTOCHeading(title string) bool {
	t := strings.Join(strings.Fields(strings.TrimSpace(title)), " ")
	low := strings.ToLower(t)
	switch low {
	case "目录", "contents", "table of contents":
		return true
	}
	if strings.HasPrefix(low, "目录") {
		return len([]rune(t)) <= 8 // 目录页/目录一览 等简短变体
	}
	return false
}

// looksLikeTOCList 内容是否为"目录清单"样式：多行短行、几乎无句读标点
// （EPUB 目录章标题常为空或为"第 N 章"，需靠正文判定；PDF 的目录页同理）。
func looksLikeTOCList(text string) bool {
	runes := []rune(text)
	if len(runes) < 8 || len(runes) > 4000 {
		return false
	}
	lines := 0
	for _, ln := range strings.Split(text, "\n") {
		s := strings.TrimSpace(ln)
		if s == "" {
			continue
		}
		// 允许 markdown 链接/列表前缀（- 1. * 等），但整体仍须为短行
		r := []rune(s)
		if len(r) > 34 {
			return false
		}
		if strings.ContainsAny(s, "。！？!?…；") {
			return false // 含完整句读的正文段落，非目录清单
		}
		lines++
	}
	return lines >= 4
}

// isTOCSection 判断一个章节/页面是否为目录（生成有声书时跳过，不朗读）。
func isTOCSection(title, text string) bool {
	if isTOCHeading(title) {
		return true
	}
	return looksLikeTOCList(text)
}

// BuildAudiobookSegments 从图书提取全文并分段（生成音频前调用）。
// speed 为朗读语速系数，用于按 Edge TTS 单次合成时长上限动态压缩单段字符数；
// 返回的分段包含标题、文本与字符数；音频文件字段按段序号命名。
func BuildAudiobookSegments(book *models.Book, speed float64) ([]models.SegmentInfo, error) {
	filePath := utils.ResolvePath(book.FilePath)
	ft := strings.ToLower(book.FileType)
	maxChars := AudiobookMaxCharsFor(speed)
	var segs []models.SegmentInfo

	appendChunks := func(title string, chapterIdx int, chunks []string) {
		for i, c := range chunks {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if len(segs) >= audiobookMaxSegments {
				return
			}
			suffix := ""
			if len(chunks) > 1 {
				suffix = " · " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(chunks))
			}
			segs = append(segs, models.SegmentInfo{
				Index:      len(segs) + 1,
				Title:      title + suffix,
				File:       fmt.Sprintf("segment_%04d.mp3", len(segs)+1),
				Text:       c, // 完整文本，TTS 合成直接使用；前端展示时自行截断
				CharCount:  len([]rune(c)),
				ChapterIdx: chapterIdx,
			})
		}
	}

	switch ft {
	case "epub", "kepub", "azw3", "mobi":
		var reader interface {
			GetContent(int) ([]byte, error)
			Close()
		}
		switch ft {
		case "azw3", "mobi":
			az, err := OpenAZW3(filePath)
			if err != nil {
				return nil, err
			}
			defer az.Close()
			reader = az
		default:
			ep, err := OpenEPUB(filePath)
			if err != nil {
				return nil, err
			}
			defer ep.Close()
			reader = ep
		}
		// 用目录标题命名分段；无目录时退回"章节 N"
		toc, _ := ExtractTOC(filePath, ft)
		titleBy := map[int]string{}
		if len(toc) > 0 {
			for _, it := range toc {
				if it.Title != "" {
					titleBy[it.Index] = it.Title
				}
			}
		}
		chapter := 0
		for len(segs) < audiobookMaxSegments {
			content, err := reader.GetContent(chapter)
			if err != nil {
				break
			}
			text := htmlToText(content)
			if t := strings.TrimSpace(text); t != "" {
				title := titleBy[chapter]
				if title == "" {
					title = fmt.Sprintf("第 %d 章", chapter+1)
				}
				// 跳过目录页/目录章节（无正文价值），不生成语音
				if isTOCSection(title, text) {
					log.Printf("audiobook: 跳过目录章节 第 %d 章（%s）", chapter+1, title)
					chapter++
					continue
				}
				appendChunks(title, chapter, splitTextChunks(text, maxChars))
			}
			chapter++
		}

	case "pdf":
		pages, err := pdfPageText(filePath)
		if err != nil {
			return nil, err
		}
		np := len(pages) - 1
		if np <= 0 {
			return nil, fmt.Errorf("PDF 无页面内容")
		}
		// 剔除目录清单页（多行短行、无句读），不生成语音
		skippedTOC := 0
		for pg := 1; pg <= np; pg++ {
			if looksLikeTOCList(pages[pg]) {
				pages[pg] = ""
				skippedTOC++
			}
		}
		if skippedTOC > 0 {
			log.Printf("audiobook: PDF 跳过 %d 个目录页", skippedTOC)
		}
		// 目录页码 → 分段标题（段起始页命中目录条目时使用章节标题）
		toc, _ := ExtractTOC(filePath, "pdf")
		titleByPage := map[int]string{}
		if len(toc) > 0 {
			for _, it := range toc {
				if it.Title != "" && it.Page > 0 {
					titleByPage[it.Page] = it.Title
				}
			}
		}
		buf := strings.Builder{}
		bufStart := 0
		flushPages := func(end int) {
			text := strings.TrimSpace(buf.String())
			buf.Reset()
			if text == "" {
				return
			}
			title := fmt.Sprintf("第 %d-%d 页", bufStart, end)
			if t, ok := titleByPage[bufStart]; ok {
				title = t
			}
			appendChunks(title, bufStart, splitTextChunks(text, maxChars))
		}
		for pg := 1; pg <= np && len(segs) < audiobookMaxSegments; pg++ {
			t := pages[pg]
			if t == "" {
				continue
			}
			if buf.Len() == 0 {
				bufStart = pg
			}
			buf.WriteString(t)
			buf.WriteString("\n")
			if len([]rune(buf.String())) >= maxChars {
				flushPages(pg)
			}
		}
		flushPages(np)

	case "md", "txt":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return nil, fmt.Errorf("文件无文本内容")
		}
		if ft == "md" {
			// md 按标题行切分为节，保留结构感
			var parts []string
			cur := strings.Builder{}
			for _, line := range strings.Split(text, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "#") && cur.Len() > 0 {
					parts = append(parts, cur.String())
					cur.Reset()
				}
				cur.WriteString(line)
				cur.WriteString("\n")
			}
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
			}
			for i, part := range parts {
				title := fmt.Sprintf("第 %d 节", i+1)
				for _, ln := range strings.Split(part, "\n") {
					if t := strings.TrimSpace(ln); strings.HasPrefix(t, "#") {
						title = strings.TrimSpace(strings.TrimLeft(t, "#"))
						break
					}
				}
				// 跳过"# 目录"节，不生成语音
				if isTOCSection(title, part) {
					continue
				}
				appendChunks(title, i, splitTextChunks(part, maxChars))
				if len(segs) >= audiobookMaxSegments {
					break
				}
			}
		} else {
			appendChunks(book.Title, 0, splitTextChunks(text, maxChars))
		}

	default:
		return nil, fmt.Errorf("不支持的文件格式：%s", ft)
	}

	if len(segs) == 0 {
		return nil, fmt.Errorf("未能从书中提取到可朗读的文本")
	}
	return segs, nil
}

// ============ 分段元数据持久化 ============

// audiobookSegmentsVersion 分段元数据版本号：分段策略调整后递增，
// 旧版本元数据会被丢弃并重新构建（同时清理旧音频，避免续传错位）。
// v4：分段时跳过电子书目录（目录页/目录章节不生成语音）。
const audiobookSegmentsVersion = 4

// audiobookSegmentsFile 分段元数据文件结构
type audiobookSegmentsFile struct {
	Version  int                  `json:"version"`
	Segments []models.SegmentInfo `json:"segments"`
}

// SaveAudiobookSegments 保存分段元数据到任务目录（供进度查询与播放列表展示）
func SaveAudiobookSegments(dir string, segs []models.SegmentInfo) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(audiobookSegmentsFile{Version: audiobookSegmentsVersion, Segments: segs}, "", "  ")
	if err != nil {
		return err
	}
	tmp := audiobookSegmentsPath(dir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, audiobookSegmentsPath(dir))
}

// LoadAudiobookSegments 读取分段元数据（不存在或版本不匹配返回 nil，触发重建）
func LoadAudiobookSegments(dir string) []models.SegmentInfo {
	data, err := os.ReadFile(audiobookSegmentsPath(dir))
	if err != nil {
		return nil
	}
	var f audiobookSegmentsFile
	if json.Unmarshal(data, &f) != nil {
		return nil // 旧格式（纯数组），直接重建
	}
	if f.Version != audiobookSegmentsVersion {
		return nil
	}
	return f.Segments
}

// RecoverInterruptedAudiobooks 启动时将中断的任务（服务重启）标记为失败，
// 已生成的分段文件保留，重新生成时自动续传。
func RecoverInterruptedAudiobooks() {
	res := config.DB.Model(&models.AudiobookTask{}).
		Where("status IN ?", []string{"pending", "processing"}).
		Updates(map[string]interface{}{
			"status": "failed", "error": "服务重启，任务中断，请重新生成",
			"updated_at": time.Now(),
		})
	if res.RowsAffected > 0 {
		log.Printf("audiobook: %d interrupted task(s) marked as failed", res.RowsAffected)
	}
}

// DeleteAudiobookFiles 删除某本书的全部有声书文件
func DeleteAudiobookFiles(uid, bookID uint) error {
	dir := AudiobookDirFor(uid, bookID)
	if _, err := os.Stat(dir); err == nil {
		return os.RemoveAll(dir)
	}
	return nil
}

// ============ 异步生成 ============

// StartAudiobookGeneration 启动异步生成（goroutine 内执行，立即返回）
func StartAudiobookGeneration(taskID uint) {
	go runAudiobookGeneration(taskID)
}

func runAudiobookGeneration(taskID uint) {
	audiobookSem <- struct{}{} // 同时只允许一个任务
	defer func() {
		<-audiobookSem
		clearAudiobookCancel(taskID)
	}()

	var task models.AudiobookTask
	if err := config.DB.First(&task, taskID).Error; err != nil {
		log.Printf("audiobook: task %d not found: %v", taskID, err)
		return
	}
	if task.Status == "cancelled" || task.Status == "done" || task.Status == "paused" {
		return
	}

	var book models.Book
	if err := config.DB.First(&book, task.BookID).Error; err != nil {
		finishAudiobookTask(&task, "failed", "图书不存在或已删除")
		return
	}
	dir := AudiobookDirFor(task.UserID, task.BookID)

	// 合成配置：继承用户 TTS 设置（本地引擎地址等），任务字段覆盖。
	// 需在分段前构建：单段字符数按语速动态压缩，保证不超 Edge TTS 时长上限。
	cfg := GetTTSConfig(task.UserID)
	if task.Engine != "" {
		cfg.Engine = task.Engine
	}
	if task.Voice != "" {
		cfg.Voice = task.Voice
	}
	if task.SpeedFactor > 0 {
		cfg.SpeedFactor = task.SpeedFactor
	}
	if task.TextLang != "" {
		cfg.TextLang = task.TextLang
	}

	// 分段：优先复用已生成过的元数据（断点续传），否则重新构建
	segs := LoadAudiobookSegments(dir)
	if len(segs) == 0 {
		// 分段策略升级/首次构建：清理旧音频文件，避免分段结构变化后续传错位
		if olds, _ := filepath.Glob(filepath.Join(dir, "segment_*.mp3")); len(olds) > 0 {
			for _, f := range olds {
				os.Remove(f)
			}
			log.Printf("audiobook: task %d 分段结构已更新，清理 %d 个旧音频", taskID, len(olds))
		}
		var err error
		segs, err = BuildAudiobookSegments(&book, cfg.SpeedFactor)
		if err != nil {
			finishAudiobookTask(&task, "failed", err.Error())
			return
		}
		if err := SaveAudiobookSegments(dir, segs); err != nil {
			finishAudiobookTask(&task, "failed", "分段元数据写入失败")
			return
		}
	}

	task.Status = "processing"
	task.TotalSegments = len(segs)
	task.DoneSegments = 0
	config.DB.Model(&task).Updates(map[string]interface{}{
		"status": "processing", "total_segments": len(segs), "done_segments": 0,
		"error": "", "updated_at": time.Now(),
	})

	// 恢复进度：已生成文件且 done 标记的段跳过
	done := 0
	for i := range segs {
		if segs[i].Done && fileExists(filepath.Join(dir, segs[i].File)) {
			done++
		}
	}
	task.DoneSegments = done
	config.DB.Model(&task).Update("done_segments", done)

	log.Printf("audiobook: task %d start, %d segments (resume %d)", taskID, len(segs), done)

	// 并行合成：Edge TTS 为网络 IO 型，并发多路 WS 可大幅提升吞吐（默认 4 路）。
	// 段索引、失败计数与进度更新由互斥锁保护；取消/暂停/连续失败中止通过
	// 共享 stop 标志停止取段，worker 在每段处理前检查取消/暂停。
	// 单段合成失败自动重试 audiobookRetryTimes 次，命中限流/过载类错误自动降并发。
	var mu sync.Mutex
	consecutiveFails := 0
	lastSynthErr := error(nil) // 最近一段合成失败的真实原因，任务中止时带给用户
	stop := false
	doneSegs := done // 续传进度
	nextIdx := 0
	workers := audiobookSegmentWorkers()

	// 单任务并发控制器：worker 处理每段前先占用并发名额（上限=workers）。
	// 命中限流/过载类错误时在重试逻辑中 reduce() 折半上限，自动降低请求并发。
	lim := newAudiobookLimiter(workers)

	processSegment := func(i int) {
		seg := &segs[i]
		mp3Path := filepath.Join(dir, seg.File)
		if seg.Done && fileExists(mp3Path) {
			return
		}

		// 失败重试：单段合成失败多为瞬时网络抖动或上游限流，最多重试
		// audiobookRetryTimes 次（指数退避）。强信号（服务端明确限流/过载/断连）
		// 自动折半本任务并发并拉长等待；弱信号（客户端超时/网络抖动）仅普通退避，
		// 不降并发，避免把网络抖动误判为上游过载而过度降级。
		var data []byte
		backoff := audiobookRetryBackoffBase
		for attempt := 0; ; attempt++ {
			d, _, err := SynthesizeWithConfig(cfg, seg.Text, "", cfg.SpeedFactor, cfg.TextLang, "mp3")
			if err != nil {
				// Edge TTS 单次合成有 10 分钟时长硬上限；命中"音频超长"类错误时，
				// 将本段按句拆半分别合成后拼接（分段已按语速压缩，此为兜底）
				if isEdgeTTSTooLongError(err) {
					if d2, e2 := synthesizeSplitHalf(cfg, seg.Text); e2 == nil {
						d, err = d2, nil
						log.Printf("audiobook: task %d segment %d 音频超长，拆半重试成功", taskID, seg.Index)
					} else {
						err = e2 // 以拆半失败的真实原因（限流/超时等）参与重试判断
					}
				}
			}
			if err == nil {
				data = d
				break // 合成成功
			}

			// 错误分级：强信号=服务端明确限流/过载/断连 → 降并发并拉长退避；
			// 弱信号=客户端超时/网络抖动 → 仅普通退避重试（不降并发）；
			// 其余未知错误 → 同样普通退避重试。
			throttled := isTTSServerThrottleError(err)
			wait := backoff
			switch {
			case throttled:
				cur := lim.reduce() // 限流降级：并发上限折半（最低 1）
				wait *= audiobookRetryBackoffRateLimit
				log.Printf("audiobook: task %d segment %d 疑似服务限流/被掐断，并发降至 %d，%v 后重试：%v",
					taskID, seg.Index, cur, wait.Round(time.Millisecond), err)
			case isTTSTimeoutError(err):
				log.Printf("audiobook: task %d segment %d 疑似网络超时/抖动，%v 后重试（保持并发）：%v",
					taskID, seg.Index, wait.Round(time.Millisecond), err)
			default:
				log.Printf("audiobook: task %d segment %d attempt %d 失败，%v 后重试：%v",
					taskID, seg.Index, attempt+1, wait.Round(time.Millisecond), err)
			}

			if attempt >= audiobookRetryTimes {
				// 重试用尽：计入连续失败段数，达到阈值则中止整个任务
				mu.Lock()
				consecutiveFails++
				fails := consecutiveFails
				lastSynthErr = err
				mu.Unlock()
				log.Printf("audiobook: task %d segment %d failed after %d retries: %v",
					taskID, seg.Index, audiobookRetryTimes, err)
				if fails >= audiobookMaxConsecutiveFails {
					mu.Lock()
					stop = true
					mu.Unlock()
				}
				return
			}

			// 退避期间若任务被暂停/取消则退出本段（不计失败，续传时自动重做）
			if sleepInterruptible(taskID, wait) {
				return
			}
			backoff *= 2
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("audiobook: task %d mkdir failed: %v", taskID, err)
			mu.Lock()
			stop = true
			mu.Unlock()
			return
		}
		if err := os.WriteFile(mp3Path, data, 0644); err != nil {
			log.Printf("audiobook: task %d write failed: %v", taskID, err)
			mu.Lock()
			stop = true
			mu.Unlock()
			return
		}
		seg.Done = true
		seg.Duration = estimateDuration(seg.CharCount, cfg.SpeedFactor)

		mu.Lock()
		doneSegs++
		consecutiveFails = 0
		n := doneSegs
		mu.Unlock()
		task.DoneSegments = n
		config.DB.Model(&task).Updates(map[string]interface{}{
			"done_segments": n, "updated_at": time.Now(),
		})
		if err := SaveAudiobookSegments(dir, segs); err != nil {
			log.Printf("audiobook: task %d save segments meta failed: %v", taskID, err)
		}
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				// 先占用并发名额再取段：并发上限被限流降级后，多余 worker 自动排队
				lim.acquire()

				mu.Lock()
				if stop || nextIdx >= len(segs) {
					mu.Unlock()
					lim.release()
					return
				}
				i := nextIdx
				nextIdx++
				mu.Unlock()

				// 取消/暂停检查（等待并发名额期间可能收到请求，取段后再查一次）
				if isAudiobookCancelled(taskID) {
					lim.release()
					finishAudiobookTask(&task, "cancelled", "")
					log.Printf("audiobook: task %d cancelled", taskID)
					return
				}
				if isAudiobookPaused(taskID) {
					lim.release()
					finishAudiobookTask(&task, "paused", "")
					log.Printf("audiobook: task %d paused at %d/%d", taskID, doneSegs, task.TotalSegments)
					return
				}

				// 限流降级后的自适应回升：持续稳定 audiobookRecoverGrace 未再命中
				// 限流强信号时，试探恢复一级并发，逐级回升到初始并发
				if cur, recovered := lim.maybeRecover(); recovered {
					log.Printf("audiobook: task %d 限流后已稳定，并发恢复至 %d", taskID, cur)
				}

				processSegment(i)
				lim.release()
			}
		}()
	}
	wg.Wait()

	// 连续失败中止：worker 结束后统一收尾，避免并发重复写任务状态
	mu.Lock()
	fails := consecutiveFails
	synthErr := lastSynthErr
	mu.Unlock()
	if fails >= audiobookMaxConsecutiveFails {
		// 带上最后一段的真实失败原因（含排障指引），用户可直接看到 403/连接失败等根因
		msg := fmt.Sprintf("连续 %d 段合成失败，已中止", fails)
		if synthErr != nil {
			msg += "：" + synthErr.Error()
		}
		finishAudiobookTask(&task, "failed", msg)
		log.Printf("audiobook: task %d aborted (consecutive fails %d): %v", taskID, fails, synthErr)
		return
	}

	if isAudiobookCancelled(taskID) {
		finishAudiobookTask(&task, "cancelled", "")
		return
	}
	if isAudiobookPaused(taskID) {
		finishAudiobookTask(&task, "paused", "")
		return
	}
	if task.DoneSegments >= task.TotalSegments {
		finishAudiobookTask(&task, "done", "")
	} else {
		// 同样带上最后一段的真实失败原因（含排障指引）
		msg := fmt.Sprintf("仅完成 %d/%d 段，其余段合成失败", task.DoneSegments, task.TotalSegments)
		if synthErr != nil {
			msg += "：" + synthErr.Error()
		}
		finishAudiobookTask(&task, "failed", msg)
	}
}

// ResumeAudiobookTask 恢复已暂停的任务（重新入队，已生成分段自动续传）
func ResumeAudiobookTask(taskID uint) error {
	var task models.AudiobookTask
	if err := config.DB.First(&task, taskID).Error; err != nil {
		return err
	}
	if task.Status != "paused" {
		return fmt.Errorf("任务不在暂停状态")
	}
	now := time.Now()
	config.DB.Model(&task).Updates(map[string]interface{}{
		"status": "pending", "error": "", "finished_at": nil, "updated_at": now,
	})
	StartAudiobookGeneration(task.ID)
	return nil
}

func finishAudiobookTask(task *models.AudiobookTask, status, errMsg string) {
	now := time.Now()
	config.DB.Model(task).Updates(map[string]interface{}{
		"status": status, "error": errMsg, "updated_at": now, "finished_at": now,
	})
}

func estimateDuration(chars int, speed float64) int {
	if speed <= 0 {
		speed = 1.0
	}
	// 中文朗读约 5 字/秒，按语速折算
	return int(float64(chars) / (5.0 * speed))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
