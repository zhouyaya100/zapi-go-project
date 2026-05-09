package core

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zapi/zapi-go/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func CreateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{"sub": fmt.Sprintf("%d", userID), "exp": time.Now().Add(time.Duration(config.Cfg.Security.JWTExpireHours) * time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Cfg.Security.SecretKey))
}

func ParseJWT(tokenStr string) (uint, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.Cfg.Security.SecretKey), nil
	})
	if err != nil || !token.Valid { return 0, fmt.Errorf("invalid token") }
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok { return 0, fmt.Errorf("invalid claims") }
	sub, ok := claims["sub"].(string)
	if !ok { return 0, fmt.Errorf("invalid sub") }
	var uid uint; fmt.Sscanf(sub, "%d", &uid); return uid, nil
}

func GenerateAPIKey() string { b := make([]byte, 24); rand.Read(b); return "sk-" + hex.EncodeToString(b) }
func HashPassword(pw string) (string, error) { h, e := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost); return string(h), e }
func CheckPassword(hash, pw string) bool { return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil }

func ValidateUsername(name string) string {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 32 { return "\u7528\u6237\u540d\u97002-32\u4e2a\u5b57\u7b26" }
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || (ch >= '\u4e00' && ch <= '\u9fff')) {
			return "\u7528\u6237\u540d\u53ea\u80fd\u5305\u542b\u5b57\u6bcd\u3001\u6570\u5b57\u3001\u4e0b\u5212\u7ebf\u548c\u4e2d\u6587"
		}
	}
	return ""
}

func ValidatePasswordStrength(pw string) string {
	if len(pw) < config.Cfg.Registration.MinPasswordLength { return fmt.Sprintf("\u5bc6\u7801\u957f\u5ea6\u4e0d\u80fd\u5c11\u4e8e%d\u4f4d", config.Cfg.Registration.MinPasswordLength) }
	if len(pw) > 128 { return "\u5bc6\u7801\u957f\u5ea6\u4e0d\u80fd\u8d85\u8fc7128\u4f4d" }
	hasLetter, hasDigit := false, false
	for _, ch := range pw {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') { hasLetter = true }
		if ch >= '0' && ch <= '9' { hasDigit = true }
	}
	if !hasLetter { return "\u5bc6\u7801\u5fc5\u987b\u5305\u542b\u81f3\u5c11\u4e00\u4e2a\u5b57\u6bcd" }
	if !hasDigit { return "\u5bc6\u7801\u5fc5\u987b\u5305\u542b\u81f3\u5c11\u4e00\u4e2a\u6570\u5b57" }
	return ""
}

type loginRecord struct { Attempts int; LockedUntil time.Time }
var loginAttempts = make(map[string]*loginRecord)
var loginAttemptsMu sync.RWMutex

func CheckLoginRate(key string) error {
	loginAttemptsMu.RLock(); r, ok := loginAttempts[key]; loginAttemptsMu.RUnlock()
	if !ok { return nil }
	if r.Attempts >= 5 && time.Now().Before(r.LockedUntil) { return fmt.Errorf("\u767b\u5f55\u5c1d\u8bd5\u8fc7\u4e8e\u9891\u7e41\uff0c\u8bf75\u5206\u949f\u540e\u518d\u8bd5") }
	return nil
}

func RecordLoginFailure(key string) {
	loginAttemptsMu.Lock(); defer loginAttemptsMu.Unlock()
	r, ok := loginAttempts[key]
	if !ok { r = &loginRecord{}; loginAttempts[key] = r }
	r.Attempts++
	if r.Attempts >= 5 { r.LockedUntil = time.Now().Add(5 * time.Minute) }
}

func RecordLoginSuccess(key string) { loginAttemptsMu.Lock(); delete(loginAttempts, key); loginAttemptsMu.Unlock() }

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			loginAttemptsMu.Lock()
			now := time.Now()
			for k, v := range loginAttempts {
				// Clean locked entries past their lock time
				if v.Attempts >= 5 && now.After(v.LockedUntil) {
					delete(loginAttempts, k)
				// Clean entries older than 30 minutes regardless of attempt count
				} else if now.Sub(v.LockedUntil.Add(5*time.Minute)) > 25*time.Minute || (v.LockedUntil.IsZero() && v.Attempts > 0 && len(loginAttempts) > 1000) {
					delete(loginAttempts, k)
				}
			}
			loginAttemptsMu.Unlock()
		}
	}()
}

func IsAdminToken(token string) bool { return subtle.ConstantTimeCompare([]byte(token), []byte(config.Cfg.Security.AdminToken)) == 1 }
