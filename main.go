package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
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

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(static))
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
