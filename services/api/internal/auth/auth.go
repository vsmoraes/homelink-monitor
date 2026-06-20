package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/store"
)

const CookieName = "connection_monitor_session"

type contextKey string

const userKey contextKey = "user"

type Service struct {
	store *store.Store
}

func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

func (s *Service) EnsureInitialAdmin(ctx context.Context, username, password string) error {
	count, err := s.store.UserCount(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if strings.TrimSpace(username) == "" || password == "" {
		return errors.New("ADMIN_USERNAME and ADMIN_PASSWORD are required when creating the first user")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.store.CreateUser(ctx, strings.TrimSpace(username), hash, "admin", time.Now().UTC())
	return err
}

func (s *Service) Login(ctx context.Context, username, password string) (string, domain.User, error) {
	user, err := s.store.UserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return "", domain.User{}, err
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", domain.User{}, errors.New("invalid username or password")
	}
	token, err := randomToken()
	if err != nil {
		return "", domain.User{}, err
	}
	now := time.Now().UTC()
	if err := s.store.CreateSession(ctx, token, user.ID, now, now.Add(30*24*time.Hour)); err != nil {
		return "", domain.User{}, err
	}
	return token, user.User, nil
}

func (s *Service) UserForRequest(ctx context.Context, r *http.Request) (*domain.User, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return nil, nil
	}
	return s.store.UserBySession(ctx, cookie.Value, time.Now().UTC())
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, token)
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func ContextWithUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userKey).(domain.User)
	return user, ok
}

func randomToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
}
