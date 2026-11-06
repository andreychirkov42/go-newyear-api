// Package api exposes the New Year countdown as an HTTP JSON API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/andreychirkov42/go-newyear-api/internal/newyear"
)

// DateLayout is the only accepted format of the date query parameter
// (ISO 8601 calendar date, for example 2024-02-29).
const DateLayout = time.DateOnly

// DateParam is the name of the query parameter that holds the date.
const DateParam = "date"

// Paths of the API endpoints.
const (
	DaysPath   = "/api/v1/days-until-new-year"
	HealthPath = "/healthz"
)

// DaysResponse is the successful response of the days endpoint.
type DaysResponse struct {
	Date        string `json:"date"`
	NextNewYear string `json:"next_new_year"`
	DaysLeft    int    `json:"days_left"`
}

// ErrorResponse is returned for any request that cannot be processed.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse is the response of the health check endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}

// NewHandler returns the HTTP handler of the service. now supplies the
// current time for requests without a date; logger receives one record per
// request.
func NewHandler(now func() time.Time, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	// GET patterns also match HEAD. Patterns without a method catch every
	// other method on the same path and answer 405 in JSON, and "/" answers
	// 404 in JSON instead of the plain-text defaults of ServeMux.
	mux.HandleFunc("GET "+DaysPath, daysHandler(now))
	mux.HandleFunc(DaysPath, methodNotAllowedHandler)
	mux.HandleFunc("GET "+HealthPath, healthHandler)
	mux.HandleFunc(HealthPath, methodNotAllowedHandler)
	mux.HandleFunc("/", notFoundHandler)

	return logRequests(logger, mux)
}

func daysHandler(now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day, err := requestDate(r, now)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, DaysResponse{
			Date:        day.Format(DateLayout),
			NextNewYear: newyear.Next(day).Format(DateLayout),
			DaysLeft:    newyear.DaysUntil(day),
		})
	}
}

// requestDate returns the date from the query string, or the current date
// when the parameter is absent.
func requestDate(r *http.Request, now func() time.Time) (time.Time, error) {
	values, ok := r.URL.Query()[DateParam]
	if !ok {
		return now(), nil
	}
	if len(values) != 1 {
		return time.Time{}, errDuplicateDate
	}

	day, err := time.Parse(DateLayout, values[0])
	if err != nil {
		return time.Time{}, errInvalidDate
	}
	return day, nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "not found"})
}

func methodNotAllowedHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Allow", "GET, HEAD")
	writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// The status line is already sent, so an encoding error can only be
	// caused by a broken connection and there is no one left to report it to.
	_ = json.NewEncoder(w).Encode(body)
}
