package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionAuth_Middleware_RedirectsHTMLRequestsInGUIMode(t *testing.T) {
	auth := NewSessionAuth("admin", "secret")
	protected := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/apps", nil)
	recorder := httptest.NewRecorder()
	protected.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status, got %d", recorder.Code)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}
}

func TestSessionAuth_Middleware_AllowsMissingGUIRoutesToReturn404InAPIOnlyMode(t *testing.T) {
	auth := NewSessionAuth("admin", "secret")
	auth.SetAPIOnly(true)
	protected := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	request := httptest.NewRequest(http.MethodGet, "/apps", nil)
	recorder := httptest.NewRecorder()
	protected.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", recorder.Code)
	}
}

func TestSessionAuth_Middleware_AllowsBasicAuthInAPIOnlyMode(t *testing.T) {
	auth := NewSessionAuth("admin", "secret")
	auth.SetAPIOnly(true)
	protected := auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	request.SetBasicAuth("admin", "secret")
	recorder := httptest.NewRecorder()
	protected.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected OK status, got %d", recorder.Code)
	}
}
