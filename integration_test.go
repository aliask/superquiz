//go:build integration

// These tests hit the real SMH, Riddle and Saturday Paper sites through the
// proxy, following the same steps as public/app.js, and check that the data
// the page parses is still there. Run with: go test -tags integration ./...
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"testing"
)

// viaProxy fetches target through /smh, as fetchViaProxy in app.js does.
func viaProxy(t *testing.T, srv *httptest.Server, target string) string {
	t.Helper()
	resp, err := http.Get(srv.URL + "/smh?q=" + url.QueryEscape(target))
	if err != nil {
		t.Fatalf("proxy %s: %v", target, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("proxy %s: %v", target, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("proxy %s: status %d: %s", target, resp.StatusCode, body)
	}
	return string(body)
}

// scriptJSON decodes the JSON inside <script id="id"> into v.
func scriptJSON(t *testing.T, html, id string, v any) {
	t.Helper()
	re := regexp.MustCompile(`(?s)<script[^>]*\bid="` + regexp.QuoteMeta(id) + `"[^>]*>(.*?)</script>`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no <script id=%q> in page", id)
	}
	if err := json.Unmarshal([]byte(m[1]), v); err != nil {
		t.Fatalf("decoding <script id=%q>: %v", id, err)
	}
}

// latestSMHQuiz asks the SMH GraphQL API for the newest Superquiz article, as
// fetchSMHArticle in app.js does (the browser calls this API directly).
func latestSMHQuiz(t *testing.T) string {
	t.Helper()
	query := `query($brand: String!, $tagID: ID!, $newsroom: NewsroomType) {
		tag(brand: $brand, id: $tagID, newsroom: $newsroom) {
			assetsConnection(brand: $brand, count: 1, render: WEB, types: [article]) {
				assets { urls { canonical { path } } }
			}
		}
	}`
	reqBody, _ := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]string{"brand": "smh", "tagID": "6guy", "newsroom": "METRO"},
	})
	resp, err := http.Post("https://api.smh.com.au/graphql", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result struct {
		Data struct {
			Tag struct {
				AssetsConnection struct {
					Assets []struct {
						URLs struct {
							Canonical struct{ Path string }
						}
					}
				}
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding GraphQL response: %v", err)
	}
	assets := result.Data.Tag.AssetsConnection.Assets
	if len(assets) == 0 || assets[0].URLs.Canonical.Path == "" {
		t.Fatal("GraphQL response has no Superquiz articles")
	}
	return "https://www.smh.com.au" + assets[0].URLs.Canonical.Path
}

func TestSMHQuizLoads(t *testing.T) {
	srv := newServer(t)
	article := latestSMHQuiz(t)
	t.Logf("latest SMH quiz: %s", article)
	page := viaProxy(t, srv, article)

	// Same patterns as findRiddleId in app.js
	m := regexp.MustCompile(`riddle\.com/(?:view|embed/a)/([A-Za-z0-9]+)`).FindStringSubmatch(page)
	if m == nil {
		m = regexp.MustCompile(`data-rid-id="([A-Za-z0-9]+)"`).FindStringSubmatch(page)
	}
	if m == nil {
		t.Fatal("no Riddle embed in the SMH article")
	}

	// Mirrors parseRiddleQuiz: without the Referer the proxy adds, Riddle
	// serves a page with no quiz data.
	embed := viaProxy(t, srv, "https://www.riddle.com/embed/a/"+m[1])
	var riddle struct {
		Title  string
		Blocks []struct {
			Type    string
			Content struct {
				Title     string
				Flashcard struct{ BacksideTitle string }
			}
		}
	}
	scriptJSON(t, embed, "variable-data", &riddle)

	questions := 0
	for _, b := range riddle.Blocks {
		if b.Type != "Flashcard" {
			continue
		}
		questions++
		if b.Content.Title == "" || b.Content.Flashcard.BacksideTitle == "" {
			t.Errorf("flashcard %d is missing its question or answer", questions)
		}
	}
	if questions == 0 {
		t.Fatal("Riddle quiz has no Flashcard questions")
	}
	t.Logf("%q: %d questions", riddle.Title, questions)
}

func TestSaturdayPaperQuizLoads(t *testing.T) {
	srv := newServer(t)

	// Mirrors fetchSaturdayPaperQuiz
	index := viaProxy(t, srv, "https://www.thesaturdaypaper.com.au/quiz/")
	links := regexp.MustCompile(`href="(/quiz/\d{4}/\d{2}/\d{2})"`).FindAllStringSubmatch(index, -1)
	if len(links) == 0 {
		t.Fatal("no quiz links on the Saturday Paper quiz index")
	}
	paths := make([]string, len(links))
	for i, l := range links {
		paths[i] = l[1]
	}
	quizURL := "https://www.thesaturdaypaper.com.au" + slices.Max(paths)
	t.Logf("latest Saturday Paper quiz: %s", quizURL)

	// Mirrors parseNextDataQuiz
	page := viaProxy(t, srv, quizURL)
	var next struct {
		Props struct {
			PageProps struct {
				Node struct {
					FieldEdition   struct{ Title string } `json:"field_edition"`
					FieldQuestions []struct {
						FieldQuestion struct{ Processed string } `json:"field_question"`
						FieldAnswer   struct{ Processed string } `json:"field_answer"`
					} `json:"field_questions"`
				}
			}
		}
	}
	scriptJSON(t, page, "__NEXT_DATA__", &next)

	node := next.Props.PageProps.Node
	if len(node.FieldQuestions) == 0 {
		t.Fatal("Saturday Paper quiz has no questions")
	}
	for i, q := range node.FieldQuestions {
		if q.FieldQuestion.Processed == "" || q.FieldAnswer.Processed == "" {
			t.Errorf("question %d is missing its question or answer", i+1)
		}
	}
	t.Logf("%q: %d questions", node.FieldEdition.Title, len(node.FieldQuestions))
}
