package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// toUTF8 探测字节流的文本编码并转换为 UTF-8 字符串。
// 优先按 UTF-8 处理（兼容绝大多数 .md/.txt）；若不是合法 UTF-8，则尝试
// 中文常见编码 GB18030(含 GBK/GB2312)/Big5，最后再交给通用字符集探测兜底。
// 含 UTF-8/UTF-16 BOM 时会自动识别。
func toUTF8(raw []byte) string {
	// 去除 UTF-8 BOM，避免零宽字符混入正文
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	if utf8.Valid(raw) {
		return string(raw)
	}
	// 中文文本书绝大多数为 GBK/GB18030（简体）或 Big5（繁体）。
	// 这些解码器对任意字节都能解出“合法 UTF-8”，故以解码结果是否为合法
	// UTF-8 作为判定依据，GB18030 优先（其覆盖 GBK/GB2312）。
	for _, dec := range []encoding.Encoding{
		simplifiedchinese.GB18030,
		traditionalchinese.Big5,
		simplifiedchinese.HZGB2312,
	} {
		if decoded, err := dec.NewDecoder().Bytes(raw); err == nil && utf8.Valid(decoded) {
			return string(decoded)
		}
	}
	// 兜底：交给通用字符集探测（Shift-JIS、EUC-KR 等）
	if enc, _, _ := charset.DetermineEncoding(raw, ""); enc != nil {
		if decoded, err := enc.NewDecoder().Bytes(raw); err == nil {
			return string(decoded)
		}
	}
	// 仍无法识别则原样返回，避免 panic
	return string(raw)
}

// ConvertTextToEPUB 将 Markdown(.md) 或纯文本(.txt) 转换为标准 EPUB 文件，
// 生成的文件位于 outDir/books 目录，返回其绝对路径。转换成功后原文件可删除，
// 由调用方负责。转换得到的 EPUB 与系统内置 EPUB 走完全相同的阅读/目录/封面通道。
//
// Markdown 会以一级标题(# )为界拆分为多章节，便于阅读器目录导航。
func ConvertTextToEPUB(srcPath, title, fileType, outDir string) (string, error) {
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}
	// 文本文件可能是 GBK/GB18030/Big5 等编码，需探测并统一转成 UTF-8，
	// 否则中文会显示为乱码。
	content := toUTF8(raw)

	var chapters []mdChapter
	switch strings.ToLower(fileType) {
	case "md":
		chapters = splitMarkdownChapters(content)
	default: // txt
		chapters = []mdChapter{{Title: "", Body: content}}
	}
	// 将超大章节按段落切分为多个子章节，避免单章体积过大（尤其大体积 txt）
	// 导致阅读器一次性加载整个 DOM 而解析失败或卡死。
	chapters = boundChapters(chapters)

	// 转换各章节正文为 HTML
	for i := range chapters {
		if strings.ToLower(fileType) == "md" {
			chapters[i].Body = convertMarkdownLines(chapters[i].Body)
		} else {
			body, _ := txtToHTML(chapters[i].Body)
			chapters[i].Body = body
		}
	}

	// 书名：未显式指定时取首个一级标题，再回退文件名
	if title == "" {
		if len(chapters) > 0 && chapters[0].Title != "" {
			title = chapters[0].Title
		} else {
			title = strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		}
	}
	// 章节标题兜底
	for i := range chapters {
		if chapters[i].Title == "" {
			if i == 0 {
				chapters[i].Title = title
			} else {
				chapters[i].Title = fmt.Sprintf("第 %d 章", i+1)
			}
		}
	}

	uid := uuid.New().String()
	booksDir := filepath.Join(outDir, "books")
	if err := os.MkdirAll(booksDir, 0o755); err != nil {
		return "", err
	}
	outPath := filepath.Join(booksDir, uid+".epub")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// mimetype 必须是压缩包内第一个文件且采用 STORE（不压缩），符合 EPUB 规范
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mw, err := zw.CreateHeader(mh)
	if err == nil {
		mw.Write([]byte("application/epub+zip"))
	}

	write := func(name, data string) {
		fw, e := zw.Create(name)
		if e != nil {
			return
		}
		fw.Write([]byte(data))
	}
	write("META-INF/container.xml", containerXML)
	write("OEBPS/style.css", styleCSS)
	for i, ch := range chapters {
		write(fmt.Sprintf("OEBPS/chapter-%d.xhtml", i+1), buildChapterXHTML(ch.Title, ch.Body))
	}
	write("OEBPS/toc.ncx", buildNCX(uid, title, chapters))
	write("OEBPS/content.opf", buildOPF(uid, title, chapters))

	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

