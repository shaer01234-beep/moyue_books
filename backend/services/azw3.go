package services

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// AZW3Reader 解析并阅读 AZW3/MOBI（含 KF8 与 MOBI7）电子书。
// 支持：元数据(EXTH)、正文(未压缩/PalmDOC/Huffman)、章节(骨架/碎片表或分页标记)、
// 图片资源、封面、目录(NCX 索引或章节标题)。
type AZW3Reader struct {
	Path        string
	Title       string
	Authors     []string
	Language    string
	Description string
	Publisher   string
	PublishDate string
	ISBN        string
	Tags        []string
	CoverPath   string
	Contents    []string

	chapters []azw3Chapter
	images   map[string][]byte
	tocItems []TOCItem

	compression int // 0=未压缩 1/2=PalmDOC 17480=Huffman
	encoding    int // 文本编码（65001=UTF-8）
	isKF8       bool
	recordCount int
	hdrIdx      int
	ncxIdx      int
	huff        *huffcdicReader
	index       *azw3Index

	closed bool
}

type azw3Chapter struct {
	index int
	title string
	level int
	html  string
}

// azw3EXTH EXTH 记录
type azw3EXTH struct {
	Type    int
	Size    int
	Content []byte
}

// OpenAZW3 打开并解析 AZW3/MOBI 文件
func OpenAZW3(path string) (*AZW3Reader, error) {
	a := &AZW3Reader{Path: path, images: map[string][]byte{}, ncxIdx: -1}
	if err := a.parseAZW3(); err != nil {
		return nil, err
	}
	return a, nil
}

// parseAZW3 解析 PDB 容器 + PalmDOC/MOBI 头 + 正文压缩数据 + 章节/图片/目录
func (a *AZW3Reader) parseAZW3() error {
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return err
	}
	records, err := parsePDBRecords(data)
	if err != nil {
		return err
	}
	hdrIdx, hdr, err := findMobiHeader(records)
	if err != nil {
		return err
	}
	a.hdrIdx = hdrIdx
	a.index = &azw3Index{records: records}

	pal := records[hdrIdx] // PalmDOC 头(16B) + MOBI 头
	if len(pal) < 16 {
		return errors.New("truncated MOBI header record")
	}
	a.compression = int(binary.BigEndian.Uint16(pal[0:2]))
	recordCount := int(binary.BigEndian.Uint16(pal[8:10]))
	a.recordCount = recordCount

	mobiLength := hdrU32(hdr, 0x04)
	mobiType := hdrU32(hdr, 0x08)
	_ = mobiType
	a.encoding = int(hdrU32(hdr, 0x0C))
	version := hdrU32(hdr, 0x14)
	mobiVersion := hdrU32(hdr, 0x58)
	firstResource := hdrU32(hdr, 0x5C)
	exthFlags := hdrU32(hdr, 0x70)
	huffOff := hdrU32(hdr, 0x60)
	huffNum := hdrU32(hdr, 0x64)
	fullNameOff := hdrU32(hdr, 0x44)
	fullNameLen := hdrU32(hdr, 0x48)

	a.isKF8 = version >= 8
	a.ncxIdx = int(hdrU32(hdr, 0xE4)) // rec0+0xF4
	if a.ncxIdx != 0xFFFFFFFF {
		a.ncxIdx += hdrIdx
	} else {
		a.ncxIdx = -1
	}

	// EXTH 元数据
	var exths []azw3EXTH
	if exthFlags&0x40 != 0 {
		exths = parseEXTHRecords(hdr, mobiLength)
	}
	a.Title = exthString(exths, 503)
	if a.Title == "" {
		a.Title = exthString(exths, 127, 131)
	}
	a.Language = exthString(exths, 118)
	if a.Language == "" {
		a.Language = exthString(exths, 524)
	}
	if author := exthString(exths, 100, 125); author != "" {
		a.Authors = splitAuthor(author)
	}
	a.Description = exthString(exths, 103)
	a.Publisher = exthString(exths, 101, 102)
	a.ISBN = exthString(exths, 117)
	if a.ISBN == "" {
		a.ISBN = exthString(exths, 104) // 部分出版商的 ISBN 存于 104
	}
	a.PublishDate = exthString(exths, 106, 109)
	a.Tags = splitTags(exthString(exths, 108))

	// 标题回退：full_name 字段
	if a.Title == "" && fullNameLen > 0 && fullNameOff != 0xFFFFFFFF {
		var candidates [][]byte
		if int(fullNameOff+fullNameLen) <= len(pal) {
			candidates = append(candidates, pal[fullNameOff:fullNameOff+fullNameLen])
		}
		if int(fullNameOff+fullNameLen) <= len(hdr) {
			candidates = append(candidates, hdr[fullNameOff:fullNameOff+fullNameLen])
		}
		if int(16+fullNameOff+fullNameLen) <= len(pal) {
			candidates = append(candidates, pal[16+fullNameOff:16+fullNameOff+fullNameLen])
		}
		for _, c := range candidates {
			if t := strings.TrimSpace(decodeText(c, a.encoding)); t != "" && !strings.ContainsRune(t, '\x00') {
				a.Title = t
				break
			}
		}
	}
	if a.Title == "" {
		base := filepath.Base(a.Path)
		a.Title = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// multibyte/trailers 标记（KF8 记录末尾附加数据）
	multibyte, trailers := 0, 0
	if mobiLength >= 0xE4 && mobiVersion >= 5 && len(pal) >= 0xF4 {
		flags := int(binary.BigEndian.Uint16(pal[0xF2:0xF4]))
		multibyte = flags & 1
		for f := flags; f > 1; f >>= 1 {
			if f&2 != 0 {
				trailers++
			}
		}
	}

	// Huffman 表
	if a.compression == 17480 {
		hr := &huffcdicReader{}
		if huffOff != 0xFFFFFFFF && int(huffOff)+hdrIdx < len(records) {
			if err := hr.loadHuff(records[int(huffOff)+hdrIdx]); err != nil {
				return fmt.Errorf("HUFF: %w", err)
			}
			for i := 1; i < int(huffNum); i++ {
				idx := int(huffOff) + hdrIdx + i
				if idx >= len(records) {
					break
				}
				if err := hr.loadCdic(records[idx]); err != nil {
					return fmt.Errorf("CDIC: %w", err)
				}
			}
		}
		a.huff = hr
	}

	// 解压正文记录 1..recordCount
	var raw []byte
	for i := 1; i <= recordCount; i++ {
		if hdrIdx+i >= len(records) {
			break
		}
		rec := records[hdrIdx+i]
		for t := 0; t < trailers; t++ {
			n := sizeOfTrailingDataEntry(rec)
			if n <= 0 || n >= len(rec) {
				break
			}
			rec = rec[:len(rec)-n]
		}
		if multibyte != 0 && len(rec) > 0 {
			n := int(rec[len(rec)-1]&3) + 1
			if n >= len(rec) {
				n = len(rec)
			}
			rec = rec[:len(rec)-n]
		}
		var dec []byte
		switch a.compression {
		case 0:
			dec = rec
		case 1, 2:
			dec = decompressPalmDOC(rec)
		case 17480:
			if a.huff == nil {
				return errors.New("huffman table missing")
			}
			dec, err = a.huff.unpack(rec)
			if err != nil {
				return fmt.Errorf("huffman record %d: %w", i, err)
			}
		default:
			return fmt.Errorf("unsupported compression %d", a.compression)
		}
		raw = append(raw, dec...)
	}
	if len(raw) == 0 {
		return errors.New("no text content")
	}

	if a.isKF8 {
		if err := a.buildKF8Content(records, raw, hdr); err != nil {
			return err
		}
	} else {
		a.buildMOBI7Content(records, raw)
	}

	// 图片资源
	if firstResource != 0xFFFFFFFF {
		firstResource += uint32(hdrIdx)
		for i := int(firstResource); i < len(records); i++ {
			img := bytes.TrimRight(records[i], "\x00")
			if !looksLikeImage(img) {
				continue
			}
			n := i - int(firstResource) + 1
			a.images[fmt.Sprintf("images/%d", n)] = img
		}
	}

	a.buildTOC()
	// 填充 Contents（章节虚拟路径列表）：与 EPUBReader.Contents（spine 路径）保持字段语义一致，
	// 避免该字段永远为空而被误用。azw3 无真实文件路径，用 chapter/N 虚拟路径代替。
	a.Contents = make([]string, 0, len(a.chapters))
	for i := range a.chapters {
		a.Contents = append(a.Contents, fmt.Sprintf("chapter/%d", i))
	}
	a.CoverPath = findAZW3Cover(exths, a.images)
	return nil
}

