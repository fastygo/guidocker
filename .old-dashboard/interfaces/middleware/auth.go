package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookieName = "paas_session"
	sessionTTL        = 24 * time.Hour
)

// SessionAuth provides form login and Basic Auth support.
type SessionAuth struct {
	user     string
	pass     string
	now      func() time.Time
	mu       sync.RWMutex
	sessions map[string]time.Time
}

// NewSessionAuth creates a session-based auth helper.
func NewSessionAuth(user, pass string) *SessionAuth {
	return &SessionAuth{
		user:     user,
		pass:     pass,
		now:      time.Now,
		sessions: make(map[string]time.Time),
	}
}

// BasicAuth returns an auth middleware with an isolated in-memory session store.
func BasicAuth(user, pass string) func(http.Handler) http.Handler {
	return NewSessionAuth(user, pass).Middleware()
}

// Middleware protects all routes except /login.
func (a *SessionAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/login" {
				next.ServeHTTP(w, r)
				return
			}

			if a.hasValidSession(r) {
				next.ServeHTTP(w, r)
				return
			}

			if username, password, ok := r.BasicAuth(); ok && a.validCredentials(username, password) {
				a.setSessionCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("WWW-Authenticate", `Basic realm="PaaS"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			http.Redirect(w, r, "/login", http.StatusSeeOther)
		})
	}
}

// LoginHandler serves the login screen and creates a session cookie.
func (a *SessionAuth) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			renderLoginPage(w, "")
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				renderLoginPageWithStatus(w, "Invalid login form", http.StatusBadRequest)
				return
			}

			if !a.validCredentials(r.FormValue("username"), r.FormValue("password")) {
				renderLoginPageWithStatus(w, "Invalid username or password", http.StatusUnauthorized)
				return
			}

			a.setSessionCookie(w)
			http.Redirect(w, r, "/apps", http.StatusSeeOther)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (a *SessionAuth) hasValidSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return false
	}

	a.mu.RLock()
	expiresAt, ok := a.sessions[cookie.Value]
	a.mu.RUnlock()
	if !ok {
		return false
	}

	if expiresAt.Before(a.now()) {
		a.mu.Lock()
		delete(a.sessions, cookie.Value)
		a.mu.Unlock()
		return false
	}

	return true
}

func (a *SessionAuth) validCredentials(user, pass string) bool {
	return subtle.ConstantTimeCompare([]byte(user), []byte(a.user)) == 1 &&
		subtle.ConstantTimeCompare([]byte(pass), []byte(a.pass)) == 1
}

func (a *SessionAuth) setSessionCookie(w http.ResponseWriter) {
	token := make([]byte, 32)
	_, _ = rand.Read(token)

	sessionID := hex.EncodeToString(token)
	expiresAt := a.now().Add(sessionTTL)

	a.mu.Lock()
	a.sessions[sessionID] = expiresAt
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func renderLoginPage(w http.ResponseWriter, message string) {
	renderLoginPageWithStatus(w, message, http.StatusOK)
}

func renderLoginPageWithStatus(w http.ResponseWriter, message string, status int) {
	messageBlock := ""
	if strings.TrimSpace(message) != "" {
		messageBlock = `<div style="padding:12px 14px;border:1px solid #fecaca;border-radius:8px;background:#fef2f2;color:#991b1b;font-size:14px;">` + html.EscapeString(message) + `</div>`
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>PaaS Login</title>
</head>
<body style="margin:0;font-family:Arial,sans-serif;background:#f8fafc;color:#0f172a;">
	<div style="min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;">
		<form method="post" action="/login" style="width:100%;max-width:420px;background:#ffffff;border:1px solid #e2e8f0;border-radius:12px;padding:24px;box-shadow:0 12px 30px rgba(15,23,42,0.08);display:flex;flex-direction:column;gap:16px;">
			<div>
				<div style="font-size:14px;color:#475569;">PaaS Console</div>
				<h1 style="margin:8px 0 0;font-size:28px;">Sign in</h1>
			</div>
			` + messageBlock + `
			<label style="display:flex;flex-direction:column;gap:8px;font-size:14px;">
				<span>Username</span>
				<input type="text" name="username" autocomplete="username" required style="padding:10px 12px;border:1px solid #cbd5e1;border-radius:8px;" />
			</label>
			<label style="display:flex;flex-direction:column;gap:8px;font-size:14px;">
				<span>Password</span>
				<input type="password" name="password" autocomplete="current-password" required style="padding:10px 12px;border:1px solid #cbd5e1;border-radius:8px;" />
			</label>
			<button type="submit" style="padding:12px 16px;border:none;border-radius:8px;background:#0f172a;color:#ffffff;font-size:14px;cursor:pointer;">Login</button>
		</form>
	</div>
</body>
</html>`))
}
