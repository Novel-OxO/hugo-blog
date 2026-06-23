package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// renderPage는 한 Notion 페이지를 Hugo leaf bundle(index.md + 이미지)로 쓴다.
// 반환값은 생성된 슬러그다.
func renderPage(c *Client, p Page, contentDir string) (string, error) {
	title := plainText(p.prop("Title").Title)
	if title == "" {
		title = "untitled"
	}
	slug := plainText(p.prop("Slug").RichText)
	if slug == "" {
		slug = slugify(title)
	}

	bundleDir := filepath.Join(contentDir, slug)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return slug, err
	}

	// 이미지 다운로더: page bundle 안에 img-N.<ext>로 저장하고 파일명을 돌려준다.
	imgN := 0
	dl := func(url string) (string, error) {
		imgN++
		return downloadAsset(url, bundleDir, fmt.Sprintf("img-%d", imgN))
	}

	blocks, err := c.GetBlocks(p.ID)
	if err != nil {
		return slug, err
	}
	body, err := renderBlocks(c, blocks, dl, 0)
	if err != nil {
		return slug, err
	}

	fm := buildFrontMatter(p, title, slug, bundleDir)
	out := fm + "\n" + strings.TrimRight(body, "\n") + "\n"
	return slug, os.WriteFile(filepath.Join(bundleDir, "index.md"), []byte(out), 0o644)
}

func buildFrontMatter(p Page, title, slug, bundleDir string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + yamlString(title) + "\n")

	if d := p.prop("Date").Date; d != nil && d.Start != "" {
		b.WriteString("date: " + d.Start + "\n")
	}
	b.WriteString("slug: " + yamlString(slug) + "\n")

	if desc := plainText(p.prop("Summary").RichText); desc != "" {
		b.WriteString("description: " + yamlString(desc) + "\n")
	}

	if tags := p.prop("Tags").MultiSelect; len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, t := range tags {
			names = append(names, yamlString(t.Name))
		}
		b.WriteString("tags: [" + strings.Join(names, ", ") + "]\n")
	}

	// 커버 이미지(1시간 만료 대상): DB "Cover" 파일 속성 우선, 없으면 페이지 배너.
	coverURL := ""
	if files := p.prop("Cover").Files; len(files) > 0 {
		coverURL = files[0].URL()
	}
	if coverURL == "" {
		coverURL = p.Cover.URL()
	}
	if coverURL != "" {
		if name, err := downloadAsset(coverURL, bundleDir, "cover"); err == nil {
			b.WriteString("cover: " + yamlString(name) + "\n")
		} else {
			fmt.Fprintf(os.Stderr, "  ⚠ 커버 다운로드 실패: %v\n", err)
		}
	}

	b.WriteString("draft: false\n")
	b.WriteString("---\n")
	return b.String()
}