// buildKF8Content 按 FDST 分段 + 骨架/碎片表组装章节（每个骨架=一个 xhtml 文件）
func (a *AZW3Reader) buildKF8Content(records [][]byte, raw []byte, hdr []byte) error {
	// 注：FDST 表（rec0+0xC0 / 0xC4）定义了多 flow 的分段边界，但骨架/碎片索引里的
	// tagMap[6] 偏移是相对整个 rawML 的绝对偏移，直接使用完整 raw 即可定位。
	skelIdx := int(hdrU32(hdr, 0xEC)) // rec0+0xFC
	fragIdx := int(hdrU32(hdr, 0xE8)) // rec0+0xF8
	if skelIdx != 0xFFFFFFFF {
		skelIdx += a.hdrIdx
	}
	if fragIdx != 0xFFFFFFFF {
		fragIdx += a.hdrIdx
	}

	// 主文本必须用完整 rawML：骨架/碎片的 tagMap[6] 偏移是相对整个 rawML 的绝对偏移，
	// 若按 FDST 切成单个 flow 会导致偏移整体错位（正文碎片会被插到 <html> 框架之前，
	// 章节丢失 <head> 里的 CSS <link>，渲染出来就是一坨无样式的裸文本）。
	text := raw

	skeltbl, _ := a.index.getIndexData(skelIdx)
	fragtbl, _ := a.index.getIndexData(fragIdx)

	// 骨架与碎片一一对应：每个骨架消耗 skel.numFrag 个碎片（按骨架顺序依次分配）
	fragptr := 0
	for _, skel := range skeltbl {
		if len(skel.tagMap[1]) == 0 || len(skel.tagMap[6]) < 2 {
			continue
		}
		fragcnt := skel.tagMap[1][0]
		skelpos := skel.tagMap[6][0]
		skellen := skel.tagMap[6][1]
		if skelpos+skellen > len(text) {
			continue
		}
		// 骨架数据 = HTML 框架（<html><head>...<body> 内容位置留空 </body></html>）
		skeleton := append([]byte{}, text[skelpos:skelpos+skellen]...)
		for k := 0; k < fragcnt && fragptr < len(fragtbl); k++ {
			f := fragtbl[fragptr]
			fragptr++
			if len(f.tagMap[6]) < 2 {
				continue
			}
			fOffset := f.tagMap[6][0] // 碎片数据相对（骨架结束位置）的偏移
			fLength := f.tagMap[6][1] // 碎片数据长度
			// 碎片数据在流中的绝对位置 = 骨架结束 + 碎片自身偏移
			dataPos := skelpos + skellen + fOffset
			if fLength <= 0 || dataPos+fLength > len(text) {
				continue
			}
			slice := text[dataPos : dataPos+fLength]
			// 插入位置取自碎片 entry 的 name 字段（十进制字符串，流中的绝对偏移）。
			// 注意不能用 tagMap[6][0]——那是碎片数据的读取位置，两者语义不同。
			insertAbs := skelpos
			if n, err := strconv.Atoi(strings.TrimSpace(string(f.text))); err == nil {
				insertAbs = n
			}
			rel := insertAbs - skelpos // 转成相对骨架的偏移
			if rel < 0 {
				rel = 0
			}
			if rel > len(skeleton) {
				rel = len(skeleton)
			}
			skeleton = append(append(append([]byte{}, skeleton[:rel]...), slice...), skeleton[rel:]...)
		}
		html := normalizeKF8HTML(skeleton, a.encoding)
		title, level := extractAZW3Heading(html)
		a.chapters = append(a.chapters, azw3Chapter{
			index: len(a.chapters),
			title: title,
			level: level,
			html:  html,
		})
	}
	return nil
}

