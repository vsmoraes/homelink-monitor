package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"homelink-monitor/services/api/internal/auth"
	"homelink-monitor/services/api/internal/domain"
	"homelink-monitor/services/api/internal/monitoring"
	"homelink-monitor/services/api/internal/store"
)

type Server struct {
	store     *store.Store
	monitor   *monitoring.Service
	auth      *auth.Service
	log       *slog.Logger
	staticDir string
}

func New(st *store.Store, monitor *monitoring.Service, authService *auth.Service, log *slog.Logger, staticDir string) *Server {
	return &Server{store: st, monitor: monitor, auth: authService, log: log, staticDir: staticDir}
}

func (s *Server) Routes() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		status := http.StatusInternalServerError
		message := err.Error()
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			status = httpErr.Code
			if text, ok := httpErr.Message.(string); ok {
				message = text
			} else {
				message = http.StatusText(status)
			}
		}
		_ = c.JSON(status, map[string]string{"error": message})
	}
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "0",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		ReferrerPolicy:        "same-origin",
		ContentSecurityPolicy: "",
	}))

	e.GET("/api/health", s.health)
	e.POST("/api/auth/login", s.login)
	e.POST("/api/auth/logout", s.logout)

	api := e.Group("/api", s.authMiddleware)
	api.GET("/auth/me", s.me)
	api.GET("/summary", s.summary)
	api.GET("/speed-tests", s.speedTests)
	api.POST("/speed-tests/run", s.runSpeedTest)
	api.GET("/speed-tests/latest", s.latestSpeedTest)
	api.GET("/latency", s.latency)
	api.GET("/latency/summary", s.latencySummary)
	api.GET("/dns-checks", s.dnsChecks)
	api.GET("/dns-checks/latest", s.latestDNS)
	api.GET("/outages", s.outages)
	api.GET("/outages/active", s.activeOutage)
	api.GET("/settings", s.settings)
	api.PUT("/settings", s.saveSettings)
	api.GET("/users", s.users)
	api.POST("/users", s.createUser)
	api.PUT("/users/:id", s.updateUser)
	api.DELETE("/users/:id", s.deleteUser)

	e.GET("/*", s.static)
	return e
}

func (s *Server) health(c echo.Context) error {
	dbStatus := "ok"
	if err := s.store.Ping(c.Request().Context()); err != nil {
		dbStatus = "error"
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "error", "database": dbStatus, "error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "ok", "database": dbStatus})
}

func (s *Server) login(c echo.Context) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	token, user, err := s.auth.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return writeError(c, http.StatusUnauthorized, err)
	}
	auth.SetSessionCookie(c.Response(), token)
	return c.JSON(http.StatusOK, map[string]any{"user": user})
}

func (s *Server) logout(c echo.Context) error {
	if cookie, err := c.Cookie(auth.CookieName); err == nil {
		if err := s.auth.Logout(c.Request().Context(), cookie.Value); err != nil {
			return writeError(c, http.StatusInternalServerError, err)
		}
	}
	auth.ClearSessionCookie(c.Response())
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) me(c echo.Context) error {
	user, ok := userFromEcho(c)
	if !ok {
		return writeError(c, http.StatusUnauthorized, errors.New("authentication required"))
	}
	return c.JSON(http.StatusOK, map[string]any{"user": user})
}

func (s *Server) summary(c echo.Context) error {
	summary, err := monitoring.BuildSummary(c.Request().Context(), s.store, s.monitor.SpeedRunning(), time.Now())
	return respond(c, summary, err)
}

func (s *Server) speedTests(c echo.Context) error {
	limit := intParam(c, "limit", 50, 200)
	offset := intParam(c, "offset", 0, 100000)
	from, err := timeParam(c, "from")
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	to, err := timeParam(c, "to")
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	items, err := s.store.SpeedTests(c.Request().Context(), limit, offset, from, to)
	return respond(c, map[string]any{"items": items}, err)
}

func (s *Server) runSpeedTest(c echo.Context) error {
	if !s.monitor.TriggerSpeedTest(c.Request().Context()) {
		return c.JSON(http.StatusConflict, map[string]string{"error": "speed test already running"})
	}
	return c.JSON(http.StatusAccepted, map[string]string{"status": "started"})
}

func (s *Server) latestSpeedTest(c echo.Context) error {
	item, err := s.store.LatestSpeedTest(c.Request().Context())
	return respond(c, item, err)
}

func (s *Server) latency(c echo.Context) error {
	limit := intParam(c, "limit", 200, 1000)
	from, err := timeParam(c, "from")
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	to, err := timeParam(c, "to")
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	items, err := s.store.Latency(c.Request().Context(), c.QueryParam("target"), limit, from, to)
	return respond(c, map[string]any{"items": items}, err)
}

func (s *Server) latencySummary(c echo.Context) error {
	from, err := timeParam(c, "from")
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	to, err := timeParam(c, "to")
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	summary, err := s.store.LatencySummary(c.Request().Context(), from, to)
	return respond(c, summary, err)
}

func (s *Server) dnsChecks(c echo.Context) error {
	items, err := s.store.DNSChecks(c.Request().Context(), intParam(c, "limit", 100, 500))
	return respond(c, map[string]any{"items": items}, err)
}

