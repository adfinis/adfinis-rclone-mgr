package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHttpHandlerRoot(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("GET", "/", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rw.Code)
	}
}

func TestHttpHandlerLoginMissingFields(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("POST", "/login", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

func TestHttpHandlerAuthInvalidState(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("GET", "/auth?state=invalid", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

func TestHttpHandlerGenerateMissingToken(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("POST", "/generate", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}

func TestHttpHandlerGenerateMissingOidc(t *testing.T) {
	h := newHttpHandler(context.Background(), func() {})
	req := httptest.NewRequest("POST", "/generate", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: base64.StdEncoding.EncodeToString([]byte("dummy"))})
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rw.Code)
	}
}