// buildMOBI7Content 经典 MOBI7：拼接正文并按 <mbp:pagebreak> 切分章节
func (a *AZW3Reader) buildMOBI7Content(records [][]byte, raw []byte) {
	text := decodeText(raw, a.encoding)
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "<"); idx > 0 && idx < 4096 {
		text = text[idx:]
	}
	if a.compression == 1 || a.compression == 2 {
		// PalmDOC 记录可能以 2 字节长度前缀开头，且首个记录含 3 字节长度头
		if len(text) > 3 && text[0] == '<' == false {
			text = text[2:]
		}
	}
	if !strings.Contains(strings.ToLower(text), "<html") {
		text = "<html><body>" + text + "</body></html>"
	}
	segments := azwPageBreakRE.Split(text, -1)
	if len(segments) <= 1 {
		html := normalizeKF8HTML([]byte(text), a.encoding)
		title, level := extractAZW3Heading(html)
		a.chapters = append(a.chapters, azw3Chapter{index: 0, title: title, level: level, html: html})
		return
	}
	for i, seg := range segments {
		html := normalizeKF8HTML([]byte(seg), a.encoding)
		title, level := extractAZW3Heading(html)
		a.chapters = append(a.chapters, azw3Chapter{index: i, title: title, level: level, html: html})
	}
	// 图片：按分隔符拆分记录（MOBI7 每记录可含多张图）
	imgNo := 1
	for _, rec := range records {
		for _, part := range bytes.Split(rec, imageSeparator) {
			if looksLikeImage(part) {
				a.images[fmt.Sprintf("images/%d", imgNo)] = part
				imgNo++
			}
		}
	}
}

// buildTOC 生成目录：KF8 优先使用 NCX 索引（真实层级与标题），否则回退章节标题
func (a *AZW3Reader) buildTOC() {
	a.tocItems = nil
	if a.isKF8 {
		a.tocItems = a.buildTOCFromNCX()
	}
	if len(a.tocItems) == 0 {
		a.tocItems = a.fallbackTOC()
	}
}

// buildTOCFromNCX 从 NCX 索引构建目录（KF8）
func (a *AZW3Reader) buildTOCFromNCX() []TOCItem {
	if a.ncxIdx < 0 {
		return nil
	}
	outtbl, ctocText := a.index.getIndexData(a.ncxIdx)
	var items []TOCItem
	for _, e := range outtbl {
		if len(e.tagMap[3]) == 0 {
			continue
		}
		rawTitle, ok := ctocText[e.tagMap[3][0]]
		if !ok {
			continue
		}
		title := strings.TrimSpace(stripTags(decodeText(rawTitle, a.encoding)))
		if title == "" {
			continue
		}
		level := 1
		if len(e.tagMap[4]) > 0 {
			level = e.tagMap[4][0] + 1
		}
		if level > 4 {
			level = 4
		}
		part := -1
		if len(e.tagMap[6]) > 0 {
			part = e.tagMap[6][0]
		}
		if part < 0 || part >= len(a.chapters) {
			continue
		}
		items = append(items, TOCItem{
			Index: part,
			Title: title,
			Level: level,
			Path:  fmt.Sprintf("chapter/%d", part),
		})
	}
	return items
}

// fallbackTOC 以章节（part）标题构建目录
func (a *AZW3Reader) fallbackTOC() []TOCItem {
	var items []TOCItem
	for i, ch := range a.chapters {
		title := ch.title
		if title == "" {
			title = fmt.Sprintf("章节 %d", i+1)
		}
		level := ch.level
		if level < 1 {
			level = 1
		}
		items = append(items, TOCItem{
			Index: i,
			Title: title,
			Level: level,
			Path:  fmt.Sprintf("chapter/%d", i),
		})
	}
	return items
}

// GetContent 返回第 index 个章节的 HTML
func (a *AZW3Reader) GetContent(index int) ([]byte, error) {
	if a.closed || index < 0 || index >= len(a.chapters) {
		return nil, fmt.Errorf("chapter %d out of range", index)
	}
	return []byte(a.chapters[index].html), nil
}

