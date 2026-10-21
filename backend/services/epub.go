package services

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// EPUBReader EPUB 文件阅读辅助
type EPUBReader struct {
	Path       string
	OPFPath    string
	Contents   []string // HTML/XHTML 文件列表
	Title      string
	Authors    []string
	CoverPath  string
	isOpen     bool
	reader     *zip.ReadCloser
}

// OpenEPUB 打开 EPUB 文件
func OpenEPUB(path string) (*EPUBReader, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	epub := &EPUBReader{
		Path:   path,
		reader: zr,
		isOpen: true,
	}

	// 解析 container.xml 找到 OPF
	for _, f := range zr.File {
		if f.Name == "META-INF/container.xml" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			re := regexp.MustCompile(`full-path="([^"]+)"`)
			m := re.FindSubmatch(data)
			if len(m) > 1 {
				epub.OPFPath = string(m[1])
			}
			break
		}
	}

	// 解析 OPF
	if epub.OPFPath != "" {
		for _, f := range zr.File {
			if f.Name == epub.OPFPath {
				rc, _ := f.Open()
				data, _ := io.ReadAll(rc)
				rc.Close()

				var opf OPFPackage
				if err := xml.Unmarshal(data, &opf); err == nil {
					if len(opf.Metadata.Title) > 0 {
						epub.Title = opf.Metadata.Title[0].Text
					}
					for _, c := range opf.Metadata.Creator {
						epub.Authors = append(epub.Authors, c.Text)
					}
					// 找封面
					for _, item := range opf.Manifest {
						lowerID := strings.ToLower(item.ID)
						if strings.Contains(lowerID, "cover") &&
							(strings.HasSuffix(strings.ToLower(item.Href), ".jpg") ||
								strings.HasSuffix(strings.ToLower(item.Href), ".jpeg") ||
								strings.HasSuffix(strings.ToLower(item.Href), ".png")) {
							opfDir := filepath.Dir(epub.OPFPath)
							epub.CoverPath = filepath.Join(opfDir, item.Href)
							break
						}
					}
					// 内容文件列表
					for _, item := range opf.Spine.ItemRef {
						for _, manifest := range opf.Manifest {
							if manifest.ID == item.IDRef {
								opfDir := filepath.Dir(epub.OPFPath)
								epub.Contents = append(epub.Contents, filepath.Join(opfDir, manifest.Href))
								break
							}
						}
					}
				}
				break
			}
		}
	}

	return epub, nil
}

// GetCover 获取封面数据
func (e *EPUBReader) GetCover() ([]byte, error) {
	if e.CoverPath == "" {
		return nil, fmt.Errorf("no cover found")
	}
	for _, f := range e.reader.File {
		if f.Name == e.CoverPath || filepath.ToSlash(f.Name) == filepath.ToSlash(e.CoverPath) {
			rc, _ := f.Open()
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("cover not found in archive")
}

// GetContent 获取指定索引的章节内容
func (e *EPUBReader) GetContent(index int) ([]byte, error) {
	if index < 0 || index >= len(e.Contents) {
		return nil, fmt.Errorf("content index out of range")
	}
	path := e.Contents[index]
	for _, f := range e.reader.File {
		if f.Name == path || filepath.ToSlash(f.Name) == filepath.ToSlash(path) {
			rc, _ := f.Open()
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("content not found: %s", path)
}

// GetResource 获取 EPUB 内指定路径的资源文件（图片、CSS 等），供 <img> 等直接加载
func (e *EPUBReader) GetResource(path string) ([]byte, error) {
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.Contains(clean, "..") {
		return nil, fmt.Errorf("invalid resource path: %s", path)
	}
	for _, f := range e.reader.File {
		if f.Name == clean || filepath.ToSlash(f.Name) == clean {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("resource not found: %s", path)
}

// GetChapterList 获取章节列表
func (e *EPUBReader) GetChapterList() []map[string]interface{} {
	var chapters []map[string]interface{}
	for i, path := range e.Contents {
		chapters = append(chapters, map[string]interface{}{
			"index": i,
			"title": fmt.Sprintf("章节 %d", i+1),
			"path":  path,
		})
	}
	return chapters
}

// Close 关闭 EPUB
func (e *EPUBReader) Close() {
	if e.isOpen && e.reader != nil {
		e.reader.Close()
		e.isOpen = false
	}
}
