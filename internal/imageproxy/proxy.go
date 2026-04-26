package imageproxy

import (
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

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")

	io.Copy(w, io.LimitReader(resp.Body, 10<<20)) // 10MB limit
}
