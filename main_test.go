package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(newHandler())
	t.Cleanup(srv.Close)
	return srv
}

func TestStaticFiles(t *testing.T) {
	srv := newServer(t)
	for path, want := range map[string]string{
		"/":       "<title>Superquiz</title>",
		"/app.js": "function loadQuiz",
	} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("GET %s: status %d, want body containing %q", path, resp.StatusCode, want)
		}
		if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
			t.Errorf("GET %s: Cache-Control %q, want %q", path, got, "no-cache")
		}
	}
}

func TestProxyRefusesOtherURLs(t *testing.T) {
	var hits atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer internal.Close()

	srv := newServer(t)
	for _, target := range []string{
		internal.URL,
		"",
		"file:///etc/passwd",
		"http://www.smh.com.au/",
		"https://www.smh.com.au:8443/",
		"https://user:pass@www.smh.com.au/",
		"https://www.smh.com.au.example.com/",
		"https://smh.com.au/",
		"https://localhost/",
		"https://169.254.169.254/latest/meta-data/",
	} {
		resp, err := http.Get(srv.URL + "/smh?q=" + url.QueryEscape(target))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("proxy %q: status %d, want %d", target, resp.StatusCode, http.StatusBadRequest)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("internal server was hit %d times", n)
	}
}

func TestRedirectsMustStayOnAllowedHosts(t *testing.T) {
	check := newClient().CheckRedirect
	for target, allowed := range map[string]bool{
		"https://www.thesaturdaypaper.com.au/quizzes": true,
		"http://127.0.0.1:8080/":                      false,
		"https://169.254.169.254/":                    false,
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if err := check(req, nil); (err == nil) != allowed {
			t.Errorf("redirect to %s: err = %v, want allowed = %v", target, err, allowed)
		}
	}
}
