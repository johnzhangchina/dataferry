package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type AuthMiddleware struct {
	password string
	mu       sync.RWMutex
	tokens   map[string]time.Time // token -> expiry
}

func NewAuthMiddleware(password string) *AuthMiddleware {
	return &AuthMiddleware{
		password: password,
		tokens:   make(map[string]time.Time),
	}
}

// Enabled returns true if a password is configured.
func (a *AuthMiddleware) Enabled() bool {
	return a.password != ""
}

// Login POST /api/login
func (a *AuthMiddleware) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(a.password)) != 1 {
		writeError(w, http.StatusUnauthorized, "incorrect password")
		return
	}

	token := generateToken()
	expiry := time.Now().Add(24 * time.Hour)

	a.mu.Lock()
	a.tokens[token] = expiry
	a.cleanup()
	a.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "dataferry_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Logout POST /api/logout
func (a *AuthMiddleware) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("dataferry_token"); err == nil {
		a.mu.Lock()
		delete(a.tokens, cookie.Value)
		a.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "dataferry_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CheckAuth GET /api/auth — returns auth status
func (a *AuthMiddleware) CheckAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": a.isAuthenticated(r),
		"required":      a.Enabled(),
	})
}

// Protect wraps an http.Handler, requiring authentication.
func (a *AuthMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		if !a.isAuthenticated(r) {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ProtectFunc wraps an http.HandlerFunc.
func (a *AuthMiddleware) ProtectFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			next(w, r)
			return
		}
		if !a.isAuthenticated(r) {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next(w, r)
	}
}

func (a *AuthMiddleware) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("dataferry_token")
	if err != nil {
		return false
	}
	a.mu.RLock()
	expiry, ok := a.tokens[cookie.Value]
	a.mu.RUnlock()
	return ok && time.Now().Before(expiry)
}

func (a *AuthMiddleware) cleanup() {
	now := time.Now()
	for token, expiry := range a.tokens {
		if now.After(expiry) {
			delete(a.tokens, token)
		}
	}
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
