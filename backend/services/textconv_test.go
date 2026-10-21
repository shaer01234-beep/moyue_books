package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertTextToEPUB_MD(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "demo.md")
	md := "# 标题\n\n这是**粗体**和*斜体*，还有`代码`。\n\n- 列表项1\n- 列表项2\n\n```go\nfmt.Println(\"hi\")\n```\n\n> 引用一句\n"
	if err := os.WriteFile(src, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := ConvertTextToEPUB(src, "我的书", "md", dir)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	r, err := OpenEPUB(out)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	if len(r.Contents) != 1 {
		t.Fatalf("want 1 content, got %d", len(r.Contents))
	}
	if r.Title != "我的书" {
		t.Fatalf("title got %q", r.Title)
	}
	c, err := r.GetContent(0)
	if err != nil || len(c) == 0 {
		t.Fatalf("get content: %v len=%d", err, len(c))
	}
}

func TestConvertTextToEPUB_TXT(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "demo.txt")
	if err := os.WriteFile(src, []byte("第一段。\n\n第二段，有换行。\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := ConvertTextToEPUB(src, "纯文本书", "txt", dir)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	r, err := OpenEPUB(out)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	if r.Title != "纯文本书" {
		t.Fatalf("title got %q", r.Title)
	}
}

func TestConvertTextToEPUB_MD_MultiChapter(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "book.md")
	md := "# 第一章\n\n内容一。\n\n# 第二章\n\n内容二。\n\n## 小节\n\n子内容。\n\n# 第三章\n\n内容三。\n"
	if err := os.WriteFile(src, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := ConvertTextToEPUB(src, "", "md", dir)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	r, err := OpenEPUB(out)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	// 三个一级标题 -> 三章
	if len(r.Contents) != 3 {
		t.Fatalf("want 3 chapters, got %d", len(r.Contents))
	}
	// 未显式指定书名时应取首个一级标题
	if r.Title != "第一章" {
		t.Fatalf("title got %q", r.Title)
	}
	// 验证第一章：正文应含章节内容，且原文一级标题已被抽离为章节标题（正文内仅保留一个 <h1> 即章节标题，不重复）
	c0, _ := r.GetContent(0)
	if !strings.Contains(string(c0), "内容一") {
		t.Fatalf("chapter 0 missing its body content: %s", c0)
	}
	if strings.Count(string(c0), "<h1") != 1 {
		t.Fatalf("chapter body should have exactly one <h1> (chapter title), got: %s", c0)
	}
	c2, _ := r.GetContent(2)
	if !strings.Contains(string(c2), "内容三") {
		t.Fatalf("chapter 2 missing its body content: %s", c2)
	}
}
