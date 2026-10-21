package services

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

// buildTestAZW3 构造一个最小的 KF8/AZW3 文件用于测试解析器
func buildTestAZW3(t *testing.T) string {
	t.Helper()

	// ---- 章节 HTML（用 pagebreak 分割，MOBI7 风格）----
	chapters := []string{
		"<html><head><title>第一章 起点</title></head><body><p>Hello <b>AZW3</b></p><img src=\"kindle:embed:0001\"/></body></html><mbp:pagebreak/>",
		"<html><head><title>第二章 发展</title></head><body><p>World</p></body></html><mbp:pagebreak/>",
		"<html><head><title>第三章 结局</title></head><body><p>Foo</p></body></html>",
	}
	var records [][]byte

	// ---- record 0: PalmDOC header + MOBI header + EXTH + full name ----
	mobiHdr := make([]byte, 0x100)
	copy(mobiHdr[0:4], "MOBI")
	binary.BigEndian.PutUint32(mobiHdr[4:8], 0x100)      // header length
	binary.BigEndian.PutUint32(mobiHdr[8:12], 232)       // KF8 type
	binary.BigEndian.PutUint32(mobiHdr[0x0C:0x10], 65001) // UTF-8 encoding
	binary.BigEndian.PutUint32(mobiHdr[0x14:0x18], 6)    // version = MOBI7（无 INDX 索引，走 pagebreak 分割）
	binary.BigEndian.PutUint32(mobiHdr[0x44:0x48], 0x170) // full name offset（EXTH 之后）
	binary.BigEndian.PutUint32(mobiHdr[0x48:0x4C], 9)     // full name length "TestBook"
	binary.BigEndian.PutUint32(mobiHdr[0x5C:0x60], 4)     // first image index = record 4
	binary.BigEndian.PutUint32(mobiHdr[0x70:0x74], 0x40)  // EXTH flags
	binary.BigEndian.PutUint32(mobiHdr[0xA8:0xAC], 1)     // first content record
	binary.BigEndian.PutUint32(mobiHdr[0xAC:0xB0], 3)     // last content record

	// EXTH：title(131), author(125), language(118), cover offset(121)
	exthBody := []byte{}
	exthBody = append(exthBody, exthRec(131, "Test Book")...)
	exthBody = append(exthBody, exthRec(125, "Test Author")...)
	exthBody = append(exthBody, exthRec(118, "zh")...)
	co := make([]byte, 4)
	binary.BigEndian.PutUint32(co, 0) // cover offset = 0（第一张图）
	exthBody = append(exthBody, exthRec(121, string(co))...)
	exth := make([]byte, 12)
	copy(exth[0:4], "EXTH")
	binary.BigEndian.PutUint32(exth[4:8], 12+uint32(len(exthBody)))
	binary.BigEndian.PutUint32(exth[8:12], 4)
	exth = append(exth, exthBody...)

	rec0 := make([]byte, 16)
	binary.BigEndian.PutUint16(rec0[0:2], 2)     // compression = PalmDOC
	binary.BigEndian.PutUint16(rec0[8:10], 3)   // record count = 正文记录数
	binary.BigEndian.PutUint32(rec0[4:8], 0xCA9) // mobi length
	rec0 = append(rec0, mobiHdr...)
	rec0 = append(rec0, exth...)
	rec0 = append(rec0, []byte("TestBook")...)
	records = append(records, rec0)

	// ---- record 1..3: PalmDOC 压缩的章节 ----
	for _, ch := range chapters {
		records = append(records, palmDOCCompress([]byte(ch)))
	}

	// ---- record 4: 图片（伪 JPEG）----
	img := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	records = append(records, img)

	// ---- PDB 容器（header 80 字节，记录表从 offset 80 开始）----
	pdb := make([]byte, 80)
	copy(pdb[0:32], "TestBook")
	copy(pdb[60:64], "BOOK")
	copy(pdb[64:68], "MOBI")
	binary.BigEndian.PutUint16(pdb[76:78], uint16(len(records))) // numRecords
	binary.BigEndian.PutUint16(pdb[78:80], 80)                   // record list offset
	pdb = append(pdb, make([]byte, 8*len(records))...)
	off := 80 + 8*len(records)
	for i, rec := range records {
		binary.BigEndian.PutUint32(pdb[80+i*8:84+i*8], uint32(off))
		pdb[84+i*8] = 0
		off += len(rec)
	}
	for _, rec := range records {
		pdb = append(pdb, rec...)
	}

	path := t.TempDir() + "/test.azw3"
	if err := os.WriteFile(path, pdb, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func exthRec(typ int, data string) []byte {
	rec := make([]byte, 8)
	binary.BigEndian.PutUint32(rec[0:4], uint32(typ))
	binary.BigEndian.PutUint32(rec[4:8], uint32(8+len(data)))
	return append(rec, []byte(data)...)
}

// palmDOCCompress 用字面量方式构造 PalmDOC 压缩数据（用于测试解压）
// 注意：0x01-0x08 会被解压器当作“复制 n 字节”，>=0x80 的字节会被当作
// 空格/长度对指令，因此这些字面量都要用 0x02 打包成对输出。
func palmDOCCompress(b []byte) []byte {
	var out []byte
	i := 0
	for i < len(b) {
		c := b[i]
		if c >= 0x80 || c <= 8 {
			next := byte(0)
			if i+1 < len(b) {
				next = b[i+1]
			}
			out = append(out, 0x02, c, next)
			i += 2
		} else {
			out = append(out, c)
			i++
		}
	}
	return out
}

func TestOpenAZW3(t *testing.T) {
	path := buildTestAZW3(t)
	r, err := OpenAZW3(path)
	if err != nil {
		t.Fatalf("OpenAZW3 failed: %v", err)
	}
	defer r.Close()

	if r.Title != "Test Book" {
		t.Errorf("title = %q, want %q", r.Title, "Test Book")
	}
	if len(r.Authors) != 1 || r.Authors[0] != "Test Author" {
		t.Errorf("authors = %v, want [Test Author]", r.Authors)
	}
	if r.Language != "zh" {
		t.Errorf("language = %q, want zh", r.Language)
	}
	if len(r.chapters) != 3 {
		t.Fatalf("chapters = %d, want 3", len(r.chapters))
	}

	// 章节 HTML：图片引用已替换、编码正确
	c0, err := r.GetContent(0)
	if err != nil {
		t.Fatalf("GetContent(0): %v", err)
	}
	s0 := string(c0)
	if !strings.Contains(s0, "<p>Hello <b>AZW3</b></p>") {
		t.Errorf("chapter 0 content missing text: %s", s0)
	}
	if strings.Contains(s0, "kindle:embed:0001") {
		t.Errorf("chapter 0 still contains kindle:embed: %s", s0)
	}
	if !strings.Contains(s0, `src="images/1"`) {
		t.Errorf("chapter 0 image ref not rewritten: %s", s0)
	}

	// 图片资源
	img, err := r.GetResource("images/1")
	if err != nil || len(img) < 8 || !looksLikeImage(img) {
		t.Errorf("GetResource(images/1) failed: err=%v len=%d", err, len(img))
	}

	// 封面
	cover, err := r.GetCover()
	if err != nil {
		t.Errorf("GetCover: %v", err)
	}
	if r.CoverPath != "images/1" {
		t.Errorf("CoverPath = %q, want images/1", r.CoverPath)
	}
	if len(cover) == 0 {
		t.Error("cover data empty")
	}

	// 章节列表
	cl := r.GetChapterList()
	if len(cl) != 3 {
		t.Errorf("GetChapterList len = %d, want 3", len(cl))
	}
	if cl[0]["title"] != "第一章 起点" {
		t.Errorf("chapter 0 title = %v, want 第一章 起点", cl[0]["title"])
	}

	// 目录（无 NCX，回退为按章节）
	items := r.TOCItems()
	if len(items) != 3 {
		t.Fatalf("TOC len = %d, want 3", len(items))
	}
	if items[0].Index != 0 || items[0].Title != "第一章 起点" {
		t.Errorf("TOC[0] = %+v", items[0])
	}
	if items[2].Title != "第三章 结局" {
		t.Errorf("TOC[2] = %+v", items[2])
	}

	// 越界保护
	if _, err := r.GetContent(99); err == nil {
		t.Error("GetContent(99) should fail")
	}
	if _, err := r.GetResource("../etc/passwd"); err == nil {
		t.Error("GetResource with traversal should fail")
	}
}

func TestPalmDOCDecompressRoundtrip(t *testing.T) {
	// 包含特殊字节 0x00-0x04 的往返测试
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 'a', 'b', 'c', 0x00, 0xFF, 0x0D, 0x0A}
	comp := palmDOCCompress(data)
	dec := decompressPalmDOC(comp)
	if string(dec) != string(data) {
		t.Errorf("roundtrip mismatch:\n got %v\nwant %v", dec, data)
	}
}

func TestExtractAZW3Metadata(t *testing.T) {
	path := buildTestAZW3(t)
	meta, err := extractAZW3Metadata(path)
	if err != nil {
		t.Fatalf("extractAZW3Metadata: %v", err)
	}
	if meta.Title != "Test Book" {
		t.Errorf("title = %q", meta.Title)
	}
	if len(meta.CoverData) == 0 {
		t.Error("cover data missing")
	}
}
