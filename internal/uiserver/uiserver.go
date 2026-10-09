// Package uiserver serves the desktop window its page on a loopback port.
//
// The page is Sessionary's web client with the desktop layer on top. It talks
// to the server exactly as it does in a browser — relative `/api/…` URLs,
// server-sent events, the per-launch token — so this server forwards `/api`
// and `/gateway` to the backend and serves the page's own files itself: the
// embedded build, or Vite's dev server during development.
package uiserver

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Server is the page's origin.
type Server struct {
	URL     string // http://127.0.0.1:<port>
	srv     *http.Server
	backend func() string
}

// Options configure the server.
type Options struct {
	// Backend returns the Sessionary server's base URL, or "" while there is none.
	Backend func() string
	// Files is the built frontend; DevURL, when set, is used instead (Vite, with hot reload).
	Files  fs.FS
	DevURL string
}

// Start listens on a free loopback port and serves until Close.
func Start(opts Options) (*Server, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	s := &Server{URL: "http://127.0.0.1:" + strconv.Itoa(port), backend: opts.Backend}

	var files http.Handler
	if opts.DevURL != "" {
		dev, err := url.Parse(opts.DevURL)
		if err != nil {
			return nil, err
		}
		p := httputil.NewSingleHostReverseProxy(dev)
		p.FlushInterval = -1
		files = p
	} else {
		files = spa(opts.Files)
	}

	api := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target, _ := url.Parse(s.backend())
			r.SetURL(target)
			// the server only answers loopback names and loopback origins, which this page is
			r.Out.Host = target.Host
			r.Out.Header.Del("X-Forwarded-For")
		},
		FlushInterval: -1, // server-sent events and terminal output stream through
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, context.Canceled) {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"Sessionary isn’t responding."}`))
		},
	}

	mux := http.NewServeMux()
	proxy := func(w http.ResponseWriter, r *http.Request) {
		if s.backend() == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"Sessionary is starting."}`))
			return
		}
		api.ServeHTTP(w, r)
	}
	mux.HandleFunc("/api/", proxy)
	mux.HandleFunc("/gateway/", proxy)
	mux.Handle("/", files)

	s.srv = &http.Server{
		Handler:           loopbackOnly(port, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() { _ = s.srv.Serve(l) }()
	return s, nil
}

// Close stops serving.
func (s *Server) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
}

// loopbackOnly refuses requests addressed to any other name, as the server
// itself does: a page of another site that rebinds a name to 127.0.0.1 must
// not read session history through this port.
func loopbackOnly(port int, next http.Handler) http.Handler {
	ok := map[string]bool{}
	for _, h := range []string{"127.0.0.1", "localhost", "[::1]"} {
		ok[h+":"+strconv.Itoa(port)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ok[r.Host] {
			http.Error(w, "Forbidden host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// spa serves the built files, with index.html for paths that match none.
func spa(fsys fs.FS) http.Handler {
	if fsys == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "The desktop frontend was not built.", http.StatusNotFound)
		})
	}
	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(fsys, name); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			name = "index.html"
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}