// ---------- 章节拆分 ----------

type mdChapter struct {
	Title string
	Body  string
}

// splitMarkdownChapters 按一级标题(# )将 Markdown 拆分为多章节。
// 首个一级标题之前的内容作为独立的“前言”章节；若无任何一级标题，则整篇作为单章。
func splitMarkdownChapters(md string) []mdChapter {
	lines := strings.Split(md, "\n")
	var chapters []mdChapter
	var cur *mdChapter
	flush := func() {
		if cur != nil {
			chapters = append(chapters, *cur)
			cur = nil
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 仅以单独成行的一级标题(# )作为分章边界
		if m := headingRe.FindStringSubmatch(trimmed); m != nil && len(m[1]) == 1 {
			flush()
			cur = &mdChapter{Title: strings.TrimSpace(m[2]), Body: ""}
			continue
		}
		if cur == nil {
			cur = &mdChapter{Title: "", Body: ""}
		}
		cur.Body += line + "\n"
	}
	flush()
	if len(chapters) == 0 {
		chapters = append(chapters, mdChapter{Title: "", Body: md})
	}
	return chapters
}

// boundChapters 将超过单章大小上限的章节按段落（空行）切分为多个子章节。
// 这样阅读器（foliate 等）可逐章懒加载，避免单章体积过大而无法打开/卡死；
// 同时保留原章节标题：首段沿用原标题（txt 首章为空则回退书名），后续子段追加序号。
func boundChapters(chapters []mdChapter) []mdChapter {
	const maxBytes = 200 * 1024 // 每章约 200KB 原始文本
	var out []mdChapter
	for _, ch := range chapters {
		if len(ch.Body) <= maxBytes {
			out = append(out, ch)
			continue
		}
		paras := strings.Split(ch.Body, "\n")
		var buf []string
		cur := 0
		sub := 1
		flush := func() {
			if len(buf) == 0 {
				return
			}
			title := ch.Title
			if sub > 1 {
				if ch.Title != "" {
					title = fmt.Sprintf("%s（%d）", ch.Title, sub)
				} else {
					title = fmt.Sprintf("第 %d 部分", sub)
				}
			}
			out = append(out, mdChapter{Title: title, Body: strings.Join(buf, "\n")})
			buf = nil
			cur = 0
			sub++
		}
		for _, l := range paras {
			// 当前缓冲非空且继续追加会超限时，先切出一段
			if cur+len(l)+1 > maxBytes && len(buf) > 0 {
				flush()
			}
			buf = append(buf, l)
			cur += len(l) + 1
		}
		flush()
	}
	return out
}

// ---------- 文本 -> HTML ----------

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	hrRe      = regexp.MustCompile(`^(\-{3,}|\*{3,}|_{3,})$`)
	ulRe      = regexp.MustCompile(`^[-*+]\s+`)
	olRe      = regexp.MustCompile(`^\d+\.\s+`)
)

// convertMarkdownLines 将 Markdown 正文（不含被拆分出去的一级标题行）转换为 HTML 片段。
func convertMarkdownLines(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	i, n := 0, len(lines)
	for i < n {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 代码块围栏
		if strings.HasPrefix(trimmed, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			i++
			var buf []string
			for i < n && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				buf = append(buf, lines[i])
				i++
			}
			i++ // 跳过结束围栏
			code := xmlEscape(strings.Join(buf, "\n"))
			cls := ""
			if lang != "" {
				cls = fmt.Sprintf(" class=%q", "language-"+xmlEscape(lang))
			}
			out = append(out, fmt.Sprintf("<pre><code%s>%s</code></pre>", cls, code))
			continue
		}
		if trimmed == "" {
			i++
			continue
		}
		// 标题
		if m := headingRe.FindStringSubmatch(trimmed); m != nil {
			level := len(m[1])
			text := inlineToHTML(m[2])
			out = append(out, fmt.Sprintf("<h%d>%s</h%d>", level, text, level))
			i++
			continue
		}
		// 分隔线
		if hrRe.MatchString(trimmed) {
			out = append(out, "<hr/>")
			i++
			continue
		}
		// 引用块
		if strings.HasPrefix(trimmed, ">") {
			var buf []string
			for i < n && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				buf = append(buf, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), ">")))
				i++
			}
			out = append(out, fmt.Sprintf("<blockquote>%s</blockquote>", inlineToHTML(strings.Join(buf, "<br/>"))))
			continue
		}
		// 无序列表
		if ulRe.MatchString(trimmed) {
			var items []string
			for i < n && ulRe.MatchString(strings.TrimSpace(lines[i])) {
				item := strings.TrimSpace(lines[i])
				item = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(item, "-"), "*"))
				item = strings.TrimSpace(strings.TrimPrefix(item, "+"))
				items = append(items, fmt.Sprintf("<li>%s</li>", inlineToHTML(item)))
				i++
			}
			out = append(out, "<ul>"+strings.Join(items, "")+"</ul>")
			continue
		}
		// 有序列表
		if olRe.MatchString(trimmed) {
			var items []string
			for i < n && olRe.MatchString(strings.TrimSpace(lines[i])) {
				item := olRe.ReplaceAllString(strings.TrimSpace(lines[i]), "")
				items = append(items, fmt.Sprintf("<li>%s</li>", inlineToHTML(item)))
				i++
			}
			out = append(out, "<ol>"+strings.Join(items, "")+"</ol>")
			continue
		}
		// 段落：聚合连续文本行
		var para []string
		for i < n && strings.TrimSpace(lines[i]) != "" &&
			!strings.HasPrefix(strings.TrimSpace(lines[i]), "```") &&
			!strings.HasPrefix(strings.TrimSpace(lines[i]), "#") &&
			!strings.HasPrefix(strings.TrimSpace(lines[i]), ">") &&
			!ulRe.MatchString(strings.TrimSpace(lines[i])) &&
			!olRe.MatchString(strings.TrimSpace(lines[i])) &&
			!hrRe.MatchString(strings.TrimSpace(lines[i])) {
			para = append(para, lines[i])
			i++
		}
		if len(para) > 0 {
			out = append(out, fmt.Sprintf("<p>%s</p>", inlineToHTML(strings.Join(para, "<br/>"))))
		}
	}
	return strings.Join(out, "\n")
}

