package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const notionAPI = "https://api.notion.com/v1"

// Client는 Notion REST API에 대한 최소 클라이언트다(외부 의존성 없음).
type Client struct {
	token   string
	version string
	http    *http.Client
}

func NewClient(token, version string) *Client {
	return &Client{
		token:   token,
		version: version,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) do(method, url string, body any) ([]byte, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", c.version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Notion API %s: %s", resp.Status, string(data))
	}
	return data, nil
}

// ---- 데이터 타입 ----

type Page struct {
	ID         string              `json:"id"`
	Properties map[string]Property `json:"properties"`
	Cover      *FileObject         `json:"cover"`
}

func (p Page) prop(name string) Property { return p.Properties[name] }

type Property struct {
	Type        string       `json:"type"`
	Title       []RichText   `json:"title"`
	RichText    []RichText   `json:"rich_text"`
	Select      *SelectOpt   `json:"select"`
	MultiSelect []SelectOpt  `json:"multi_select"`
	Date        *DateValue   `json:"date"`
	Files       []FileObject `json:"files"`
}

type SelectOpt struct {
	Name string `json:"name"`
}

type DateValue struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type FileObject struct {
	Type     string     `json:"type"`
	File     *URLHolder `json:"file"`
	External *URLHolder `json:"external"`
}

func (f *FileObject) URL() string {
	switch {
	case f == nil:
		return ""
	case f.File != nil:
		return f.File.URL
	case f.External != nil:
		return f.External.URL
	}
	return ""
}

type URLHolder struct {
	URL string `json:"url"`
}

type RichText struct {
	PlainText   string      `json:"plain_text"`
	Href        string      `json:"href"`
	Annotations Annotations `json:"annotations"`
}

type Annotations struct {
	Bold          bool `json:"bold"`
	Italic        bool `json:"italic"`
	Strikethrough bool `json:"strikethrough"`
	Code          bool `json:"code"`
}

// ---- 쿼리 ----

type listResponse[T any] struct {
	Results    []T    `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

// QueryPublished는 Status=Published 인 모든 페이지를 Date 내림차순으로 반환한다.
func (c *Client) QueryPublished(dbID string) ([]Page, error) {
	var pages []Page
	cursor := ""
	for {
		body := map[string]any{
			"filter": map[string]any{
				"property": "Status",
				"select":   map[string]any{"equals": "Published"},
			},
			"sorts": []map[string]any{
				{"property": "Date", "direction": "descending"},
			},
			"page_size": 100,
		}
		if cursor != "" {
			body["start_cursor"] = cursor
		}
		data, err := c.do("POST", notionAPI+"/databases/"+dbID+"/query", body)
		if err != nil {
			return nil, err
		}
		var res listResponse[Page]
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, err
		}
		pages = append(pages, res.Results...)
		if !res.HasMore {
			break
		}
		cursor = res.NextCursor
	}
	return pages, nil
}

// ---- 블록 ----

type Block struct {
	ID               string         `json:"id"`
	Type             string         `json:"type"`
	HasChildren      bool           `json:"has_children"`
	Paragraph        *RichHolder    `json:"paragraph"`
	Heading1         *RichHolder    `json:"heading_1"`
	Heading2         *RichHolder    `json:"heading_2"`
	Heading3         *RichHolder    `json:"heading_3"`
	BulletedListItem *RichHolder    `json:"bulleted_list_item"`
	NumberedListItem *RichHolder    `json:"numbered_list_item"`
	Quote            *RichHolder    `json:"quote"`
	Code             *CodeBlock     `json:"code"`
	Image            *FileObjectCap `json:"image"`
	ToDo             *ToDoBlock     `json:"to_do"`
}

type RichHolder struct {
	RichText []RichText `json:"rich_text"`
}

type CodeBlock struct {
	RichText []RichText `json:"rich_text"`
	Language string     `json:"language"`
}

type FileObjectCap struct {
	Type     string     `json:"type"`
	File     *URLHolder `json:"file"`
	External *URLHolder `json:"external"`
	Caption  []RichText `json:"caption"`
}

func (f *FileObjectCap) URL() string {
	switch {
	case f == nil:
		return ""
	case f.File != nil:
		return f.File.URL
	case f.External != nil:
		return f.External.URL
	}
	return ""
}

type ToDoBlock struct {
	RichText []RichText `json:"rich_text"`
	Checked  bool       `json:"checked"`
}

// GetBlocks는 한 블록(또는 페이지)의 모든 자식 블록을 페이지네이션을 따라 반환한다.
func (c *Client) GetBlocks(blockID string) ([]Block, error) {
	var blocks []Block
	cursor := ""
	for {
		url := notionAPI + "/blocks/" + blockID + "/children?page_size=100"
		if cursor != "" {
			url += "&start_cursor=" + cursor
		}
		data, err := c.do("GET", url, nil)
		if err != nil {
			return nil, err
		}
		var res listResponse[Block]
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, err
		}
		blocks = append(blocks, res.Results...)
		if !res.HasMore {
			break
		}
		cursor = res.NextCursor
	}
	return blocks, nil
}