// renderBlocks는 블록 슬라이스를 마크다운 문자열로 변환한다. depth는 리스트 중첩 들여쓰기용.
func renderBlocks(c *Client, blocks []Block, dl func(string) (string, error), depth int) (string, error) {
	var b strings.Builder
	indent := strings.Repeat("  ", depth)
	num := 0
	prevList := false

	for _, blk := range blocks {
		if blk.Type != "numbered_list_item" {
			num = 0
		}
		// 리스트 그룹이 끝나고 비(非)리스트 블록이 오면 빈 줄을 넣는다.
		// (빈 줄이 없으면 뒤따르는 문단이 마지막 리스트 항목에 흡수됨)
		if prevList && !isListItem(blk.Type) {
			b.WriteString("\n")
		}
		prevList = isListItem(blk.Type)

		switch blk.Type {
		case "paragraph":
			if txt := renderRich(blk.Paragraph.RichText); strings.TrimSpace(txt) != "" {
				b.WriteString(indent + txt + "\n\n")
			}
		case "heading_1":
			b.WriteString("# " + renderRich(blk.Heading1.RichText) + "\n\n")
		case "heading_2":
			b.WriteString("## " + renderRich(blk.Heading2.RichText) + "\n\n")
		case "heading_3":
			b.WriteString("### " + renderRich(blk.Heading3.RichText) + "\n\n")
		case "bulleted_list_item":
			b.WriteString(indent + "- " + renderRich(blk.BulletedListItem.RichText) + "\n")
			if err := appendChildren(c, &b, blk, dl, depth+1); err != nil {
				return "", err
			}
		case "numbered_list_item":
			num++
			b.WriteString(fmt.Sprintf("%s%d. %s\n", indent, num, renderRich(blk.NumberedListItem.RichText)))
			if err := appendChildren(c, &b, blk, dl, depth+1); err != nil {
				return "", err
			}
		case "to_do":
			mark := " "
			if blk.ToDo.Checked {
				mark = "x"
			}
			b.WriteString(fmt.Sprintf("%s- [%s] %s\n", indent, mark, renderRich(blk.ToDo.RichText)))
		case "quote":
			b.WriteString("> " + renderRich(blk.Quote.RichText) + "\n\n")
		case "code":
			b.WriteString("```" + mapLang(blk.Code.Language) + "\n")
			b.WriteString(plainText(blk.Code.RichText) + "\n")
			b.WriteString("```\n\n")
		case "image":
			if url := blk.Image.URL(); url != "" {
				name, err := dl(url)
				if err != nil {
					return "", err
				}
				b.WriteString(fmt.Sprintf("![%s](%s)\n\n", renderRich(blk.Image.Caption), name))
			}
		case "divider":
			b.WriteString("---\n\n")
		default:
			// 미지원 블록이라도 자식이 있으면 이어서 렌더한다.
			if blk.HasChildren {
				if err := appendChildren(c, &b, blk, dl, depth); err != nil {
					return "", err
				}
			}
		}
	}
	return b.String(), nil
}

func appendChildren(c *Client, b *strings.Builder, blk Block, dl func(string) (string, error), depth int) error {
	if !blk.HasChildren {
		return nil
	}
	children, err := c.GetBlocks(blk.ID)
	if err != nil {
		return err
	}
	s, err := renderBlocks(c, children, dl, depth)
	if err != nil {
		return err
	}
	b.WriteString(s)
	return nil
}

// renderRich는 rich text 배열을 인라인 마크다운(굵게/기울임/코드/링크)으로 변환한다.
func renderRich(rts []RichText) string {
	var b strings.Builder
	for _, rt := range rts {
		t := rt.PlainText
		if t == "" {
			continue
		}
		a := rt.Annotations
		if a.Code {
			t = "`" + t + "`"
		}
		if a.Bold {
			t = "**" + t + "**"
		}
		if a.Italic {
			t = "_" + t + "_"
		}
		if a.Strikethrough {
			t = "~~" + t + "~~"
		}
		if rt.Href != "" && !a.Code {
			t = fmt.Sprintf("[%s](%s)", t, rt.Href)
		}
		b.WriteString(t)
	}
	return b.String()
}

func isListItem(t string) bool {
	switch t {
	case "bulleted_list_item", "numbered_list_item", "to_do":
		return true
	default:
		return false
	}
}

func plainText(rts []RichText) string {
	var b strings.Builder
	for _, rt := range rts {
		b.WriteString(rt.PlainText)
	}
	return b.String()
}

// ---- 유틸 ----

var slugStrip = regexp.MustCompile(`[^a-z0-9가-힣]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugStrip.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "post"
	}
	return s
}

func yamlString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func mapLang(l string) string {
	switch l {
	case "plain text", "plaintext", "":
		return ""
	default:
		return l
	}
}

// downloadAsset은 url의 콘텐츠를 dir/base.<ext>로 저장하고 파일명을 반환한다.
// Notion 이미지 URL은 1시간 후 만료되므로 동기화 실행 중에 즉시 받아야 한다.
func downloadAsset(url, dir, base string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("자산 다운로드 실패 %s: %s", url, resp.Status)
	}
	name := base + assetExt(url, resp.Header.Get("Content-Type"))
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return name, nil
}

func assetExt(url, contentType string) string {
	if i := strings.IndexByte(url, '?'); i >= 0 {
		url = url[:i]
	}
	if ext := strings.ToLower(filepath.Ext(url)); ext != "" && len(ext) <= 5 {
		return ext
	}
	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "jpeg"):
		return ".jpg"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "svg"):
		return ".svg"
	}
	return ".img"
}
