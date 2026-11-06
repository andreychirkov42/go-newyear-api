package api_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreychirkov42/go-newyear-api/internal/api"
)

// fixedNow is the "current" time used by the test server.
var fixedNow = time.Date(2024, time.December, 30, 18, 0, 0, 0, time.UTC)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(api.NewHandler(func() time.Time { return fixedNow }, logger))
	t.Cleanup(srv.Close)
	return srv
}

// get performs a real HTTP request and decodes the JSON body into out.
func get(t *testing.T, url string, out any) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("GET %s: Content-Type = %q, want application/json", url, ct)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("GET %s: decode body: %v", url, err)
	}
	return resp
}

func TestDaysUntilNewYear(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name  string
		query string
		want  api.DaysResponse
	}{
		{
			name:  "without date uses current date",
			query: "",
			want:  api.DaysResponse{Date: "2024-12-30", NextNewYear: "2025-01-01", DaysLeft: 2},
		},
		{
			name:  "leap day",
			query: "?date=2024-02-29",
			want:  api.DaysResponse{Date: "2024-02-29", NextNewYear: "2025-01-01", DaysLeft: 307},
		},
		{
			name:  "start of year",
			query: "?date=2023-01-01",
			want:  api.DaysResponse{Date: "2023-01-01", NextNewYear: "2024-01-01", DaysLeft: 365},
		},
		{
			name:  "end of year",
			query: "?date=2023-12-31",
			want:  api.DaysResponse{Date: "2023-12-31", NextNewYear: "2024-01-01", DaysLeft: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got api.DaysResponse
			resp := get(t, srv.URL+api.DaysPath+tt.query, &got)

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}
			if got != tt.want {
				t.Errorf("body = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDaysUntilNewYearInvalidInput(t *testing.T) {
	srv := newTestServer(t)

	queries := []string{
		"?date=",
		"?date=2024-13-01",
		"?date=2023-02-29",
		"?date=31.12.2024",
		"?date=2024-1-1",
		"?date=2024-01-01T00:00:00Z",
		"?date=abc",
		"?date=2024-01-01&date=2024-01-02",
	}

	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			var got api.ErrorResponse
			resp := get(t, srv.URL+api.DaysPath+q, &got)

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
			if got.Error == "" {
				t.Error("error message is empty")
			}
		})
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)

	var got api.HealthResponse
	resp := get(t, srv.URL+api.HealthPath, &got)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got.Status != "ok" {
		t.Errorf("status field = %q, want %q", got.Status, "ok")
	}
}

func TestUnknownPath(t *testing.T) {
	srv := newTestServer(t)

	var got api.ErrorResponse
	resp := get(t, srv.URL+"/no-such-page", &got)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)

	for _, path := range []string{api.DaysPath, api.HealthPath} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader("{}"))
			if err != nil {
				t.Fatalf("POST %s: %v", path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
			}
			if allow := resp.Header.Get("Allow"); allow != "GET, HEAD" {
				t.Errorf("Allow = %q, want %q", allow, "GET, HEAD")
			}
		})
	}
}
