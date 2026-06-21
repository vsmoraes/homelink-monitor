package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"homelink-monitor/services/api/internal/auth"
)

func TestHashPasswordRejectsBcryptTruncationRange(t *testing.T) {
	if _, err := auth.HashPassword(strings.Repeat("a", 73)); err == nil {
		t.Fatal("expected long password to be rejected")
	}
}

func TestValidateUsername(t *testing.T) {
	valid := []string{"admin", "user.name", "user-name", "user_name", "user@example"}
	for _, username := range valid {
		if err := auth.ValidateUsername(username); err != nil {
			t.Fatalf("expected %q to be valid: %v", username, err)
		}
	}

	invalid := []string{"", "bad name", "bad/name", strings.Repeat("a", 65)}
	for _, username := range invalid {
		if err := auth.ValidateUsername(username); err == nil {
			t.Fatalf("expected %q to be invalid", username)
		}
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	service := auth.NewService(nil)
	service.SetCookieSecure(true)

	res := httptest.NewRecorder()
	service.SetSessionCookie(res, "token")

	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure {
		t.Fatalf("expected secure httponly cookie: %#v", cookie)
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected strict samesite cookie, got %v", cookie.SameSite)
	}
}
