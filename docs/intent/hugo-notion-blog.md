# Hugo + Notion 블로그 — 확정 의도

> `interview-me` 세션(2026-06-23)으로 확정한 프로젝트 의도 문서.
> 다음 세션/핸드오프 시 이 문서를 먼저 읽으면 맥락이 복원됩니다.

## 한 줄 요약

Notion에서 글을 쓰면 → Go 동기화 CLI가 Hugo 콘텐츠(.md + 이미지)로 변환·커밋 →
Tailwind v4로 직접 만든(테마 없는) 사이트로 빌드 → GitHub Pages에 배포.
전 과정이 GitHub Actions `workflow_dispatch` 한 번으로 수동 실행된다.

## 확정 스펙

| 항목 | 결정 |
|---|---|
| Outcome | Notion Published 글 → Hugo 변환·커밋 → Tailwind4 빌드 → GitHub Pages 배포 |
| User | 1인 운영. 독자는 동료 개발자 + 브랜딩 대상 |
| Why now | 개인 기술 블로그를 브랜딩 목적으로 새로 시작. 글쓰기는 Notion에서 편하게 |
| Success | Notion에 Published 글을 두고 워크플로 실행 → 이미지 깨짐 없이 사이트 반영·배포 |
| 디자인 | v1은 최소 베이스라인. 정교한 비주얼은 추후 본인이 직접 제공 → **교체가 쉬운 구조**가 핵심 |

## 제약 (binding constraints)

1. **Notion 이미지 URL 1시간 만료** — Notion API가 주는 파일/이미지 URL은 pre-signed
   S3 URL로 1시간 후 만료된다. 따라서 동기화 CLI가 **같은 실행 안에서** 이미지를 내려받아
   page bundle에 저장하고, 마크다운 경로를 로컬 상대경로로 치환·커밋해야 한다.
2. **테마 없이 Tailwind v4 직접** — Hugo 네이티브 `css.TailwindCSS` 파이프라인 사용.
   빌드에 Node(`@tailwindcss/cli`)와 Hugo Extended가 필요.

## 기능 범위

- **v1 포함**: 코드 하이라이트 / 태그·아카이브 / RSS / OG 메타
- **제외(나중에)**: 다크모드, 검색, 댓글, 방문 분석
- **나중에 본인 제공**: 정교한 비주얼/모션 디자인

## 아키텍처

```
Notion DB (Blog Posts)
   │  Status = Published 인 행만
   ▼
Go 동기화 CLI (tools/notion-sync, jomei/notionapi)
   │  - 속성 → front matter 매핑
   │  - 블록 → 마크다운 변환
   │  - 1시간 만료 전에 이미지 다운로드 → page bundle에 저장 + 경로 치환
   ▼
content/posts/<slug>/index.md (+ 이미지)   ← repo에 커밋
   ▼
Hugo (테마 없음) + Tailwind v4 (css.TailwindCSS)
   ▼
public/  →  GitHub Pages 배포
```

전체 흐름은 GitHub Actions `workflow_dispatch` 한 번으로:
**CLI 실행 → 변경분 커밋 → Hugo 빌드 → Pages 배포**

## Notion DB 스키마 (Blog Posts)

- Database ID: `958404d6a943456fbe496aee58b4cad0`
- Data source ID: `5009f990-00fb-4305-8d2f-2b1556e1afab`

| 속성 | 타입 | front matter | 용도 |
|---|---|---|---|
| Title | Title | `title` | 글 제목 |
| Slug | Text | URL 슬러그 | 비우면 Title에서 자동 생성 |
| Status | Select(Draft/Published) | — | 발행 필터(Published만 빌드) |
| Date | Date | `date` | 발행일 |
| Tags | Multi-select | `tags` | Go / Hugo / Frontend / Backend / DevOps |
| Summary | Text | `description` | OG description / 목록 요약 |
| Cover | Files | (썸네일) | OG 썸네일 |

## 필요한 사전 준비 (사용자 직접)

1. Notion **Integration(Internal)** 생성 → API 토큰 발급
2. 위 **Blog Posts DB를 그 integration에 Connect** (공유)
3. 토큰을 GitHub Actions Secret `NOTION_TOKEN` 으로 등록
4. Repo Settings → Pages → Source를 **GitHub Actions** 로 설정

## 진행 상태

- [x] Notion DB 설계·생성 + 예시 글 1개
- [x] 의도 문서화 / private 레포 생성
- [x] Hugo 스캐폴드 (테마 없음) — 로컬 빌드 검증 완료(Hugo v0.163.3)
- [x] Tailwind v4 연결 (css.TailwindCSS) — `@source hugo_stats.json` 클래스 스캔 확인
- [x] Go 동기화 CLI (notion-sync) — stdlib만 사용, `go vet`/`go build` 통과
- [x] GitHub Actions 워크플로 (workflow_dispatch → sync → commit → build → deploy)

### 남은 작업 (사용자 액션 후 첫 배포)

1. Notion Integration 토큰 발급 + Blog Posts DB Connect
2. GitHub Secret `NOTION_TOKEN` 등록
3. Repo Settings → Pages → Source = GitHub Actions
4. Actions → Deploy → Run workflow 로 첫 배포
