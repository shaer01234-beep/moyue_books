package services

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// TOCItem 统一的目录条目结构
type TOCItem struct {
	Index  int    `json:"index"`  // 阅读器定位序号（EPUB=spine 序号；页式格式=页码-1）
	Title  string `json:"title"`
	Level  int    `json:"level"`  // 层级，1 为一级标题
	Path   string `json:"path"`   // EPUB 内容文件路径 / CBZ 图片路径（渲染用）
	Page   int    `json:"page"`   // 页式格式的页码（0 表示不适用）
	Anchor string `json:"anchor"` // EPUB 内容文件内的锚点（如 "ch1.xhtml#sec2"，不含 #，空表示无）
}

// ExtractTOC 统一目录解析入口。按文件类型分发到各格式解析器；
// 无法解析的格式返回空目录（不报错，交由调用方跳过）。
func ExtractTOC(filePath, fileType string) ([]TOCItem, error) {
	ft := strings.ToLower(fileType)
	switch ft {
	case "epub", "kepub":
		return extractEPUBTOC(filePath), nil
	case "azw3", "mobi":
		return extractAZW3TOC(filePath)
	case "md":
		return extractMarkdownTOC(filePath), nil
	case "pdf":
		return extractPDFTOC(filePath), nil
	default:
		// 漫画 / 音频等不支持格式：跳过
		return []TOCItem{}, nil
	}
}

// ============ EPUB / KEPUB ============

// extractEPUBTOC 优先解析 NCX 目录，其次 NAV，最后回退 spine 顺序章节。
func extractEPUBTOC(filePath string) []TOCItem {
	epub, err := OpenEPUB(filePath)
	if err != nil {
		return []TOCItem{}
	}
	defer epub.Close()
	spine := epub.Contents

	if items := parseEPUBNCX(epub, spine); len(items) > 0 {
		return items
	}
	if items := parseEPUBNAV(epub, spine); len(items) > 0 {
		return items
	}
	// 回退：按 spine 顺序生成"章节 N"
	var items []TOCItem
	for i, path := range spine {
		items = append(items, TOCItem{
			Index: i,
			Title: fmt.Sprintf("章节 %d", i+1),
			Level: 1,
			Path:  path,
		})
	}
	return items
}

type ncxPoint struct {
	Label    struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content  struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Children []ncxPoint `xml:"navPoint"`
}

// parseEPUBNCX 解析 EPUB 的 NCX 目录（toc.ncx）
func parseEPUBNCX(e *EPUBReader, spine []string) []TOCItem {
	for _, f := range e.reader.File {
		name := strings.ToLower(f.Name)
		if !strings.HasSuffix(name, ".ncx") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(rc)
		rc.Close()

		var root struct {
			NavMap struct {
				NavPoint []ncxPoint `xml:"navPoint"`
			} `xml:"navMap"`
		}
		if xml.Unmarshal(data, &root) != nil {
			continue
		}
		var items []TOCItem
		collectNCX(root.NavMap.NavPoint, 1, spine, &items)
		if len(items) > 0 {
			return items
		}
	}
	return nil
}

func collectNCX(points []ncxPoint, level int, spine []string, items *[]TOCItem) {
	for _, p := range points {
		if idx, anchor, ok := spineIndex(p.Content.Src, spine); ok {
			title := strings.TrimSpace(p.Label.Text)
			if title != "" {
				*items = append(*items, TOCItem{Index: idx, Title: title, Level: level, Path: spine[idx], Anchor: anchor})
			}
		}
		collectNCX(p.Children, level+1, spine, items)
	}
}

type navA struct {
	Href  string `xml:"href,attr"`
	Inner string `xml:",innerxml"`
}

type navLI struct {
	A  navA   `xml:"a"`
	OL *navOL `xml:"ol"`
}

type navOL struct {
	Li []navLI `xml:"li"`
}

