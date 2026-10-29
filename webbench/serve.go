// webbench/serve.go

package webbench

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Serve publishes a built SPA over loopback HTTP, since file:// breaks module scripts and fetch.
func Serve(dir string) (url string, stop func(), err error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", nil, err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return "", nil, fmt.Errorf("webbench: %s is not a directory", dir)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	files := http.FileServer(http.Dir(abs))
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Unknown paths serve index.html, except under asset directories, where a
		// missing chunk must stay a 404.
		clean := filepath.Clean(r.URL.Path)
		if !strings.Contains(clean, "_nuxt") && !strings.Contains(clean, "assets") {
			if _, err := os.Stat(filepath.Join(abs, clean)); os.IsNotExist(err) {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		files.ServeHTTP(w, r)
	})}

	go func() { _ = srv.Serve(ln) }()

	return "http://" + ln.Addr().String() + "/", func() { _ = srv.Close() }, nil
}
