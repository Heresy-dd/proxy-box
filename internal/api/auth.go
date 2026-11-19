package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"proxybox/internal/store"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	store     *store.Store
	jwtSecret []byte
}

func NewAuthService(s *store.Store) *AuthService {
	// Generate or retrieve persistent JWT signing key
	key, err := s.GetSetting("jwt_secret")
	if err != nil || key == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		key = hex.EncodeToString(b)
		_ = s.SetSetting("jwt_secret", key)
	}

	return &AuthService{
		store:     s,
		jwtSecret: []byte(key),
	}
}

func (a *AuthService) IsInitialized() bool {
	pwdHash, err := a.store.GetSetting("admin_password_hash")
	return err == nil && pwdHash != ""
}

func (a *AuthService) SetupPassword(password string) error {
	if a.IsInitialized() {
		return fmt.Errorf("admin password already initialized")
	}
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return a.store.SetSetting("admin_password_hash", string(hash))
}

func (a *AuthService) VerifyPassword(password string) bool {
	hash, err := a.store.GetSetting("admin_password_hash")
	if err != nil || hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (a *AuthService) GenerateToken(duration time.Duration) string {
	expire := time.Now().Add(duration).Unix()
	payload := fmt.Sprintf("admin:%d", expire)
	mac := hmac.New(sha256.New, a.jwtSecret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s.%s", payload, sig)
}

func (a *AuthService) ValidateToken(tokenStr string) bool {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return false
	}
	payload := parts[0]
	sig := parts[1]

	mac := hmac.New(sha256.New, a.jwtSecret)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false
	}

	pParts := strings.Split(payload, ":")
	if len(pParts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(pParts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}

	return true
}

// Middleware guards routes requiring admin auth
func (a *AuthService) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow initialization and login routes without token
		path := r.URL.Path
		if path == "/api/auth/status" || path == "/api/auth/setup" || path == "/api/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if not initialized yet, prompt setup
		if !a.IsInitialized() {
			http.Error(w, `{"error":"system not initialized, please complete initial setup"}`, http.StatusPreconditionRequired)
			return
		}

		// Read Bearer token
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token") // allow query param for SSE stream
		}

		if token == "" || !a.ValidateToken(token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}
