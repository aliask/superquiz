package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const addr = "0.0.0.0:8080"

//go:embed public
var publicFS embed.FS

// allowedHosts are the only sites the proxy will fetch from, so it can't be
// used to reach anything else (e.g. services on the host's network).
var allowedHosts = map[string]bool{
	"www.smh.com.au":              true,
	"www.theage.com.au":           true,
	"www.thesaturdaypaper.com.au": true,
	"www.riddle.com":              true,
}

var errNotAllowed = errors.New("URL not allowed")

func checkURL(u *url.URL) error {
	if u.Scheme != "https" || u.User != nil || u.Port() != "" || !allowedHosts[u.Hostname()] {
		return fmt.Errorf("%w: %s", errNotAllowed, u.Redacted())
	}
	return nil
}

func newClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		// Redirects must stay on allowed hosts too
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return checkURL(req.URL)
		},
	}
}

func newHandler() http.Handler {
	static, err := fs.Sub(publicFS, "public")
	if err != nil {
		panic(err)
	}
	client := newClient()

	index := pinnedIndex(static)

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		// Cloudflare doesn't cache HTML, but it does cache app.js whatever we
		// send, so the page must always come fresh and name the current JS
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
	})
	mux.HandleFunc("GET /smh", func(w http.ResponseWriter, r *http.Request) {
		body, err := fetch(client, r.URL.Query().Get("q"))
		if err != nil {
			log.Print(err)
			if errors.Is(err, errNotAllowed) {
				http.Error(w, "URL not allowed", http.StatusBadRequest)
			} else {
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
			return
		}
		// The page only reads this as text, so don't let the browser render it
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(body)
	})
	return mux
}

// pinnedIndex returns index.html with app.js pinned to a hash of its content,
// so a deploy that changes the JS also changes its URL and misses any cache.
func pinnedIndex(static fs.FS) []byte {
	index, err := fs.ReadFile(static, "index.html")
	if err != nil {
		panic(err)
	}
	js, err := fs.ReadFile(static, "app.js")
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(js)
	const tag = `src="/app.js"`
	if strings.Count(string(index), tag) != 1 {
		panic("index.html must load app.js exactly once, via " + tag)
	}
	pinned := fmt.Sprintf(`src="/app.js?v=%s"`, hex.EncodeToString(sum[:6]))
	return []byte(strings.Replace(string(index), tag, pinned, 1))
}

// fetch returns the body of target, whatever the response status.
func fetch(client *http.Client, target string) ([]byte, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if err := checkURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	// Riddle refuses to serve embeds unless it can see the embedding domain
	if u.Hostname() == "www.riddle.com" {
		req.Header.Set("Referer", "https://www.smh.com.au/")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func main() {
	log.Printf("Running on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, newHandler()))
}
