package router

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"homelink-monitor/services/api/internal/domain"
)

type Settings struct {
	URL      string
	Username string
	Password string
}

type Snapshot struct {
	Capability domain.RouterTrafficCapability
	Sample     domain.RouterTrafficSample
	Clients    []domain.RouterTrafficClient
}

type Provider struct {
	client         *http.Client
	mu             sync.Mutex
	cachedSession  *session
	cachedKey      string
	sessionExpires time.Time
}

func NewProvider(client *http.Client) *Provider {
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				// TP-Link local management commonly uses a self-signed HTTPS
				// certificate. This client is used only for the configured
				// LAN router URL, never for arbitrary internet requests.
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
	}
	return &Provider{client: client}
}

func (p *Provider) ProbeAndCollect(ctx context.Context, settings Settings) Snapshot {
	if strings.TrimSpace(settings.URL) == "" || settings.Password == "" {
		now := time.Now().UTC()
		capability := domain.RouterTrafficCapability{Provider: "tplink-web", CheckedAt: now}
		sample := domain.RouterTrafficSample{CheckedAt: now, Provider: capability.Provider}
		err := "router traffic URL and password are required"
		capability.Error = err
		sample.Error = err
		return Snapshot{Capability: capability, Sample: sample}
	}

	baseURL, err := normalizeBaseURL(settings.URL)
	if err != nil {
		now := time.Now().UTC()
		capability := domain.RouterTrafficCapability{Provider: "tplink-web", CheckedAt: now}
		sample := domain.RouterTrafficSample{CheckedAt: now, Provider: capability.Provider}
		capability.Error = err.Error()
		sample.Error = err.Error()
		return Snapshot{Capability: capability, Sample: sample}
	}

	snapshot := p.probeBaseURL(ctx, settings, baseURL)
	if httpsURL, ok := httpsFallbackURL(baseURL); ok && shouldRetryHTTPS(snapshot) {
		httpsSnapshot := p.probeBaseURL(ctx, settings, httpsURL)
		if httpsSnapshot.Capability.Authenticated || httpsSnapshot.Sample.Success {
			return httpsSnapshot
		}
	}
	return snapshot
}

func (p *Provider) probeBaseURL(ctx context.Context, settings Settings, baseURL string) Snapshot {
	now := time.Now().UTC()
	capability := domain.RouterTrafficCapability{
		Provider:  "tplink-web",
		CheckedAt: now,
	}
	sample := domain.RouterTrafficSample{
		CheckedAt: now,
		Provider:  capability.Provider,
	}
	session, err := p.session(ctx, settings, baseURL)
	if err != nil {
		capability.Reachable = true
		capability.Error = err.Error()
		sample.Error = err.Error()
		return Snapshot{Capability: capability, Sample: sample}
	}

	capability.Reachable = true
	capability.Authenticated = true
	responses := collectEndpointResponses(ctx, session)
	if len(responses) == 0 {
		p.invalidateSession(session)
		if fresh, err := p.session(ctx, settings, baseURL); err == nil {
			session = fresh
			responses = collectEndpointResponses(ctx, session)
		}
	}
	clients := ExtractClients(responses)
	capability.Sources = sourceNames(responses)
	capability.ClientListAvailable = len(clients) > 0
	capability.DownloadAvailable = anyClientHas(clients, func(c domain.RouterTrafficClient) bool { return c.DownloadBps != nil })
	capability.UploadAvailable = anyClientHas(clients, func(c domain.RouterTrafficClient) bool { return c.UploadBps != nil })
	capability.TotalTrafficAvailable = anyClientHas(clients, func(c domain.RouterTrafficClient) bool { return c.TotalBps != nil || c.TotalBytes != nil })
	sample.ClientCount = len(clients)
	sample.DownloadAvailable = capability.DownloadAvailable
	sample.UploadAvailable = capability.UploadAvailable
	sample.TotalTrafficAvailable = capability.TotalTrafficAvailable
	sample.DownloadBps = sumClients(clients, func(c domain.RouterTrafficClient) *float64 { return c.DownloadBps })
	sample.UploadBps = sumClients(clients, func(c domain.RouterTrafficClient) *float64 { return c.UploadBps })
	sample.TotalBps = sumClients(clients, func(c domain.RouterTrafficClient) *float64 { return c.TotalBps })
	if sample.TotalBps == nil && sample.DownloadBps != nil && sample.UploadBps != nil {
		total := *sample.DownloadBps + *sample.UploadBps
		sample.TotalBps = &total
	}
	sample.Success = capability.ClientListAvailable
	if !sample.Success {
		msg := "router responded, but no supported client traffic fields were detected"
		capability.Error = msg
		sample.Error = msg
	}
	return Snapshot{Capability: capability, Sample: sample, Clients: clients}
}