// parseEPUBNAV 解析 EPUB 的 NAV 目录（nav.xhtml，epub:type="toc"）
func parseEPUBNAV(e *EPUBReader, spine []string) []TOCItem {
	for _, f := range e.reader.File {
		name := strings.ToLower(f.Name)
		if !strings.HasSuffix(name, ".xhtml") && !strings.HasSuffix(name, ".html") {
			continue
		}
		if !strings.Contains(name, "nav") && !strings.Contains(name, "toc") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(rc)
		rc.Close()

		var doc struct {
			Body struct {
				Nav []struct {
					Type string `xml:"http://www.idpf.org/2007/ops type,attr"`
					OL   *navOL `xml:"ol"`
				} `xml:"nav"`
			} `xml:"body"`
		}
		if xml.Unmarshal(data, &doc) != nil {
			continue
		}
		var items []TOCItem
		for _, nav := range doc.Body.Nav {
			if nav.OL == nil {
				continue
			}
			collectNAV(nav.OL.Li, 1, spine, &items)
		}
		if len(items) > 0 {
			return items
		}
	}
	return nil
}

func collectNAV(li []navLI, level int, spine []string, items *[]TOCItem) {
	for _, l := range li {
		if l.A.Href != "" {
			if idx, anchor, ok := spineIndex(l.A.Href, spine); ok {
				title := strings.TrimSpace(stripTags(l.A.Inner))
				if title != "" {
					*items = append(*items, TOCItem{Index: idx, Title: title, Level: level, Path: spine[idx], Anchor: anchor})
				}
			}
		}
		if l.OL != nil {
			collectNAV(l.OL.Li, level+1, spine, items)
		}
	}
}

// spineIndex 将目录条目 href 映射到 spine 文件序号。
// 同时返回锚点（去掉 "#" 前缀），用于在已加载的 iframe 内精确滚动到具体小节。
// 返回 (spine 序号, 锚点, 是否匹配)。
func spineIndex(src string, spine []string) (int, string, bool) {
	if src == "" {
		return 0, "", false
	}
	href := src
	anchor := ""
	if i := strings.Index(href, "#"); i >= 0 {
		anchor = href[i+1:]
		href = href[:i]
	}
	if i := strings.Index(href, "?"); i >= 0 {
		href = href[:i]
	}
	href = filepath.ToSlash(href)
	href = strings.TrimPrefix(href, "./")
	if un, err := url.PathUnescape(href); err == nil {
		href = un
	}
	for i, s := range spine {
		if filepath.ToSlash(s) == href {
			return i, anchor, true
		}
	}
	// 容错：仅文件名匹配
	base := filepath.Base(href)
	for i, s := range spine {
		if filepath.Base(filepath.ToSlash(s)) == base {
			return i, anchor, true
		}
	}
	return 0, "", false
}

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return strings.TrimSpace(htmlTagRE.ReplaceAllString(s, ""))
}

// ============ Markdown ============

var mdHeadingRE = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// extractMarkdownTOC 解析 Markdown 的 # ~ ###### 标题生成层级目录
func extractMarkdownTOC(filePath string) []TOCItem {
	f, err := os.Open(filePath)
	if err != nil {
		return []TOCItem{}
	}
	defer f.Close()

	var items []TOCItem
	idx := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}
		m := mdHeadingRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		title := strings.TrimSpace(m[2])
		if title == "" {
			continue
		}
		items = append(items, TOCItem{
			Index: idx,
			Title: title,
			Level: len(m[1]),
			Page:  idx + 1,
		})
		idx++
	}
	return items
}

// ============ PDF ============

// extractPDFTOC 解析 PDF 书签（Outline）生成层级目录；
// PDF 无书签时回退到文本层识别（定位"目录"页并解析条目），
// 两者都失败则返回空目录。
func extractPDFTOC(filePath string) []TOCItem {
	f, err := os.Open(filePath)
	if err != nil {
		return []TOCItem{}
	}

	bookmarks, err := api.Bookmarks(f, nil)
	f.Close()
	if err != nil || len(bookmarks) == 0 {
		// 无书签：回退到文本层目录页识别
		return extractPDFTextTOC(filePath)
	}

	var items []TOCItem
	collectPDFBookmarks(bookmarks, 1, &items)
	return items
}