// GetResource 返回资源（images/N）
func (a *AZW3Reader) GetResource(path string) ([]byte, error) {
	if a.closed {
		return nil, errors.New("reader closed")
	}
	if data, ok := a.images[path]; ok {
		return data, nil
	}
	if p := strings.TrimPrefix(path, "images/"); p != path {
		if n, err := strconv.Atoi(p); err == nil {
			if data, ok := a.images[fmt.Sprintf("images/%d", n)]; ok {
				return data, nil
			}
		}
	}
	// 兜底：前端以章节虚拟路径（chapter/N）为基准解析相对 src（images/M）时
	// 会得到 chapter/images/M，取最后一段再匹配 images/N
	if i := strings.LastIndex(path, "/"); i >= 0 {
		if n, err := strconv.Atoi(path[i+1:]); err == nil {
			if data, ok := a.images[fmt.Sprintf("images/%d", n)]; ok {
				return data, nil
			}
		}
	}
	return nil, fmt.Errorf("resource %q not found", path)
}

// GetChapterList 返回目录列表（map 形式，供前端渲染）
func (a *AZW3Reader) GetChapterList() []map[string]interface{} {
	items := a.TOCItems()
	list := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		list = append(list, map[string]interface{}{
			"index": it.Index,
			"title": it.Title,
			"level": it.Level,
			"path":  it.Path,
		})
	}
	return list
}

// TOCItems 返回内部目录列表（TOCItem 形式）
func (a *AZW3Reader) TOCItems() []TOCItem {
	if a.tocItems == nil {
		a.buildTOC()
	}
	return a.tocItems
}

// GetCover 返回封面图片二进制
func (a *AZW3Reader) GetCover() ([]byte, error) {
	if a.CoverPath == "" {
		return nil, errors.New("no cover found")
	}
	if data, ok := a.images[a.CoverPath]; ok {
		return data, nil
	}
	if p := strings.TrimPrefix(a.CoverPath, "images/"); p != a.CoverPath {
		if n, err := strconv.Atoi(p); err == nil {
			if data, ok := a.images[fmt.Sprintf("images/%d", n)]; ok {
				return data, nil
			}
		}
	}
	return nil, errors.New("cover not found")
}

// Close 关闭阅读器
func (a *AZW3Reader) Close() {
	a.closed = true
}

// ---------------------------------------------------------------------------
// PDB 容器解析
// ---------------------------------------------------------------------------

// parsePDBRecords 解析 PDB 记录表（标准 78 字节 header 后紧跟记录表，每项 8 字节）
// 兼容少数文件记录表从 80 字节开始（78:80 处存记录表偏移）
func parsePDBRecords(data []byte) ([][]byte, error) {
	if len(data) < 80 {
		return nil, errors.New("file too small")
	}
	numRecords := int(binary.BigEndian.Uint16(data[76:78]))
	if numRecords <= 0 || 78+numRecords*8 > len(data) {
		return nil, errors.New("invalid PDB record table")
	}
	tableOff := 78
	if off := int(binary.BigEndian.Uint16(data[78:80])); off >= 80 && off+numRecords*8 <= len(data) {
		// 验证候选偏移：首个记录偏移应位于记录表之后且不越界
		if first := int(binary.BigEndian.Uint32(data[off:])); first >= off+numRecords*8 && first < len(data) {
			tableOff = off
		}
	}
	offsets := make([]int, numRecords+1)
	for i := 0; i < numRecords; i++ {
		offsets[i] = int(binary.BigEndian.Uint32(data[tableOff+i*8 : tableOff+i*8+4]))
	}
	offsets[numRecords] = len(data)
	records := make([][]byte, numRecords)
	for i := 0; i < numRecords; i++ {
		if offsets[i+1] < offsets[i] || offsets[i] > len(data) || offsets[i+1] > len(data) {
			return nil, errors.New("bad PDB record offset")
		}
		records[i] = data[offsets[i]:offsets[i+1]]
	}
	return records, nil
}

// findMobiHeader 在记录中定位 MOBI magic，返回记录索引与 magic 起始的头部数据
func findMobiHeader(records [][]byte) (int, []byte, error) {
	found := -1
	for i, rec := range records {
		pos := bytes.Index(rec, []byte("MOBI"))
		if pos < 0 || pos+20 > len(rec) {
			continue
		}
		if i == 0 && pos == 16 {
			return i, rec[pos:], nil
		}
		if found == -1 {
			found = i
		}
		if pos+12 <= len(rec) {
			t := int(binary.BigEndian.Uint32(rec[pos+8 : pos+12]))
			if t == 232 || t == 2 {
				return i, rec[pos:], nil
			}
		}
	}
	if found != -1 {
		pos := bytes.Index(records[found], []byte("MOBI"))
		return found, records[found][pos:], nil
	}
	return -1, nil, errors.New("not a MOBI/AZW3 file")
}

// ---------------------------------------------------------------------------
// EXTH / 元数据
// ---------------------------------------------------------------------------