func (p *Provider) session(ctx context.Context, settings Settings, baseURL string) (*session, error) {
	key := sessionKey(settings, baseURL)
	now := time.Now()

	p.mu.Lock()
	cached := p.cachedSession
	if cached != nil && p.cachedKey == key && now.Before(p.sessionExpires) {
		p.mu.Unlock()
		return cached, nil
	}
	p.mu.Unlock()

	next := &session{provider: p, baseURL: baseURL, username: settings.Username, password: settings.Password}
	if err := next.login(ctx); err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.cachedSession = next
	p.cachedKey = key
	p.sessionExpires = now.Add(2 * time.Minute)
	p.mu.Unlock()
	return next, nil
}

func (p *Provider) invalidateSession(s *session) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cachedSession == s {
		p.cachedSession = nil
		p.cachedKey = ""
		p.sessionExpires = time.Time{}
	}
}

func sessionKey(settings Settings, baseURL string) string {
	sum := sha256.Sum256([]byte(settings.Password))
	return baseURL + "|" + strings.TrimSpace(settings.Username) + "|" + hex.EncodeToString(sum[:])
}

func collectEndpointResponses(ctx context.Context, session *session) []endpointResponse {
	responses := []endpointResponse{}
	for _, endpoint := range probeEndpoints {
		data, err := session.request(ctx, endpoint.path, endpoint.operation)
		if err != nil {
			continue
		}
		responses = append(responses, endpointResponse{source: endpoint.name, data: data})
	}
	return responses
}

func httpsFallbackURL(baseURL string) (string, bool) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "http" {
		return "", false
	}
	u.Scheme = "https"
	return u.String(), true
}

func shouldRetryHTTPS(snapshot Snapshot) bool {
	return snapshot.Capability.Authenticated && !snapshot.Sample.Success
}

type endpoint struct {
	name      string
	path      string
	operation string
}

var probeEndpoints = []endpoint{
	{name: "qos/game_accelerator_speeds", path: "admin/smart_network?form=game_accelerator", operation: "loadSpeed"},
}

type endpointResponse struct {
	source string
	data   any
}

type session struct {
	provider    *Provider
	baseURL     string
	username    string
	password    string
	stok        string
	cookie      string
	authMode    string
	aesKey      string
	aesIV       string
	hash        string
	seq         int
	rsaN        string
	rsaE        string
	replaceHash bool
	sysauth     string
}

func (s *session) login(ctx context.Context) error {
	var attempts []string
	if err := s.loginBE550(ctx); err == nil {
		return nil
	} else {
		attempts = append(attempts, "BE550/simple RSA login failed: "+err.Error())
	}
	if err := s.loginSG(ctx); err == nil {
		return nil
	} else {
		attempts = append(attempts, "sg encrypted login failed: "+err.Error())
	}
	payload := map[string]any{"operation": "login", "password": s.password}
	if s.username != "" {
		payload["username"] = s.username
	}
	data, err := s.post(ctx, "admin/login", payload)
	if err != nil {
		legacyPassword := base64.StdEncoding.EncodeToString([]byte(s.password))
		data, err = s.post(ctx, "cgi-bin/luci/;stok=/login?form=login", map[string]any{"operation": "login", "password": legacyPassword})
		if err != nil {
			attempts = append(attempts, "legacy login failed: "+err.Error())
			return fmt.Errorf("router login failed: %s", strings.Join(attempts, "; "))
		}
	}
	s.stok = findString(data, "stok", "token")
	if s.stok == "" {
		s.stok = findString(data, "data.stok")
	}
	if s.stok == "" {
		attempts = append(attempts, "legacy login failed: router login response missing stok")
		return fmt.Errorf("router login failed: %s", strings.Join(attempts, "; "))
	}
	return nil
}

