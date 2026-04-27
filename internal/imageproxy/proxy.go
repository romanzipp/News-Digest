package imageproxy

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

func Handler(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		http.Error(w, "not an image", http.StatusBadRequest)
		return
	}

	etag := fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(rawURL)))

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	w.Header().Set("ETag", etag)

	io.Copy(w, io.LimitReader(resp.Body, 10<<20))
}