// parseEXTHRecords 解析 EXTH 记录（flags 在 MOBI magic+0x70，EXTH 数据在 hdr[headerLen:]）
func parseEXTHRecords(hdr []byte, headerLen uint32) []azw3EXTH {
	if hdrU32(hdr, 0x70)&0x40 == 0 {
		return nil
	}
	off := int(headerLen)
	if off < 0x70 || off+12 > len(hdr) {
		return nil
	}
	if string(hdr[off:off+4]) != "EXTH" {
		return nil
	}
	exthLength := int(binary.BigEndian.Uint32(hdr[off+4:]))
	count := int(binary.BigEndian.Uint32(hdr[off+8:]))
	if exthLength < 12 || off+exthLength > len(hdr) {
		exthLength = len(hdr) - off
	}
	exthEnd := off + exthLength
	pos := off + 12
	var exths []azw3EXTH
	for i := 0; i < count && pos+8 <= exthEnd; i++ {
		typ := int(binary.BigEndian.Uint32(hdr[pos:]))
		size := int(binary.BigEndian.Uint32(hdr[pos+4:]))
		if size < 8 || pos+size > exthEnd {
			break
		}
		exths = append(exths, azw3EXTH{Type: typ, Size: size, Content: hdr[pos+8 : pos+size]})
		pos += size
	}
	return exths
}