func (s *session) logout(ctx context.Context) {
	_, _ = s.request(ctx, "admin/system?form=logout", "write")
}

func (s *session) request(ctx context.Context, path, operation string) (any, error) {
	if s.authMode == "sg" {
		return s.requestSG(ctx, path, operation)
	}
	if s.authMode == "be550" {
		return s.requestPlainForm(ctx, path, operation)
	}
	fullPath := path
	if operation != "" {
		separator := "?"
		if strings.Contains(fullPath, "?") {
			separator = "&"
		}
		fullPath += separator + "operation=" + url.QueryEscape(operation)
	}
	payload := map[string]any{"operation": operation}
	return s.post(ctx, fullPath, payload)
}

func (s *session) loginBE550(ctx context.Context) error {
	pwdData, err := s.postFormPlain(ctx, "cgi-bin/luci/;stok=/login?form=keys", "operation=read", nil)
	if err != nil {
		return err
	}
	pwdKeys := findStringList(pwdData, "password")
	if len(pwdKeys) < 2 {
		return errors.New("router password RSA keys unavailable")
	}
	encryptedPassword, err := rsaEncryptPKCS1v15(s.password, pwdKeys[0], pwdKeys[1])
	if err != nil {
		return err
	}
	response, err := s.postFormPlain(ctx, "cgi-bin/luci/;stok=/login?form=login", "operation=login&password="+encryptedPassword, nil)
	if err != nil {
		return err
	}
	if success := findBool(response, "success"); !success {
		if code := findString(response, "errorcode", "error_code"); code != "" {
			return fmt.Errorf("router returned error code %s", code)
		}
		return errors.New("router returned unsuccessful login response")
	}
	s.stok = findString(response, "stok", "data.stok")
	if s.stok == "" {
		return errors.New("router login response missing stok")
	}
	s.authMode = "be550"
	return nil
}

func (s *session) loginSG(ctx context.Context) error {
	s.detectCertification(ctx)
	pwdData, err := s.postFormPlain(ctx, "cgi-bin/luci/;stok=/login?form=keys", "operation=read", nil)
	pwdKeys := []string{}
	if err == nil {
		pwdKeys = findStringList(pwdData, "password")
	}

	usernames := []string{"admin"}
	if username := strings.TrimSpace(s.username); username != "" && username != "admin" {
		usernames = append(usernames, username)
	}
	usernames = append(usernames, "")

	var attempts []string
	for _, username := range usernames {
		if len(pwdKeys) >= 2 {
			encryptedPassword, err := rsaEncryptPKCS1v15(s.password, pwdKeys[0], pwdKeys[1])
			if err != nil {
				attempts = append(attempts, "RSA password encryption failed: "+err.Error())
			} else if err := s.prepareSGLogin(ctx, username); err != nil {
				attempts = append(attempts, "RSA password login setup failed: "+err.Error())
			} else if err := s.finishSGLogin(ctx, encryptedPassword); err == nil {
				return nil
			} else {
				attempts = append(attempts, "RSA password login failed: "+err.Error())
			}
		}
		if err := s.prepareSGLogin(ctx, username); err != nil {
			attempts = append(attempts, "plain password login setup failed: "+err.Error())
		} else if err := s.finishSGLogin(ctx, s.password); err == nil {
			return nil
		} else {
			attempts = append(attempts, "plain password login failed: "+err.Error())
		}
	}
	return errors.New(strings.Join(attempts, "; "))
}

func (s *session) prepareSGLogin(ctx context.Context, username string) error {
	authData, err := s.postFormPlain(ctx, "cgi-bin/luci/;stok=/login?form=auth", "operation=read", nil)
	if err != nil {
		return err
	}
	authKeys := findStringList(authData, "key")
	if len(authKeys) < 2 {
		return errors.New("router auth RSA keys unavailable")
	}
	seqValue := findFloat(authData, "seq")
	if seqValue == nil {
		return errors.New("router auth sequence unavailable")
	}
	s.seq = int(*seqValue)
	s.rsaN = authKeys[0]
	s.rsaE = authKeys[1]
	sum := sha256.Sum256([]byte(username + s.password))
	s.hash = hex.EncodeToString(sum[:])
	s.aesKey = randomDigits(16)
	s.aesIV = randomDigits(16)
	return nil
}

