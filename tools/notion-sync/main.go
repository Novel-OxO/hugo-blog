// Command notion-sync는 Notion DB의 Published 글을 Hugo 콘텐츠로 변환한다.
//
// Notion API가 주는 이미지 URL은 1시간 후 만료되는 pre-signed URL이므로,
// 이 도구는 같은 실행 안에서 이미지를 page bundle에 내려받고 마크다운 경로를
// 로컬 상대경로로 치환한다. 결과물(content/posts/<slug>/index.md + 이미지)은
// repo에 커밋되어 Notion 원본이 사라져도 살아남는다.
package main

import (
	"log"
	"os"
	"path/filepath"
)

func main() {
	token := os.Getenv("NOTION_TOKEN")
	if token == "" {
		log.Fatal("환경변수 NOTION_TOKEN 이 필요합니다")
	}
	dbID := envOr("NOTION_DATABASE_ID", "958404d6a943456fbe496aee58b4cad0")
	contentDir := envOr("CONTENT_DIR", filepath.Join("content", "posts"))
	version := envOr("NOTION_VERSION", "2022-06-28")

	client := NewClient(token, version)

	pages, err := client.QueryPublished(dbID)
	if err != nil {
		log.Fatalf("Notion DB 쿼리 실패: %v", err)
	}
	log.Printf("Published 글 %d개 발견", len(pages))

	// Notion이 단일 소스이므로 기존 생성물을 지우고 다시 만든다
	// (Notion에서 unpublish/삭제한 글이 사이트에서도 사라지도록).
	if err := os.RemoveAll(contentDir); err != nil {
		log.Fatalf("기존 콘텐츠 삭제 실패: %v", err)
	}
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		log.Fatalf("콘텐츠 디렉토리 생성 실패: %v", err)
	}

	for _, p := range pages {
		slug, err := renderPage(client, p, contentDir)
		if err != nil {
			log.Fatalf("페이지 변환 실패(%s): %v", p.ID, err)
		}
		log.Printf("  ✓ %s", slug)
	}
	log.Printf("완료: %s 에 %d개 글 생성", contentDir, len(pages))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