func exthString(exths []azw3EXTH, types ...int) string {
	for _, t := range types {
		for _, e := range exths {
			if e.Type == t {
				s := strings.TrimSpace(string(e.Content))
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func exthInt(exths []azw3EXTH, types ...int) int {
	for _, t := range types {
		for _, e := range exths {
			if e.Type == t {
				if n, err := strconv.Atoi(strings.TrimSpace(string(e.Content))); err == nil {
					return n
				}
				if len(e.Content) == 4 {
					return int(binary.BigEndian.Uint32(e.Content))
				}
			}
		}
	}
	return -1
}

// findAZW3Cover 确定封面资源 key（images/N）
func findAZW3Cover(exths []azw3EXTH, images map[string][]byte) string {
	if uri := exthString(exths, 129); uri != "" {
		if m := regexp.MustCompile(`kindle:embed:(\d+)`).FindStringSubmatch(uri); len(m) > 1 {
			key := "images/" + m[1]
			if _, ok := images[key]; ok {
				return key
			}
		}
	}
	for _, typ := range []int{201, 202, 121, 122} {
		if v := exthInt(exths, typ); v > 0 {
			for _, key := range []string{fmt.Sprintf("images/%d", v), fmt.Sprintf("images/%d", v+1), fmt.Sprintf("images/%d", v-1)} {
				if _, ok := images[key]; ok {
					return key
				}
			}
		}
	}
	if _, ok := images["images/1"]; ok {
		return "images/1"
	}
	return ""
}

// ---------------------------------------------------------------------------
// 压缩解码
// ---------------------------------------------------------------------------

// sizeOfTrailingDataEntry 读取记录末尾附加数据的长度（高位置位=段起点）
func sizeOfTrailingDataEntry(data []byte) int {
	num := 0
	start := len(data) - 4
	if start < 0 {
		start = 0
	}
	for _, b := range data[start:] {
		if b&0x80 != 0 {
			num = 0
		}
		num = (num << 7) | int(b&0x7F)
	}
	return num
}

// decompressPalmDOC PalmDOC LZ77 解压（compression 1/2）
func decompressPalmDOC(data []byte) []byte {
	var out []byte
	p := 0
	for p < len(data) {
		c := int(data[p])
		p++
		switch {
		case c >= 1 && c <= 8:
			if p+c <= len(data) {
				out = append(out, data[p:p+c]...)
				p += c
			}
		case c < 128:
			out = append(out, byte(c))
		case c >= 192:
			out = append(out, ' ', byte(c^128))
		default:
			if p >= len(data) {
				break
			}
			c = (c << 8) | int(data[p])
			p++
			m := (c >> 3) & 0x07FF
			n := (c & 7) + 3
			if m == 0 {
				m = 1
			}
			l := len(out)
			if l < m {
				break
			}
			if m > n {
				if l >= m+n {
					out = append(out, out[l-m:l-m+n]...)
				}
			} else {
				ref := out[l-m]
				for k := 0; k < n; k++ {
					out = append(out, ref)
				}
			}
		}
	}
	return out
}

// huffcdicReader MOBI Huffman 解码器（HUFF + CDIC）
type huffcdicReader struct {
	codeLen [256]int
	term    [256]bool
	maxcode [256]uint32
	mincode [33]uint32
	maxc2   [33]uint32
	dict    [][]byte
	dictSet []bool
}

func (r *huffcdicReader) loadHuff(huff []byte) error {
	if len(huff) < 8 || !bytes.Equal(huff[0:8], []byte("HUFF\x00\x00\x00\x18")) {
		return errors.New("invalid HUFF header")
	}
	off1 := int(binary.BigEndian.Uint32(huff[8:12]))
	off2 := int(binary.BigEndian.Uint32(huff[12:16]))
	if off1+256*4 > len(huff) || off2+64*4 > len(huff) {
		return errors.New("HUFF tables out of range")
	}
	for i := 0; i < 256; i++ {
		v := binary.BigEndian.Uint32(huff[off1+i*4:])
		codelen := int(v & 0x1F)
		if codelen == 0 {
			return errors.New("bad huffman codelen")
		}
		term := v&0x80 != 0
		if codelen <= 8 && !term {
			return errors.New("bad huffman term")
		}
		maxcode := (v >> 8) + 1
		if codelen < 32 {
			maxcode <<= 32 - codelen
		}
		maxcode--
		r.codeLen[i] = codelen
		r.term[i] = term
		r.maxcode[i] = maxcode
	}
	r.mincode[0] = 0
	r.maxc2[0] = 0
	for cl := 1; cl <= 32; cl++ {
		minc := binary.BigEndian.Uint32(huff[off2+(cl-1)*8:])
		maxc := binary.BigEndian.Uint32(huff[off2+(cl-1)*8+4:])
		if cl < 32 {
			r.mincode[cl] = minc << (32 - cl)
			r.maxc2[cl] = ((maxc + 1) << (32 - cl)) - 1
		} else {
			r.mincode[cl] = minc
			r.maxc2[cl] = maxc
		}
	}
	return nil
}

func (r *huffcdicReader) loadCdic(cdic []byte) error {
	if len(cdic) < 16 || !bytes.Equal(cdic[0:8], []byte("CDIC\x00\x00\x00\x10")) {
		return errors.New("invalid CDIC header")
	}
	phrases := int(binary.BigEndian.Uint32(cdic[8:12]))
	bits := int(binary.BigEndian.Uint32(cdic[12:16]))
	n := 1 << bits
	if remaining := phrases - len(r.dict); n > remaining {
		n = remaining
	}
	for i := 0; i < n; i++ {
		off := int(binary.BigEndian.Uint16(cdic[16+i*2:]))
		if 16+off+2 > len(cdic) {
			return errors.New("CDIC phrase offset out of range")
		}
		blen := int(binary.BigEndian.Uint16(cdic[16+off:]))
		end := 18 + off + (blen & 0x7FFF)
		if end > len(cdic) {
			return errors.New("CDIC phrase data out of range")
		}
		slice := cdic[18+off : end]
		flag := blen&0x8000 != 0
		r.dict = append(r.dict, slice)
		r.dictSet = append(r.dictSet, flag)
	}
	return nil
}

func (r *huffcdicReader) unpack(data []byte) ([]byte, error) {
	bitsleft := len(data) * 8
	padded := make([]byte, len(data)+8)
	copy(padded, data)
	pos := 0
	x := binary.BigEndian.Uint64(padded[pos : pos+8])
	n := 32
	var out []byte
	for {
		if n <= 0 {
			pos += 4
			if pos+8 > len(padded) {
				break
			}
			x = binary.BigEndian.Uint64(padded[pos : pos+8])
			n += 32
		}
		code := uint32(x >> n)
		idx := int(code >> 24)
		codelen := r.codeLen[idx]
		term := r.term[idx]
		maxcode := r.maxcode[idx]
		if !term {
			for code < r.mincode[codelen] {
				codelen++
			}
			maxcode = r.maxc2[codelen]
		}
		n -= codelen
		bitsleft -= codelen
		if bitsleft < 0 {
			break
		}
		rr := (maxcode - code) >> (32 - codelen)
		di := int(rr)
		if di < 0 || di >= len(r.dict) {
			return nil, errors.New("huffman dictionary index out of range")
		}
		slice := r.dict[di]
		if !r.dictSet[di] {
			r.dict[di] = nil
			expanded, err := r.unpack(slice)
			if err != nil {
				return nil, err
			}
			r.dict[di] = expanded
			r.dictSet[di] = true
			slice = expanded
		}
		out = append(out, slice...)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// INDX 索引（骨架表 / 碎片表 / NCX）
// ---------------------------------------------------------------------------

type azw3Index struct {
	records [][]byte
}

type indxEntry struct {
	text   []byte
	tagMap map[int][]int
}

func (ix *azw3Index) getIndexData(idx int) ([]indxEntry, map[int][]byte) {
	if idx < 0 || idx >= len(ix.records) {
		return nil, nil
	}
	data := ix.records[idx]
	hdr := parseINDXHeader(data)
	if hdr == nil {
		return nil, nil
	}
	indexCount := hdr["count"]
	ctocText := map[int][]byte{}
	recOff := 0
	off := idx + indexCount + 1
	for j := 0; j < hdr["nctoc"]; j++ {
		if off+j >= len(ix.records) {
			break
		}
		cd := readCTOC(ix.records[off+j])
		for k, v := range cd {
			ctocText[k+recOff] = v
		}
		recOff += 0x10000
	}
	tagSectionStart := hdr["len"]
	controlByteCount, tagTable := readTagSection(tagSectionStart, data)
	var out []indxEntry
	for i := idx + 1; i < idx+1+indexCount; i++ {
		if i >= len(ix.records) {
			break
		}
		extra := ix.records[i]
		hdrinfo := parseINDXHeader(extra)
		if hdrinfo == nil {
			continue
		}
		idxtPos := hdrinfo["start"]
		entryCount := hdrinfo["count"]
		if idxtPos+4+2*entryCount > len(extra) {
			continue
		}
		idxPositions := make([]int, 0, entryCount+1)
		for j := 0; j < entryCount; j++ {
			idxPositions = append(idxPositions, int(binary.BigEndian.Uint16(extra[idxtPos+4+2*j:])))
		}
		idxPositions = append(idxPositions, idxtPos)
		for j := 0; j < entryCount; j++ {
			startPos := idxPositions[j]
			endPos := idxPositions[j+1]
			if startPos >= len(extra) || startPos+1 > endPos || endPos > len(extra) {
				continue
			}
			textLength := int(extra[startPos])
			if startPos+1+textLength > endPos {
				continue
			}
			text := extra[startPos+1 : startPos+1+textLength]
			tm := getTagMap(controlByteCount, tagTable, extra, startPos+1+textLength, endPos)
			out = append(out, indxEntry{text: text, tagMap: tm})
		}
	}
	return out, ctocText
}

func parseINDXHeader(data []byte) map[string]int {
	if len(data) < 56 || string(data[0:4]) != "INDX" {
		return nil
	}
	words := []string{"len", "nul1", "type", "gen", "start", "count", "code", "lng", "total", "ordt", "ligt", "nligt", "nctoc"}
	hdr := map[string]int{}
	for i, w := range words {
		hdr[w] = int(binary.BigEndian.Uint32(data[4+i*4:]))
	}
	return hdr
}

func readTagSection(start int, data []byte) (int, [][4]byte) {
	controlByteCount := 0
	var tagTable [][4]byte
	if start+12 <= len(data) && string(data[start:start+4]) == "TAGX" {
		firstEntryOffset := int(binary.BigEndian.Uint32(data[start+4:]))
		controlByteCount = int(binary.BigEndian.Uint32(data[start+8:]))
		for i := 12; i < firstEntryOffset; i += 4 {
			pos := start + i
			if pos+4 > len(data) {
				break
			}
			tagTable = append(tagTable, [4]byte{data[pos], data[pos+1], data[pos+2], data[pos+3]})
		}
	}
	return controlByteCount, tagTable
}

func getTagMap(controlByteCount int, tagTable [][4]byte, entryData []byte, startPos, endPos int) map[int][]int {
	type pendingTag struct {
		tag, valueCount, valueBytes, valuesPerEntry int
	}
	var tags []pendingTag
	tagHashMap := map[int][]int{}
	controlByteIndex := 0
	dataStart := startPos + controlByteCount
	if dataStart > len(entryData) {
		dataStart = len(entryData)
	}
	for _, t := range tagTable {
		tag, valuesPerEntry, mask, endFlag := int(t[0]), int(t[1]), int(t[2]), int(t[3])
		if endFlag == 0x01 {
			controlByteIndex++
			continue
		}
		if startPos+controlByteIndex >= len(entryData) {
			continue
		}
		cbyte := int(entryData[startPos+controlByteIndex])
		value := cbyte & mask
		if value != 0 {
			if value == mask {
				if bitsCount(mask) > 1 {
					consumed, v := getVariableWidthValue(entryData, dataStart)
					dataStart += consumed
					tags = append(tags, pendingTag{tag, -1, v, valuesPerEntry})
				} else {
					tags = append(tags, pendingTag{tag, 1, -1, valuesPerEntry})
				}
			} else {
				for mask&0x01 == 0 {
					mask >>= 1
					value >>= 1
				}
				tags = append(tags, pendingTag{tag, value, -1, valuesPerEntry})
			}
		}
	}
	for _, pt := range tags {
		var values []int
		if pt.valueCount != -1 {
			for c := 0; c < pt.valueCount; c++ {
				for v := 0; v < pt.valuesPerEntry; v++ {
					consumed, val := getVariableWidthValue(entryData, dataStart)
					dataStart += consumed
					values = append(values, val)
				}
			}
		} else {
			totalConsumed := 0
			for totalConsumed < pt.valueBytes {
				consumed, val := getVariableWidthValue(entryData, dataStart)
				dataStart += consumed
				totalConsumed += consumed
				values = append(values, val)
			}
		}
		tagHashMap[pt.tag] = values
	}
	return tagHashMap
}

func getVariableWidthValue(data []byte, offset int) (int, int) {
	value := 0
	consumed := 0
	for offset+consumed < len(data) {
		v := data[offset+consumed]
		consumed++
		value = (value << 7) | int(v&0x7F)
		if v&0x80 != 0 {
			break
		}
	}
	return consumed, value
}

func bitsCount(v int) int {
	c := 0
	for v != 0 {
		c += v & 1
		v >>= 1
	}
	return c
}

func readCTOC(txtdata []byte) map[int][]byte {
	ctoc := map[int][]byte{}
	offset := 0
	for offset < len(txtdata) {
		if txtdata[offset] == 0 {
			break
		}
		idxOffs := offset
		consumed, ilen := getVariableWidthValue(txtdata, offset)
		offset += consumed
		if offset+ilen > len(txtdata) {
			break
		}
		name := txtdata[offset : offset+ilen]
		offset += ilen
		ctoc[idxOffs] = name
	}
	return ctoc
}

// ---------------------------------------------------------------------------
// HTML 处理
// ---------------------------------------------------------------------------

var azwEmbedRE = regexp.MustCompile(`kindle:embed:(\d+)(\?[^"'\s]*)?`)
var azwPageBreakRE = regexp.MustCompile(`(?i)<mbp:pagebreak[^>]*/?>`)
var azwHeadingRE = regexp.MustCompile(`(?is)<(h[1-6])([^>]*)>(.*?)</h[1-6]>`)
var azwTitleAttrRE = regexp.MustCompile(`(?i)title\s*=\s*"([^"]*)"`)
var azwTitleTagRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// normalizeKF8HTML 将原始 KF8 文本转换为可渲染 HTML
func normalizeKF8HTML(data []byte, encoding int) string {
	text := decodeText(data, encoding)
	if idx := strings.Index(text, "<"); idx > 0 && idx < 4096 {
		text = text[idx:]
	}
	// 图片引用 → 资源路径 images/N（去掉前导零）
	text = azwEmbedRE.ReplaceAllStringFunc(text, func(m string) string {
		sm := azwEmbedRE.FindStringSubmatch(m)
		if len(sm) < 2 {
			return m
		}
		n, err := strconv.Atoi(sm[1])
		if err != nil {
			return m
		}
		return fmt.Sprintf("images/%d", n)
	})
	text = strings.ReplaceAll(text, "<mbp:pagebreak/>", "<div class=\"azw-pagebreak\"></div>")
	text = strings.ReplaceAll(text, "<mbp:pagebreak>", "<div class=\"azw-pagebreak\">")
	text = strings.ReplaceAll(text, "</mbp:pagebreak>", "</div>")
	if !strings.Contains(strings.ToLower(text), "<html") {
		text = "<html><body>" + text + "</body></html>"
	}
	return text
}

// extractAZW3Heading 从章节 HTML 提取标题与层级（h 标签的 title 属性或文本，回退 <title>）
func extractAZW3Heading(doc string) (string, int) {
	for _, m := range azwHeadingRE.FindAllStringSubmatch(doc, -1) {
		level := int(m[1][1] - '0')
		if level < 1 || level > 6 {
			level = 1
		}
		if len(m[2]) > 0 {
			if tm := azwTitleAttrRE.FindStringSubmatch(m[2]); len(tm) > 1 {
				if t := strings.TrimSpace(tm[1]); t != "" {
					return t, level
				}
			}
		}
		if t := strings.TrimSpace(stripTags(m[3])); t != "" {
			return t, level
		}
		return "", level
	}
	if m := azwTitleTagRE.FindStringSubmatch(doc); len(m) > 1 {
		if t := strings.TrimSpace(stripTags(m[1])); t != "" {
			return t, 1
		}
	}
	return "", 1
}

// ---------------------------------------------------------------------------
// 图片识别 / 编码
// ---------------------------------------------------------------------------

var imageSeparator = []byte{0xE9, 0x8E, 0x0D, 0x0A}

func looksLikeImage(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	switch {
	case data[0] == 0xFF && data[1] == 0xD8: // JPEG
		return true
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47: // PNG
		return true
	case data[0] == 'G' && data[1] == 'I' && data[2] == 'F': // GIF
		return true
	case data[0] == 'B' && data[1] == 'M': // BMP
		return true
	}
	return false
}

// decodeText 按 MOBI codepage 解码文本
func decodeText(data []byte, encoding int) string {
	switch encoding {
	case 65001:
		return string(data)
	case 1252, 0:
		return decodeCP1252(data)
	default:
		return decodeCP1252(data)
	}
}

var cp1252Map = map[byte]rune{
	0x80: '\u20AC', 0x82: '\u201A', 0x83: '\u0192', 0x84: '\u201E',
	0x85: '\u2026', 0x86: '\u2020', 0x87: '\u2021', 0x88: '\u02C6',
	0x89: '\u2030', 0x8A: '\u0160', 0x8B: '\u2039', 0x8C: '\u0152',
	0x8E: '\u017D', 0x91: '\u2018', 0x92: '\u2019', 0x93: '\u201C',
	0x94: '\u201D', 0x95: '\u2022', 0x96: '\u2013', 0x97: '\u2014',
	0x98: '\u02DC', 0x99: '\u2122', 0x9A: '\u0161', 0x9B: '\u203A',
	0x9C: '\u0153', 0x9E: '\u017E', 0x9F: '\u0178',
}

func decodeCP1252(data []byte) string {
	var sb strings.Builder
	sb.Grow(len(data))
	for _, b := range data {
		if b < 0x80 || (b >= 0xA0 && b <= 0xFF) {
			sb.WriteByte(b)
		} else if r, ok := cp1252Map[b]; ok {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('\uFFFD')
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// 元数据 / 作者标签拆分
// ---------------------------------------------------------------------------

func splitAuthor(s string) []string {
	separators := []string{";", "，", ",", "、", "|"}
	for _, sep := range separators {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			var out []string
			for _, p := range parts {
				if p = strings.TrimSpace(p); p != "" {
					out = append(out, p)
				}
			}
			return out
		}
	}
	if s = strings.TrimSpace(s); s != "" {
		return []string{s}
	}
	return nil
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '、' || r == ' '
	})
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// hdrU32 读取 MOBI magic 相对偏移处的大端 uint32（越界返回 0）
func hdrU32(hdr []byte, off uint32) uint32 {
	if int(off)+4 > len(hdr) {
		return 0
	}
	return binary.BigEndian.Uint32(hdr[off : off+4])
}

// ---------------------------------------------------------------------------
// 元数据 / 目录提取（供 handlers 调用）
// ---------------------------------------------------------------------------

func extractAZW3Metadata(path string) (BookMetadata, error) {
	a, err := OpenAZW3(path)
	if err != nil {
		return BookMetadata{}, err
	}
	defer a.Close()
	var cover []byte
	if cd, cerr := a.GetCover(); cerr == nil {
		cover = cd
	}
	return BookMetadata{
		Title:       a.Title,
		Authors:     a.Authors,
		Language:    a.Language,
		Description: a.Description,
		Publisher:   a.Publisher,
		PublishDate: a.PublishDate,
		ISBN:        a.ISBN,
		Tags:        a.Tags,
		CoverData:   cover,
	}, nil
}

func extractAZW3TOC(path string) ([]TOCItem, error) {
	a, err := OpenAZW3(path)
	if err != nil {
		return nil, err
	}
	defer a.Close()
	return a.TOCItems(), nil
}