func (s *session) detectCertification(ctx context.Context) {
	s.replaceHash = true
	data, err := s.postFormPlain(ctx, "cgi-bin/luci/;stok=/device_config?form=config", "operation=read", nil)
	if err != nil {
		return
	}
	certifications := findStringList(data, "certification")
	if len(certifications) == 0 {
		return
	}
	s.replaceHash = hasString(certifications, "SG CLS L1 STAGE2")
}

func (s *session) finishSGLogin(ctx context.Context, passwordValue string) error {
	encryptedData, err := aesCBCEncryptBase64(s.aesKey, s.aesIV, "operation=login&password="+passwordValue+"&confirm=true")
	if err != nil {
		return err
	}
	sign, err := s.sgLoginSignature(len(encryptedData))
	if err != nil {
		return err
	}
	response, err := s.postFormPlain(ctx, "cgi-bin/luci/;stok=/login?form=login", signedForm(sign, encryptedData), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return err
	}
	encryptedResponse := findString(response, "data")
	if encryptedResponse == "" {
		if success := findBool(response, "success"); success {
			if stok := findString(response, "stok", "data.stok"); stok != "" {
				s.stok = stok
				s.authMode = "sg"
				return nil
			}
		}
		if code := findString(response, "errorcode", "error_code"); code != "" {
			return fmt.Errorf("router login response missing encrypted data; errorcode=%s", code)
		}
		return errors.New("router login response missing encrypted data")
	}
	decrypted, err := aesCBCDecryptBase64(s.aesKey, s.aesIV, encryptedResponse)
	if err != nil {
		return err
	}
	var loginResponse any
	if err := json.Unmarshal([]byte(decrypted), &loginResponse); err != nil {
		return err
	}
	if success := findBool(loginResponse, "success"); !success {
		return fmt.Errorf("router SG login failed: %s", decrypted)
	}
	s.stok = findString(loginResponse, "stok")
	if s.stok == "" {
		s.stok = findString(loginResponse, "data.stok")
	}
	if s.stok == "" {
		return errors.New("router login response missing stok")
	}
	s.authMode = "sg"
	return nil
}

func (s *session) requestPlainForm(ctx context.Context, path, operation string) (any, error) {
	headers := map[string]string{"Origin": s.baseURL}
	body := "operation=" + url.QueryEscape(operation)
	data, err := s.postFormPlain(ctx, joinURL("", path, s.stok), body, headers)
	if err != nil {
		return nil, err
	}
	if success := findBool(data, "success"); !success {
		if code := findString(data, "errorcode", "error_code"); code != "" {
			return nil, fmt.Errorf("router returned error code %s", code)
		}
	}
	if value := findPath(data, []string{"data"}); value != nil {
		return value, nil
	}
	return data, nil
}

func (s *session) requestSG(ctx context.Context, path, operation string) (any, error) {
	payload := "operation=" + url.QueryEscape(operation)
	encryptedData, err := aesCBCEncryptBase64(s.aesKey, s.aesIV, payload)
	if err != nil {
		return nil, err
	}
	if s.replaceHash {
		sum := sha256.Sum256([]byte(encryptedData))
		s.hash = hex.EncodeToString(sum[:])
	}
	sign := s.sgRequestSignature(len(encryptedData))
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Origin":       s.baseURL,
	}
	data, err := s.postFormPlain(ctx, joinURL("", path, s.stok), signedForm(sign, encryptedData), headers)
	if err != nil {
		return nil, err
	}
	encryptedResponse := findString(data, "data")
	if encryptedResponse == "" {
		return data, nil
	}
	decrypted, err := aesCBCDecryptBase64(s.aesKey, s.aesIV, encryptedResponse)
	if err != nil {
		return nil, err
	}
	var response any
	if err := json.Unmarshal([]byte(decrypted), &response); err != nil {
		return nil, err
	}
	if success := findBool(response, "success"); !success {
		return nil, fmt.Errorf("router returned unsuccessful response")
	}
	if value := findPath(response, []string{"data"}); value != nil {
		return value, nil
	}
	return response, nil
}

