package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AuthMiddleware struct {
	password string
	sessions map[string]time.Time
	mu       sync.RWMutex
}

func NewAuthMiddleware(password string) *AuthMiddleware {
	return &AuthMiddleware{
		password: password,
		sessions: make(map[string]time.Time),
	}
}

// RequireAuth middleware checks for valid session or prompts for login
func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check session cookie
		cookie, err := r.Cookie("session")
		if err == nil && a.validateSession(cookie.Value) {
			next.ServeHTTP(w, r)
			return
		}

		// For API requests, return 401
		if strings.HasPrefix(r.URL.Path, "/admin/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		// Redirect to login
		if r.URL.Path != "/admin/login" {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CheckAuth handles GET /admin/api/auth/check - returns 200 if authenticated
func (a *AuthMiddleware) CheckAuth(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil && a.validateSession(cookie.Value) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": false})
}

// LoginAPI handles POST /admin/api/login - JSON API login for SPA
func (a *AuthMiddleware) LoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	// Try JSON body first
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}
	} else {
		// Form data
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid form"})
			return
		}
		req.Password = r.FormValue("password")
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.password)) != 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid password"})
		return
	}

	// Create session
	sessionID := a.createSession()
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7, // 7 days
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Login handles POST /admin/login (HTML form)
func (a *AuthMiddleware) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show login form
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(loginHTML))
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(password), []byte(a.password)) != 1 {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(loginHTML + `<p style="color: #e94560;">Invalid password</p>`))
		return
	}

	// Create session
	sessionID := a.createSession()
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7, // 7 days
	})

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Logout handles GET /admin/logout
func (a *AuthMiddleware) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		a.mu.Lock()
		delete(a.sessions, cookie.Value)
		a.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// For API requests, return JSON
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *AuthMiddleware) createSession() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	sessionID := hex.EncodeToString(bytes)

	a.mu.Lock()
	a.sessions[sessionID] = time.Now().Add(7 * 24 * time.Hour)
	a.mu.Unlock()

	return sessionID
}

func (a *AuthMiddleware) validateSession(sessionID string) bool {
	a.mu.RLock()
	expiry, exists := a.sessions[sessionID]
	a.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		a.mu.Lock()
		delete(a.sessions, sessionID)
		a.mu.Unlock()
		return false
	}

	return true
}

// CleanupExpiredSessions removes expired sessions periodically
func (a *AuthMiddleware) CleanupExpiredSessions() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for id, expiry := range a.sessions {
		if now.After(expiry) {
			delete(a.sessions, id)
		}
	}
}

const loginHTML = `<!DOCTYPE html>
<html>
<head>
	<title>Login - Torrent Server</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
			background: #1a1a2e;
			color: #eee;
			display: flex;
			justify-content: center;
			align-items: center;
			min-height: 100vh;
			margin: 0;
		}
		.login-box {
			background: #16213e;
			padding: 40px;
			border-radius: 8px;
			width: 300px;
		}
		h1 {
			color: #e94560;
			margin-top: 0;
			text-align: center;
		}
		input {
			width: 100%;
			padding: 12px;
			margin: 10px 0;
			border: 1px solid #333;
			border-radius: 4px;
			background: #0f3460;
			color: #eee;
			box-sizing: border-box;
		}
		button {
			width: 100%;
			background: #e94560;
			color: white;
			border: none;
			padding: 12px;
			border-radius: 4px;
			cursor: pointer;
			font-size: 16px;
		}
		button:hover {
			background: #ff6b6b;
		}
	</style>
</head>
<body>
	<div class="login-box">
		<h1>Torrent Server</h1>
		<form method="POST" action="/admin/login">
			<input type="password" name="password" placeholder="Admin Password" required autofocus>
			<button type="submit">Login</button>
		</form>
	</div>
</body>
</html>`