func (s *Server) latestDNS(c echo.Context) error {
	item, err := s.store.LatestDNS(c.Request().Context())
	return respond(c, item, err)
}

func (s *Server) outages(c echo.Context) error {
	items, err := s.store.Outages(c.Request().Context(), intParam(c, "limit", 100, 500))
	return respond(c, map[string]any{"items": items}, err)
}

func (s *Server) activeOutage(c echo.Context) error {
	item, err := s.store.ActiveOutage(c.Request().Context())
	return respond(c, item, err)
}

func (s *Server) settings(c echo.Context) error {
	settings, err := s.store.Settings(c.Request().Context())
	return respond(c, settings, err)
}

func (s *Server) saveSettings(c echo.Context) error {
	var settings domain.Settings
	if err := c.Bind(&settings); err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	normalizeSettings(&settings)
	err := s.store.SaveSettings(c.Request().Context(), settings)
	return respond(c, settings, err)
}

func (s *Server) users(c echo.Context) error {
	users, err := s.store.Users(c.Request().Context())
	return respond(c, map[string]any{"items": users}, err)
}

func (s *Server) createUser(c echo.Context) error {
	var req userRequest
	if err := c.Bind(&req); err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	req.normalize()
	if req.Username == "" {
		return writeError(c, http.StatusBadRequest, errors.New("username is required"))
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	id, err := s.store.CreateUser(c.Request().Context(), req.Username, hash, req.Role, time.Now().UTC())
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	user, err := s.store.UserByID(c.Request().Context(), id)
	return respond(c, user, err)
}

func (s *Server) updateUser(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	var req userRequest
	if err := c.Bind(&req); err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	req.normalize()
	if req.Username == "" {
		return writeError(c, http.StatusBadRequest, errors.New("username is required"))
	}
	var hash *string
	if req.Password != "" {
		value, err := auth.HashPassword(req.Password)
		if err != nil {
			return writeError(c, http.StatusBadRequest, err)
		}
		hash = &value
	}
	if err := s.store.UpdateUser(c.Request().Context(), id, req.Username, req.Role, hash, time.Now().UTC()); err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	user, err := s.store.UserByID(c.Request().Context(), id)
	return respond(c, user, err)
}

func (s *Server) deleteUser(c echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	current, ok := userFromEcho(c)
	if ok && current.ID == id {
		return writeError(c, http.StatusBadRequest, errors.New("cannot delete the current user"))
	}
	count, err := s.store.UserCount(c.Request().Context())
	if err != nil {
		return writeError(c, http.StatusInternalServerError, err)
	}
	if count <= 1 {
		return writeError(c, http.StatusBadRequest, errors.New("cannot delete the last user"))
	}
	if err := s.store.DeleteUser(c.Request().Context(), id); err != nil {
		return writeError(c, http.StatusBadRequest, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) static(c echo.Context) error {
	path := c.Request().URL.Path
	if strings.HasPrefix(path, "/api/") {
		return writeError(c, http.StatusNotFound, errors.New("not found"))
	}
	clean := strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
	filePath := filepath.Join(s.staticDir, clean)
	if path == "/" {
		filePath = filepath.Join(s.staticDir, "index.html")
	}
	if _, err := os.Stat(filePath); err != nil {
		filePath = filepath.Join(s.staticDir, "index.html")
	}
	return c.File(filePath)
}

func (s *Server) authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, err := s.auth.UserForRequest(c.Request().Context(), c.Request())
		if err != nil {
			return writeError(c, http.StatusInternalServerError, err)
		}
		if user == nil {
			return writeError(c, http.StatusUnauthorized, errors.New("authentication required"))
		}
		c.Set("user", *user)
		req := c.Request().WithContext(auth.ContextWithUser(c.Request().Context(), *user))
		c.SetRequest(req)
		return next(c)
	}
}

type userRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (r *userRequest) normalize() {
	r.Username = strings.TrimSpace(r.Username)
	if r.Role == "" {
		r.Role = "admin"
	}
}

func normalizeSettings(s *domain.Settings) {
	if s.SpeedTestScheduleMinutes < 0 {
		s.SpeedTestScheduleMinutes = 0
	}
	if s.LatencyIntervalSeconds < 10 {
		s.LatencyIntervalSeconds = 10
	}
	if s.DNSIntervalSeconds < 10 {
		s.DNSIntervalSeconds = 10
	}
	if s.OutageFailureThreshold < 1 {
		s.OutageFailureThreshold = 1
	}
}

func respond(c echo.Context, value any, err error) error {
	if err != nil {
		return writeError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, value)
}

func writeError(c echo.Context, status int, err error) error {
	return c.JSON(status, map[string]string{"error": err.Error()})
}

func userFromEcho(c echo.Context) (domain.User, bool) {
	user, ok := c.Get("user").(domain.User)
	return user, ok
}

func intParam(c echo.Context, key string, fallback, max int) int {
	raw := c.QueryParam(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	if v > max {
		return max
	}
	return v
}

func timeParam(c echo.Context, key string) (*time.Time, error) {
	raw := c.QueryParam(key)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func pathID(c echo.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