func (s *session) postFormPlain(ctx context.Context, path, body string, headers map[string]string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(s.baseURL, path, ""), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", strings.TrimRight(s.baseURL, "/")+"/webpages/index.html")
	if headers == nil || headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if s.cookie != "" {
		req.Header.Set("Cookie", s.cookie)
	}
	if s.sysauth != "" {
		req.AddCookie(&http.Cookie{Name: "sysauth", Value: s.sysauth})
	}
	res, err := s.provider.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if setCookie := res.Header.Values("Set-Cookie"); len(setCookie) > 0 {
		s.cookie = strings.Join(setCookie, "; ")
		for _, raw := range setCookie {
			if value := cookieValue(raw, "sysauth"); value != "" {
				s.sysauth = value
			}
		}
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("router returned HTTP %d", res.StatusCode)
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *session) sgLoginSignature(dataLen int) (string, error) {
	signString := fmt.Sprintf("k=%s&i=%s&h=%s&s=%d", s.aesKey, s.aesIV, s.hash, s.seq+dataLen)
	var out strings.Builder
	for start := 0; start < len(signString); start += 53 {
		end := start + 53
		if end > len(signString) {
			end = len(signString)
		}
		chunk, err := rsaEncryptOAEP(signString[start:end], s.rsaN, s.rsaE)
		if err != nil {
			return "", err
		}
		out.WriteString(chunk)
	}
	return out.String(), nil
}

func (s *session) sgRequestSignature(dataLen int) string {
	signString := fmt.Sprintf("h=%s&s=%d", s.hash, s.seq+dataLen)
	key := []byte(fmt.Sprintf("k=%s&i=%s", s.aesKey, s.aesIV))
	var out strings.Builder
	for start := 0; start < len(signString); start += 53 {
		end := start + 53
		if end > len(signString) {
			end = len(signString)
		}
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte(signString[start:end]))
		out.WriteString(hex.EncodeToString(mac.Sum(nil)))
	}
	return out.String()
}

