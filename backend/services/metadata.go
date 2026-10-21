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

// BookMetadata 提取的图书元数据
type BookMetadata struct {
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Publisher   string   `json:"publisher"`
	PublishDate string   `json:"publish_date"`
	Language    string   `json:"language"`
	ISBN        string   `json:"isbn"`
	Tags        []string `json:"tags"`
	Series      string   `json:"series"`
	SeriesIndex float64  `json:"series_index"`
	Pages       int      `json:"pages"`
	CoverData   []byte   `json:"-"` // 封面二进制数据
}

// ExtractMetadata 从文件路径提取元数据
func ExtractMetadata(filePath string) (*BookMetadata, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".epub", ".kepub":
		// KEPUB 为 EPUB 变种（同为 ZIP 容器 + container.xml + OPF），解析方式完全一致
		return extractEPUBMetadata(filePath)
	case ".azw3", ".mobi":
		meta, err := extractAZW3Metadata(filePath)
		if err != nil {
			return nil, err
		}
		return &meta, nil
	case ".pdf":
		return extractPDFMetadata(filePath)
	default:
		return extractFromFilename(filePath), nil
	}
}

// extractEPUBMetadata 从 EPUB 文件提取元数据
func extractEPUBMetadata(filePath string) (*BookMetadata, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return extractFromFilename(filePath), nil
	}
	defer zr.Close()

	meta := &BookMetadata{}

	// 1. 找到 container.xml 中 OPF 文件路径
	var opfPath string
	for _, f := range zr.File {
		if f.Name == "META-INF/container.xml" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			// 简单解析 full-path
			re := regexp.MustCompile(`full-path="([^"]+)"`)
			m := re.FindSubmatch(data)
			if len(m) > 1 {
				opfPath = string(m[1])
			}
			break
		}
	}

	// 2. 解析 OPF 文件
	if opfPath != "" {
		for _, f := range zr.File {
			if f.Name == opfPath {
				rc, err := f.Open()
				if err != nil {
					break
				}
				data, _ := io.ReadAll(rc)
				rc.Close()

				var opf OPFPackage
				if err := xml.Unmarshal(data, &opf); err == nil {
					if len(opf.Metadata.Title) > 0 {
						meta.Title = opf.Metadata.Title[0].Text
					}
					meta.Description = opf.Metadata.Description.Text
					meta.Publisher = opf.Metadata.Publisher.Text
					meta.Language = opf.Metadata.Language.Text
					for _, c := range opf.Metadata.Creator {
						meta.Authors = append(meta.Authors, c.Text)
					}
					for _, d := range opf.Metadata.Date {
						meta.PublishDate = d.Text
						break
					}
					for _, id := range opf.Metadata.Identifier {
						if strings.Contains(strings.ToLower(id.Scheme), "isbn") ||
							strings.Contains(id.Text, "978") ||
							strings.Contains(id.Text, "ISBN") {
							meta.ISBN = id.Text
							break
						}
					}
					for _, s := range opf.Metadata.Subject {
						meta.Tags = append(meta.Tags, s.Text)
					}
					// 丛书信息
					for _, mmeta := range opf.Metadata.Meta {
						switch mmeta.Name {
						case "calibre:series":
							meta.Series = mmeta.Content
						case "calibre:series_index":
							fmt.Sscanf(mmeta.Content, "%f", &meta.SeriesIndex)
						}
					}
				}
				break
			}
		}
	}

	// 3. 提取封面（若 OPF 未指定则尝试自动查找）
	if meta.CoverData == nil {
		for _, f := range zr.File {
			lowerName := strings.ToLower(f.Name)
			if strings.Contains(lowerName, "cover") &&
				(strings.HasSuffix(lowerName, ".jpg") || strings.HasSuffix(lowerName, ".jpeg") ||
					strings.HasSuffix(lowerName, ".png")) {
				rc, err := f.Open()
				if err == nil {
					meta.CoverData, _ = io.ReadAll(rc)
					rc.Close()
					break
				}
			}
		}
	}

	// 如果元数据为空，回退到文件名解析
	if meta.Title == "" {
		fallback := extractFromFilename(filePath)
		if meta.Title == "" {
			meta.Title = fallback.Title
		}
		if len(meta.Authors) == 0 {
			meta.Authors = fallback.Authors
		}
		meta.Publisher = coalesce(meta.Publisher, fallback.Publisher)
	}

	return meta, nil
}

// OPF XML 结构
type OPFPackage struct {
	XMLName  xml.Name   `xml:"package"`
	Metadata OPFMeta    `xml:"metadata"`
	Manifest []OPFItem  `xml:"manifest>item"`
	Spine    OPFSpine   `xml:"spine"`
}

type OPFMeta struct {
	Title       []OPFText   `xml:"title"`
	Creator     []OPFText   `xml:"creator"`
	Description OPFText     `xml:"description"`
	Publisher   OPFText     `xml:"publisher"`
	Date        []OPFText   `xml:"date"`
	Language    OPFText     `xml:"language"`
	Identifier  []OPFIdent  `xml:"identifier"`
	Subject     []OPFText   `xml:"subject"`
	Meta        []OPFMeta2  `xml:"meta"`
}
type OPFText struct {
	Text string `xml:",chardata"`
}
type OPFIdent struct {
	Text   string `xml:",chardata"`
	Scheme string `xml:"scheme,attr"`
	ID     string `xml:"id,attr"`
}
type OPFMeta2 struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}
type OPFItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}
type OPFSpine struct {
	ItemRef []struct {
		IDRef string `xml:"idref,attr"`
	} `xml:"itemref"`
}

// extractPDFMetadata 从 PDF 文件提取元数据（简化版）
func extractPDFMetadata(filePath string) (*BookMetadata, error) {
	meta := extractFromFilename(filePath)
	return meta, nil
}

// extractFromFilename 从文件名智能解析元数据
func extractFromFilename(filePath string) *BookMetadata {
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	meta := &BookMetadata{}

	// 常见命名格式: "书名 - 作者" 或 "作者 - 书名" 或 "[丛书] 书名 (作者)"
	// 先移除常见标签如 [EPUB], (2023), 等
	base = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(base, "")
	base = regexp.MustCompile(`\([^)]*\d{4}[^)]*\)`).ReplaceAllString(base, "")
	base = strings.TrimSpace(base)

	// 尝试按 " - " 分割
	if parts := strings.Split(base, " - "); len(parts) == 2 {
		left, right := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		// 如果左边看起来像作者（短且无书名号），则使用 作者-书名 格式
		if len(left) < 30 && !strings.Contains(left, "《") {
			meta.Authors = []string{left}
			meta.Title = right
		} else {
			meta.Title = left
			meta.Authors = []string{right}
		}
	} else {
		meta.Title = base
	}

	// 清理书名中的多余空格
	meta.Title = strings.Join(strings.Fields(meta.Title), " ")
	return meta
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
