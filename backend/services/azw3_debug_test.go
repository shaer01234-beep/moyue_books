package services

import (
	"fmt"
	"testing"
)

// TestDebugRealAZW3 用真实 AZW3 文件验证解析：章节、目录、图片、内容
func TestDebugRealAZW3(t *testing.T) {
	path := "/Users/apple/Documents/my/books/backend/data/uploads/books/cf5dabf6-ac83-4160-b622-3eb271350e77.azw3"
	a, err := OpenAZW3(path)
	if err != nil {
		t.Fatalf("OpenAZW3 error: %v", err)
	}
	defer a.Close()
	fmt.Printf("Title=%q Authors=%v Language=%q\n", a.Title, a.Authors, a.Language)
	fmt.Printf("chapters=%d images=%d\n", len(a.chapters), len(a.images))
	items := a.TOCItems()
	fmt.Printf("TOCItems=%d\n", len(items))
	for i, it := range items {
		if i >= 12 {
			break
		}
		fmt.Printf("  [%d] idx=%d title=%q path=%q\n", i, it.Index, it.Title, it.Path)
	}
	if len(a.chapters) == 0 {
		t.Fatalf("no chapters parsed")
	}
	if len(items) == 0 {
		t.Fatalf("no TOC parsed")
	}
	html, err := a.GetContent(0)
	if err != nil || len(html) == 0 {
		t.Fatalf("GetContent(0): err=%v len=%d", err, len(html))
	}
	fmt.Printf("content0 len=%d head=%q\n", len(html), string(html)[:min(120, len(html))])
}
