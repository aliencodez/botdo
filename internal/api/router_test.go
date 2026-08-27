package api

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/aliencodez/botdo/internal/store"
)

func TestProtectedAPIAndBrowserSession(t *testing.T) {
	s, err := store.NewJSONStore(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	server := httptest.NewServer(NewRouter(s, nil, http.FS(static), Config{
		APIKey:      "secret-token",
		CheckoutURL: "https://example.com/buy",
	}))
	defer server.Close()

	assertStatus(t, http.DefaultClient, http.MethodGet, server.URL+"/healthz", "", http.StatusOK)
	assertStatus(t, http.DefaultClient, http.MethodGet, server.URL+"/api/config", "", http.StatusOK)
	assertStatus(t, http.DefaultClient, http.MethodGet, server.URL+"/api/tasks", "", http.StatusUnauthorized)

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/tasks", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("bearer request status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	assertStatus(t, client, http.MethodPost, server.URL+"/api/session", `{"token":"wrong"}`, http.StatusUnauthorized)
	assertStatus(t, client, http.MethodPost, server.URL+"/api/session", `{"token":"secret-token"}`, http.StatusNoContent)
	assertStatus(t, client, http.MethodGet, server.URL+"/api/tasks", "", http.StatusOK)
}

func TestSecurityHeaders(t *testing.T) {
	s, err := store.NewJSONStore(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	NewRouter(s, nil, http.FS(static)).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("Content-Security-Policy is missing")
	}
}

func assertStatus(t *testing.T, client *http.Client, method, url, body string, want int) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("%s %s status = %d, want %d", method, url, res.StatusCode, want)
	}
}