// collectPDFBookmarks 递归展平 PDF 书签树，生成带页码与层级的目录。
func collectPDFBookmarks(bms []pdfcpu.Bookmark, level int, items *[]TOCItem) {
	for _, bm := range bms {
		title := strings.TrimSpace(bm.Title)
		if title != "" {
			page := bm.PageFrom
			if page < 1 {
				page = 1
			}
			*items = append(*items, TOCItem{
				Index: len(*items), // 数组序号
				Title: title,
				Level: level,
				Page:  page, // 1-based 页码，前端据此跳转
			})
		}
		collectPDFBookmarks(bm.Kids, level+1, items)
	}
}

// ============ PDF 无书签：文本层目录页识别 ============

var (
	pdfReSpace  = regexp.MustCompile(`[\s\p{Zs}]+`)
	pdfRePunct  = regexp.MustCompile(`[，。、？！；：""''（）《》【】\.,!?;:()"']`)
	pdfReZhChap = regexp.MustCompile(`^第[一二三四五六七八九十百千0-9]+[章篇部卷]`)
	pdfReEnChap = regexp.MustCompile(`(?i)^chapter\s*\d+`)
	pdfRePageNo = regexp.MustCompile(`^\d{1,3}$`)
	// 行尾页码（标题与页码同行时拆分），标题首个字符不能是数字以防误拆纯数字
	pdfReTailNo = regexp.MustCompile(`^(\D.{0,}?)(\d{1,3})$`)
	// 无目录页兜底：编号标题识别（中文序号 / 数字编号 / 步骤章节）
	pdfReNumTitle   = regexp.MustCompile(`^\d{1,3}[.、．]`)
	pdfReZhNumTitle = regexp.MustCompile(`^[一二三四五六七八九十]{1,3}[、.．]`)
	pdfReStepTitle  = regexp.MustCompile(`^第[一二三四五六七八九十0-9]{1,3}[步环节部]`)
	pdfReZhOrdinal  = regexp.MustCompile(`^第[一二三四五六七八九十0-9]{1,3}[、.．]`)
)

// normTocText 规范化页文本：去行内空白与常见标点，保留换行结构。
func normTocText(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		ln = pdfReSpace.ReplaceAllString(ln, "")
		ln = pdfRePunct.ReplaceAllString(ln, "")
		lines[i] = ln
	}
	return strings.Join(lines, "\n")
}

// tocPageEntry 文本层识别出的目录条目（解析中间态）
type tocPageEntry struct {
	title string // 原始标题（可读）
	page  int    // 印刷页码（0 表示无）
	level int    // 1 章 / 2 节
}

