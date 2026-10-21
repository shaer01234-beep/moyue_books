package services

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestConvertGBKText(t *testing.T) {
	dir := t.TempDir()
	// 原始中文（UTF-8）编码为 GBK 写入 .txt，模拟乱码来源
	utf8Text := "第一回 风雪山神庙\n林冲雪夜上梁山，替天行道。\n第二回 汴京城内事"
	gbkBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(utf8Text))
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(src, gbkBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out")
	epub, err := ConvertTextToEPUB(src, "测试书", "txt", out)
	if err != nil {
		t.Fatal(err)
	}

	// 解压 EPUB，读取实际章节文件内容（章节在 zip 内是压缩的，不能直接字符串搜索整包）
	zr, err := zip.OpenReader(epub)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var chapter string
	for _, f := range zr.File {
		if f.Name == "OEBPS/chapter-1.xhtml" {
			rc, _ := f.Open()
			var buf bytes.Buffer
			buf.ReadFrom(rc)
			rc.Close()
			chapter = buf.String()
		}
	}
	if !strings.Contains(chapter, "风雪山神庙") || !strings.Contains(chapter, "林冲雪夜上梁山") {
		t.Fatalf("GBK 内容未正确转换为 UTF-8，产物章节中未找到期望中文。章节片段: %q", chapter[:min(len(chapter), 400)])
	}
	t.Logf("OK: GBK -> UTF-8 转换成功")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
