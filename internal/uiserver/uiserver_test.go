package uiserver

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

func backendNamed(name string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			w.Header().Set("Content-Type", "text/event-stream")
			for i := 0; i < 3; i++ {
				io.WriteString(w, "data: "+name+"\n\n")
				w.(http.Flusher).Flush()
				time.Sleep(50 * time.Millisecond)
			}
			return
		}
		io.WriteString(w, name+" "+r.Host)
	}))
}

func get(t *testing.T, url string, host string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if host != "" {
		req.Host = host
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func TestProxyFollowsTheBackend(t *testing.T) {
	a, b := backendNamed("first"), backendNamed("second")
	defer a.Close()
	defer b.Close()
	var current atomic.Value
	current.Store("")
	s, err := Start(Options{Backend: func() string { return current.Load().(string) }, Files: fstest.MapFS{"index.html": {Data: []byte("page")}}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if code, _ := get(t, s.URL+"/api/status", ""); code != http.StatusServiceUnavailable {
		t.Fatalf("without a backend: %d, want 503", code)
	}
	current.Store(a.URL)
	code, body := get(t, s.URL+"/api/status", "")
	if code != 200 || !strings.HasPrefix(body, "first ") {
		t.Fatalf("got %d %q", code, body)
	}
	// the server checks Host: it must see its own loopback address, not the page's
	if !strings.Contains(body, strings.TrimPrefix(a.URL, "http://")) {
		t.Fatalf("backend saw Host %q", body)
	}
	current.Store(b.URL) // restarted on another port
	if _, body := get(t, s.URL+"/gateway/v1/models", ""); !strings.HasPrefix(body, "second ") {
		t.Fatalf("after a restart: %q", body)
	}
	if _, body := get(t, s.URL+"/m/local/sessions", ""); body != "page" {
		t.Fatalf("a page path should serve index.html, got %q", body)
	}
}

func TestEventsStream(t *testing.T) {
	a := backendNamed("live")
	defer a.Close()
	s, _ := Start(Options{Backend: func() string { return a.URL }})
	defer s.Close()
	res, err := http.Get(s.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	start := time.Now()
	line, _ := bufio.NewReader(res.Body).ReadString('\n')
	if line != "data: live\n" || time.Since(start) > 40*time.Millisecond {
		t.Fatalf("first event %q after %s: events must stream, not buffer", line, time.Since(start))
	}
}

func TestRefusesOtherHosts(t *testing.T) {
	s, _ := Start(Options{Backend: func() string { return "" }, Files: fstest.MapFS{"index.html": {Data: []byte("page")}}})
	defer s.Close()
	if code, _ := get(t, s.URL+"/", "attacker.example:80"); code != http.StatusForbidden {
		t.Fatalf("a rebinding Host got %d, want 403", code)
	}
	if code, _ := get(t, s.URL+"/", ""); code != 200 {
		t.Fatalf("own Host got %d", code)
	}
}