// extractPDFTextTOC 无书签 PDF 回退：定位"目录"页，按"标题行 + 页码行"
// 交替结构解析条目；再用目录标题在正文页文本中匹配，将印刷页码校准为
// PDF 实际页码，生成带层级与页码的目录。
func extractPDFTextTOC(filePath string) []TOCItem {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return []TOCItem{}
	}
	np := r.NumPage()
	f.Close()
	if np <= 0 {
		return []TOCItem{}
	}

	// 并发多 Reader 提取页文本（Page() 非线程安全，每 worker 独立打开文件）
	rawText := make([]string, np+1)
	normText := make([]string, np+1)
	const workers = 6
	var wg sync.WaitGroup
	tasks := make(chan int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fw, rw, err := pdf.Open(filePath)
			if err != nil {
				return
			}
			defer fw.Close()
			for pg := range tasks {
				if pg < 1 || pg > np {
					continue
				}
				p := rw.Page(pg)
				if p.V.IsNull() {
					continue
				}
				t, _ := p.GetPlainText(nil)
				rawText[pg] = t
				normText[pg] = normTocText(t)
			}
		}()
	}
	for pg := 1; pg <= np; pg++ {
		tasks <- pg
	}
	close(tasks)
	wg.Wait()

	// 定位目录起始页（前 100 页内含"目录"/"CONTENTS"）
	tocStart := 0
	limit := np
	if limit > 100 {
		limit = 100
	}
	for i := 1; i <= limit; i++ {
		if strings.Contains(normText[i], "目录") || strings.Contains(normText[i], "CONTENTS") {
			tocStart = i
			break
		}
	}
	if tocStart == 0 {
		// 无目录页：兜底识别页面中的编号标题（中文序号/数字编号/步骤）
		return extractPDFFlatTOC(filePath, np)
	}

	// 目录页区间：连续页中独立页码行 >= 3 视为目录续页（最多 8 页）
	tocPageNoCount := func(t string) int {
		c := 0
		for _, ln := range strings.FieldsFunc(t, func(r rune) bool { return r == '\n' || r == '\r' }) {
			if pdfRePageNo.MatchString(strings.TrimSpace(ln)) {
				c++
			}
		}
		return c
	}
	tocEnd := tocStart
	for i := tocStart + 1; i <= np && i-tocStart <= 8; i++ {
		if tocPageNoCount(normText[i]) >= 3 {
			tocEnd = i
		} else {
			break
		}
	}

	// 解析目录条目
	var items []tocPageEntry
	var cur strings.Builder
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" && s != "目录" && !strings.EqualFold(s, "CONTENTS") {
			items = append(items, tocPageEntry{title: s, page: 0, level: 2})
		}
		cur.Reset()
	}
	setPage := func(p string) {
		flush() // 页码出现即标题结束，先落盘当前累积
		if len(items) == 0 || items[len(items)-1].level != 2 || items[len(items)-1].page != 0 {
			return
		}
		if n, err := strconv.Atoi(p); err == nil {
			items[len(items)-1].page = n
		}
	}
	pendingChapName := false // 章标题后待并入的章名（仅限紧随的下一文本行）
	for i := tocStart; i <= tocEnd; i++ {
		lines := strings.FieldsFunc(rawText[i], func(r rune) bool { return r == '\n' || r == '\r' })
		for _, raw := range lines {
			ln := strings.TrimSpace(raw)
			if ln == "" {
				flush()
				pendingChapName = false
				continue
			}
			if pdfReZhChap.MatchString(ln) || pdfReEnChap.MatchString(ln) {
				flush()
				items = append(items, tocPageEntry{title: ln, page: 0, level: 1})
				pendingChapName = true
				continue
			}
			if pdfRePageNo.MatchString(ln) {
				setPage(ln)
				pendingChapName = false
				continue
			}
			if m := pdfReTailNo.FindStringSubmatch(ln); m != nil && len(m[1]) > 1 {
				// 标题与页码同行：追加标题并 flush，再登记页码
				cur.WriteString(strings.TrimSpace(m[1]))
				flush()
				setPage(m[2])
				pendingChapName = false
				continue
			}
			// 章标题后的下一文本行并入章名
			if pendingChapName && len(items) > 0 && items[len(items)-1].level == 1 {
				items[len(items)-1].title += " " + ln
				pendingChapName = false
				continue
			}
			cur.WriteString(ln)
		}
	}
	flush()
	if len(items) < 3 {
		return []TOCItem{}
	}

	// 印刷页码 → PDF 页码偏移校准：用目录条目在正文中的首个匹配页反推
	var offsets []int
	for idx := range items {
		if items[idx].level != 2 || items[idx].page <= 0 {
			continue
		}
		nt := normTocText(items[idx].title)
		if len([]rune(nt)) < 4 {
			continue
		}
		for pg := tocEnd + 1; pg <= np; pg++ {
			if strings.Contains(normText[pg], nt) {
				offsets = append(offsets, pg-items[idx].page)
				break
			}
		}
	}
	offset := 0
	if len(offsets) > 0 {
		sort.Ints(offsets)
		offset = offsets[len(offsets)/2]
	} else {
		// 兜底：正文印刷第 1 页约在目录末页后 1~2 页
		offset = tocEnd + 2 - 1
	}

	// 生成最终目录
	var result []TOCItem
	for idx := range items {
		it := items[idx]
		page := 0
		switch {
		case it.level == 2 && it.page > 0:
			page = it.page + offset
		case it.level == 1:
			// 章标题：用"第X章"/"Chapter N"片段在正文中匹配起始页
			nt := normTocText(it.title)
			for _, pat := range []string{pdfReZhChap.FindString(nt), pdfReEnChap.FindString(nt)} {
				if pat == "" {
					continue
				}
				for pg := tocEnd + 1; pg <= np; pg++ {
					if strings.Contains(normText[pg], pat) {
						page = pg
						break
					}
				}
				if page > 0 {
					break
				}
			}
		}
		if page < 1 {
			page = 1
		}
		if page > np {
			page = np
		}
		title := strings.TrimSpace(it.title)
		if title == "" {
			continue
		}
		result = append(result, TOCItem{
			Index: len(result),
			Title: title,
			Level: it.level,
			Page:  page,
		})
	}
	return result
}