func txtToHTML(txt string) (string, string) {
	lines := strings.Split(txt, "\n")
	var paras []string
	var first string
	var cur []string
	flush := func() {
		if len(cur) == 0 {
			return
		}
		if first == "" {
			first = strings.TrimSpace(cur[0])
		}
		paras = append(paras, "<p>"+strings.Join(cur, "<br/>")+"</p>")
		cur = nil
	}
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			flush()
			continue
		}
		cur = append(cur, xmlEscape(l))
	}
	flush()
	return strings.Join(paras, "\n"), first
}

// inlineToHTML 处理行内格式：代码、图片、链接、粗体、斜体
func inlineToHTML(s string) string {
	escaped := xmlEscape(s)

	// 先保护行内代码，避免其中的字符被二次格式化
	codeRe := regexp.MustCompile("`([^`]+)`")
	var codeStore []string
	escaped = codeRe.ReplaceAllStringFunc(escaped, func(m string) string {
		sub := codeRe.FindStringSubmatch(m)[1]
		codeStore = append(codeStore, "<code>"+sub+"</code>")
		return fmt.Sprintf("@@CODE%d@@", len(codeStore)-1)
	})

	imgRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	escaped = imgRe.ReplaceAllString(escaped, `<img alt="$1" src="$2"/>`)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	escaped = linkRe.ReplaceAllString(escaped, `<a href="$2">$1</a>`)

	escaped = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(escaped, "<em>$1</em>")
	escaped = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(escaped, "<em>$1</em>")

	for idx, c := range codeStore {
		escaped = strings.Replace(escaped, fmt.Sprintf("@@CODE%d@@", idx), c, 1)
	}
	return escaped
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// ---------- EPUB 构件 ----------

const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

const styleCSS = `body{font-family:-apple-system,"Segoe UI",Roboto,"Helvetica Neue",Arial,"PingFang SC","Microsoft YaHei",sans-serif;line-height:1.8;color:#222;margin:0;padding:1.2em 1.4em;}
h1,h2,h3,h4,h5,h6{line-height:1.3;margin:1.2em 0 .6em;word-break:break-word;}
h1{font-size:1.6em;} h2{font-size:1.4em;} h3{font-size:1.2em;}
p{margin:.8em 0;text-align:justify;}
a{color:#2563eb;text-decoration:none;}
blockquote{margin:.8em 0;padding:.4em 1em;color:#555;border-left:4px solid #ddd;background:#fafafa;}
code{background:#f3f4f6;padding:.15em .35em;border-radius:4px;font-family:Consolas,Menlo,monospace;font-size:.92em;}
pre{background:#0f172a;color:#e2e8f0;padding:1em;overflow:auto;border-radius:8px;}
pre code{background:transparent;color:inherit;padding:0;}
ul,ol{margin:.6em 0;padding-left:1.6em;}
hr{border:none;border-top:1px solid #e5e7eb;margin:1.4em 0;}
img{max-width:100%;height:auto;}
.chapter-title{text-align:center;}`

func buildChapterXHTML(title, body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="utf-8"/>
<title>%s</title>
<link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
<h1 class="chapter-title">%s</h1>
%s
</body>
</html>`, xmlEscape(title), xmlEscape(title), body)
}

const ncxTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
<head>
<meta name="dtb:uid" content="urn:uuid:%s"/>
<meta name="dtb:depth" content="1"/>
<meta name="dtb:totalPageCount" content="0"/>
<meta name="dtb:maxPageNumber" content="0"/>
</head>
<docTitle><text>%s</text></docTitle>
<navMap>
%s
</navMap>
</ncx>`

func buildNCX(uid, title string, chapters []mdChapter) string {
	var sb strings.Builder
	for i, ch := range chapters {
		sb.WriteString(fmt.Sprintf(
			`<navPoint id="nav%d" playOrder="%d"><navLabel><text>%s</text></navLabel><content src="chapter-%d.xhtml"/></navPoint>`,
			i+1, i+1, xmlEscape(ch.Title), i+1))
	}
	return fmt.Sprintf(ncxTemplate, uid, xmlEscape(title), sb.String())
}

const opfTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="bookid">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
<dc:identifier id="bookid" opf:scheme="UUID">urn:uuid:%s</dc:identifier>
<dc:title>%s</dc:title>
<dc:language>zh-CN</dc:language>
<dc:creator>%s</dc:creator>
<meta name="generator" content="my-books"/>
</metadata>
<manifest>
%s
</manifest>
<spine toc="ncx">
%s
</spine>
</package>`

func buildOPF(uid, title string, chapters []mdChapter) string {
	var manifest, spine strings.Builder
	// 样式表与 NCX 目录必须登记进 manifest，foliate 才能正确重写并加载它们
	manifest.WriteString(`<item id="style" href="style.css" media-type="text/css"/>`)
	manifest.WriteString(`<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>`)
	for i := range chapters {
		id := fmt.Sprintf("chap%d", i+1)
		manifest.WriteString(fmt.Sprintf(`<item id="%s" href="chapter-%d.xhtml" media-type="application/xhtml+xml"/>`, id, i+1))
		spine.WriteString(fmt.Sprintf(`<itemref idref="%s"/>`, id))
	}
	return fmt.Sprintf(opfTemplate, uid, xmlEscape(title), xmlEscape("佚名"), manifest.String(), spine.String())
}
