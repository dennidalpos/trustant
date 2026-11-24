package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimitBodyRejectsOversizeByContentLength(t *testing.T) {
	reached := false
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))

	req := httptest.NewRequest("POST", "/api/configuration", strings.NewReader("{}"))
	req.ContentLength = maxRequestBody + 1
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge || reached {
		t.Fatalf("oversize JSON request: status %d, handler reached %v", rec.Code, reached)
	}

	// The upload route allows more than ordinary API calls, but not unlimited.
	req = httptest.NewRequest("POST", "/api/upload", strings.NewReader("x"))
	req.ContentLength = maxUploadBody + 1
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize upload: status %d", rec.Code)
	}
	req = httptest.NewRequest("POST", "/api/upload", strings.NewReader("x"))
	req.ContentLength = maxRequestBody + 1
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusRequestEntityTooLarge || !reached {
		t.Fatalf("upload above the API limit must reach the handler: status %d", rec.Code)
	}
}

func TestLimitBodyCapsStreamsWithoutContentLength(t *testing.T) {
	var readErr error
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.Copy(io.Discard, r.Body)
	}))
	req := httptest.NewRequest("POST", "/api/configuration", io.LimitReader(zeroReader{}, maxRequestBody+1024))
	req.ContentLength = -1
	h.ServeHTTP(httptest.NewRecorder(), req)
	var tooLarge *http.MaxBytesError
	if !errors.As(readErr, &tooLarge) {
		t.Fatalf("expected MaxBytesError, got %v", readErr)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