func (s *session) post(ctx context.Context, path string, payload map[string]any) (any, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(s.baseURL, path, s.stok), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cookie != "" {
		req.Header.Set("Cookie", s.cookie)
	}
	res, err := s.provider.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if setCookie := res.Header.Values("Set-Cookie"); len(setCookie) > 0 {
		s.cookie = strings.Join(setCookie, "; ")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("router returned HTTP %d", res.StatusCode)
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if code := findFloat(data, "error_code", "errorcode"); code != nil && *code != 0 {
		return nil, fmt.Errorf("router returned error code %.0f", *code)
	}
	return data, nil
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("router URL is required")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errors.New("router URL is invalid")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func joinURL(baseURL, path, stok string) string {
	path = strings.TrimLeft(path, "/")
	if stok != "" && !strings.Contains(path, "stok=") {
		path = "cgi-bin/luci/;stok=" + url.PathEscape(stok) + "/" + path
	}
	if baseURL == "" {
		return path
	}
	return strings.TrimRight(baseURL, "/") + "/" + path
}

func randomDigits(length int) string {
	var b [1]byte
	out := make([]byte, length)
	for i := range out {
		if _, err := rand.Read(b[:]); err != nil {
			out[i] = '0'
			continue
		}
		out[i] = '0' + b[0]%10
	}
	return string(out)
}

func rsaPublicKey(nHex, eHex string) (*rsa.PublicKey, int, error) {
	n := new(big.Int)
	if _, ok := n.SetString(nHex, 16); !ok {
		return nil, 0, errors.New("invalid RSA modulus")
	}
	e := new(big.Int)
	if _, ok := e.SetString(eHex, 16); !ok {
		return nil, 0, errors.New("invalid RSA exponent")
	}
	if !e.IsInt64() {
		return nil, 0, errors.New("RSA exponent is too large")
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, len(nHex), nil
}

func rsaEncryptPKCS1v15(value, nHex, eHex string) (string, error) {
	key, keyHexLen, err := rsaPublicKey(nHex, eHex)
	if err != nil {
		return "", err
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, key, []byte(value))
	if err != nil {
		return "", err
	}
	return leftPadHex(hex.EncodeToString(encrypted), keyHexLen), nil
}

func rsaEncryptOAEP(value, nHex, eHex string) (string, error) {
	key, keyHexLen, err := rsaPublicKey(nHex, eHex)
	if err != nil {
		return "", err
	}
	encrypted, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, key, []byte(value), nil)
	if err != nil {
		return "", err
	}
	return leftPadHex(hex.EncodeToString(encrypted), keyHexLen), nil
}

func leftPadHex(value string, size int) string {
	if len(value) >= size {
		return value
	}
	return strings.Repeat("0", size-len(value)) + value
}

func aesCBCEncryptBase64(key, iv, plain string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	padded := pkcs7Pad([]byte(plain), aes.BlockSize)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, []byte(iv)).CryptBlocks(encrypted, padded)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func aesCBCDecryptBase64(key, iv, encrypted string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	if len(raw)%aes.BlockSize != 0 {
		return "", errors.New("invalid AES ciphertext size")
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	decrypted := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, []byte(iv)).CryptBlocks(decrypted, raw)
	unpadded, err := pkcs7Unpad(decrypted, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	out := make([]byte, len(data)+padding)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid PKCS7 data")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for _, value := range data[len(data)-padding:] {
		if int(value) != padding {
			return nil, errors.New("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-padding], nil
}

func cookieValue(raw, name string) string {
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		prefix := name + "="
		if strings.HasPrefix(part, prefix) {
			return strings.TrimPrefix(part, prefix)
		}
	}
	return ""
}

func signedForm(sign, data string) string {
	return "sign=" + url.QueryEscape(sign) + "&data=" + url.QueryEscape(data)
}

func hasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func ExtractClients(responses []endpointResponse) []domain.RouterTrafficClient {
	byKey := map[string]domain.RouterTrafficClient{}
	for _, response := range responses {
		for _, object := range collectObjects(response.data) {
			client, ok := objectToClient(object)
			if !ok {
				continue
			}
			key := client.MAC
			if key == "" {
				key = client.IP
			}
			if key == "" {
				key = client.Hostname
			}
			if key == "" {
				continue
			}
			existing := byKey[key]
			byKey[key] = mergeClient(existing, client)
		}
	}
	out := make([]domain.RouterTrafficClient, 0, len(byKey))
	for _, client := range byKey {
		out = append(out, client)
	}
	return out
}

func objectToClient(m map[string]any) (domain.RouterTrafficClient, bool) {
	client := domain.RouterTrafficClient{
		MAC:        firstString(m, "macaddr", "mac_addr", "mac", "host_mac"),
		IP:         firstString(m, "ipaddr", "ip_addr", "ip", "host_ip"),
		Hostname:   firstString(m, "hostname", "host_name", "name", "client_name", "deviceName"),
		Connection: firstString(m, "type", "connection", "wire_type", "interface", "deviceTag", "networkMode"),
	}
	client.DownloadBps = firstRate(m, "down_speed", "download_speed", "down_rate", "download_rate", "rx_speed", "downloadSpeed", "downSpeed")
	client.UploadBps = firstRate(m, "up_speed", "upload_speed", "up_rate", "upload_rate", "tx_speed", "uploadSpeed", "upSpeed")
	client.TotalBps = firstRate(m, "speed", "rate", "total_speed", "traffic_speed", "trafficSpeed")
	client.DownloadBytes = firstNumber(m, "download", "down_bytes", "download_bytes", "bytes_received", "downloadBytes")
	client.UploadBytes = firstNumber(m, "upload", "up_bytes", "upload_bytes", "bytes_sent", "uploadBytes")
	client.TotalBytes = firstNumber(m, "traffic_usage", "traffic", "total", "total_bytes", "trafficUsage", "trafficUsed", "totalBytes")
	hasIdentity := client.MAC != "" || client.IP != "" || client.Hostname != ""
	hasTraffic := client.DownloadBps != nil || client.UploadBps != nil || client.TotalBps != nil || client.DownloadBytes != nil || client.UploadBytes != nil || client.TotalBytes != nil
	return client, hasIdentity && hasTraffic
}

func collectObjects(v any) []map[string]any {
	switch x := v.(type) {
	case map[string]any:
		out := []map[string]any{x}
		for _, child := range x {
			out = append(out, collectObjects(child)...)
		}
		return out
	case []any:
		out := []map[string]any{}
		for _, child := range x {
			out = append(out, collectObjects(child)...)
		}
		return out
	default:
		return nil
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if s, ok := value.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func firstRate(m map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if n := numberValue(value); n != nil {
				return n
			}
		}
	}
	return nil
}

func firstNumber(m map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if n := numberValue(value); n != nil {
				return n
			}
		}
	}
	return nil
}

func numberValue(v any) *float64 {
	switch x := v.(type) {
	case float64:
		return &x
	case int:
		f := float64(x)
		return &f
	case string:
		clean := strings.TrimSpace(strings.ReplaceAll(x, ",", ""))
		if clean == "" {
			return nil
		}
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			return nil
		}
		return &f
	default:
		return nil
	}
}

func mergeClient(a, b domain.RouterTrafficClient) domain.RouterTrafficClient {
	if a.MAC == "" {
		a.MAC = b.MAC
	}
	if a.IP == "" {
		a.IP = b.IP
	}
	if a.Hostname == "" {
		a.Hostname = b.Hostname
	}
	if a.Connection == "" {
		a.Connection = b.Connection
	}
	if a.DownloadBps == nil {
		a.DownloadBps = b.DownloadBps
	}
	if a.UploadBps == nil {
		a.UploadBps = b.UploadBps
	}
	if a.TotalBps == nil {
		a.TotalBps = b.TotalBps
	}
	if a.DownloadBytes == nil {
		a.DownloadBytes = b.DownloadBytes
	}
	if a.UploadBytes == nil {
		a.UploadBytes = b.UploadBytes
	}
	if a.TotalBytes == nil {
		a.TotalBytes = b.TotalBytes
	}
	return a
}

func sumClients(clients []domain.RouterTrafficClient, pick func(domain.RouterTrafficClient) *float64) *float64 {
	var total float64
	count := 0
	for _, client := range clients {
		if v := pick(client); v != nil {
			total += *v
			count++
		}
	}
	if count == 0 {
		return nil
	}
	return &total
}

func anyClientHas(clients []domain.RouterTrafficClient, pred func(domain.RouterTrafficClient) bool) bool {
	for _, client := range clients {
		if pred(client) {
			return true
		}
	}
	return false
}

func sourceNames(responses []endpointResponse) []string {
	out := make([]string, 0, len(responses))
	for _, response := range responses {
		out = append(out, response.source)
	}
	return out
}

func findString(v any, keys ...string) string {
	for _, key := range keys {
		if strings.Contains(key, ".") {
			if value := findPath(v, strings.Split(key, ".")); value != nil {
				if s, ok := value.(string); ok {
					return s
				}
			}
			continue
		}
		for _, object := range collectObjects(v) {
			if s := firstString(object, key); s != "" {
				return s
			}
		}
	}
	return ""
}

func findStringList(v any, key string) []string {
	for _, object := range collectObjects(v) {
		value, ok := object[key]
		if !ok {
			continue
		}
		items, ok := value.([]any)
		if !ok {
			continue
		}
		out := []string{}
		for _, item := range items {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func findBool(v any, key string) bool {
	for _, object := range collectObjects(v) {
		value, ok := object[key]
		if !ok {
			continue
		}
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

func findFloat(v any, keys ...string) *float64 {
	for _, key := range keys {
		for _, object := range collectObjects(v) {
			if n := firstNumber(object, key); n != nil {
				return n
			}
		}
	}
	return nil
}

func findPath(v any, parts []string) any {
	if len(parts) == 0 {
		return v
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return findPath(m[parts[0]], parts[1:])
}