// absFloat 绝对值
func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// pdfCoordsLines 按坐标聚合页文本为可读行（解决中文 PDF 文本块被拆散的问题）。
// 按 Y（近似行）聚合并按 X 排序拼接；仅返回非空行。
// 页面内容流损坏/截断时 ledongthuc/pdf 的 Page.Content() 可能 panic，
// 此处 recover 兜底，将该页视为无文本，避免拖垮整个进程。
func pdfCoordsLines(p pdf.Page) (lines []string) {
	defer func() {
		if r := recover(); r != nil {
			lines = nil
		}
	}()
	content := p.Content()
	type rn struct {
		x, y float64
		s    string
	}
	var runs []rn
	for _, t := range content.Text {
		if strings.TrimSpace(t.S) == "" {
			continue
		}
		runs = append(runs, rn{x: t.X, y: t.Y, s: t.S})
	}
	if len(runs) == 0 {
		return nil
	}
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i].y != runs[j].y {
			return runs[i].y > runs[j].y
		}
		return runs[i].x < runs[j].x
	})
	curY := runs[0].y
	cur := runs[0].s
	for _, r := range runs[1:] {
		if absFloat(r.y-curY) < 2 {
			cur += r.s
		} else {
			lines = append(lines, cur)
			curY = r.y
			cur = r.s
		}
	}
	lines = append(lines, cur)
	return lines
}

// pdfTitleFromLine 从文本行中提取"编号标题"；非标题行返回空串。
// 支持：一、xxx / 1.xxx / 第一步：xxx 等；标题截取到首个冒号前。
func pdfTitleFromLine(ln string) string {
	ln = strings.TrimSpace(ln)
	if ln == "" {
		return ""
	}
	isStep := pdfReStepTitle.MatchString(ln)
	isZhNum := pdfReZhNumTitle.MatchString(ln)
	isDigi := pdfReNumTitle.MatchString(ln)
	isOrd := pdfReZhOrdinal.MatchString(ln)
	if !isStep && !isZhNum && !isDigi && !isOrd {
		return ""
	}
	t := ln
	if i := strings.IndexAny(t, "：:"); i > 0 {
		if isStep && len([]rune(t[:i])) < 5 {
			// 步骤标题冒号前过短（如"第一步："），保留整行避免信息丢失
			// （不做截取）
		} else {
			t = t[:i]
		}
	}
	t = strings.TrimSpace(t)
	n := len([]rune(t))
	if n < 2 || n > 30 {
		return ""
	}
	return t
}

// extractPDFFlatTOC 无目录页 PDF 兜底：逐页按坐标聚合文本行，
// 识别编号标题（一、/1./第一步 等）生成目录，页码为条目所在 PDF 页。
// 仅适用于页数较少的文档（100 页内），避免长文正文误报与解析耗时。
func extractPDFFlatTOC(filePath string, np int) []TOCItem {
	if np <= 0 || np > 100 {
		return []TOCItem{}
	}
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return []TOCItem{}
	}
	defer f.Close()

	var items []TOCItem
	for pg := 1; pg <= np; pg++ {
		p := r.Page(pg)
		if p.V.IsNull() {
			continue
		}
		seen := map[string]bool{}
		for _, ln := range pdfCoordsLines(p) {
			title := pdfTitleFromLine(ln)
			if title == "" {
				continue
			}
			if seen[title] {
				continue
			}
			seen[title] = true
			items = append(items, TOCItem{
				Index: len(items),
				Title: title,
				Level: 2,
				Page:  pg,
			})
		}
	}
	if len(items) < 2 {
		return []TOCItem{}
	}
	return items
}
