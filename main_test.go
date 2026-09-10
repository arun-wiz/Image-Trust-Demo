package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest("GET", "/", nil))

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Hello from Wiz") {
		t.Fatal("response does not contain greeting")
	}
}

func TestHandlerNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest("GET", "/missing", nil))

	if recorder.Code != 404 {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}
