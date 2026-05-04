package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"brights/api/internal/domain"
)

type xiaomiQRLoginSession struct {
	SessionID      string
	Server         string
	QRImageURL     string
	LoginURL       string
	LongPollingURL string
	Timeout        int64
	Client         *http.Client
	UserID         interface{}
	Ssecurity      string
	ServiceToken   string
	Location       string
	CreatedAt      time.Time
}

var learnerXiaomiQRSessions = struct {
	sync.Mutex
	items map[string]*xiaomiQRLoginSession
}{
	items: make(map[string]*xiaomiQRLoginSession),
}

func (s *Server) handleLearnerGetXiaomiConfig(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	item, err := s.service.GetLearnerXiaomiConfig(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleLearnerSaveXiaomiConfig(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.SaveXiaomiConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := s.service.SaveLearnerXiaomiConfig(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleLearnerClearXiaomiTokens(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	if err := s.service.ClearLearnerXiaomiTokens(c.Request.Context(), claims.UserID); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleLearnerXiaomiQRLogin(c *gin.Context) {
	var input struct {
		Server string `json:"server"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, domainError("invalid xiaomi qr login request"))
		return
	}

	loginBaseURL := xiaomiLoginBaseURL(input.Server)
	jar, err := cookiejar.New(nil)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
	}

	userAgent := generateXiaomiUserAgent()
	qrURL := fmt.Sprintf("%s/longPolling/loginUrl", loginBaseURL)
	params := url.Values{}
	params.Set("_qrsize", "480")
	params.Set("qs", "%3Fsid%3Dxiaomiio%26_json%3Dtrue")
	params.Set("callback", "https://sts.api.io.mi.com/sts")
	params.Set("_hasLogo", "false")
	params.Set("sid", "xiaomiio")
	params.Set("serviceParam", "")
	params.Set("_locale", "en_GB")
	params.Set("_dc", strconv.FormatInt(time.Now().UnixMilli(), 10))

	req, err := http.NewRequest(http.MethodGet, qrURL+"?"+params.Encode(), nil)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(body), "&&&START&&&")), &payload); err != nil {
		writeError(c, http.StatusBadGateway, domainError("failed to parse xiaomi qr response"))
		return
	}

	qrImageURL, _ := payload["qr"].(string)
	loginURL, _ := payload["loginUrl"].(string)
	longPollingURL, _ := payload["lp"].(string)
	timeout, _ := payload["timeout"].(float64)
	if strings.TrimSpace(qrImageURL) == "" || strings.TrimSpace(longPollingURL) == "" {
		writeError(c, http.StatusBadGateway, domainError("xiaomi qr response is incomplete"))
		return
	}

	imageResp, err := client.Get(qrImageURL)
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	defer imageResp.Body.Close()
	if imageResp.StatusCode != http.StatusOK {
		writeError(c, http.StatusBadGateway, domainError(fmt.Sprintf("xiaomi qr image request failed with status %d", imageResp.StatusCode)))
		return
	}
	imageBytes, err := io.ReadAll(imageResp.Body)
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}

	session := &xiaomiQRLoginSession{
		SessionID:      generateRandomString(32),
		Server:         normalizeXiaomiServerForHTTP(input.Server),
		QRImageURL:     qrImageURL,
		LoginURL:       loginURL,
		LongPollingURL: longPollingURL,
		Timeout:        int64(timeout),
		Client:         client,
		CreatedAt:      time.Now(),
	}
	storeLearnerXiaomiQRSession(session)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"session_id": session.SessionID,
		"qr_image":   "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes),
		"login_url":  loginURL,
		"timeout":    session.Timeout,
		"server":     session.Server,
		"message":    "scan the qr code in mi home or xiaomi account app",
	})
}

func (s *Server) handleLearnerXiaomiQRCheck(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}

	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		writeError(c, http.StatusBadRequest, domainError("session_id is required"))
		return
	}

	session, exists := loadLearnerXiaomiQRSession(sessionID)
	if !exists {
		writeError(c, http.StatusNotFound, domainError("xiaomi qr session does not exist or has expired"))
		return
	}

	if session.Timeout > 0 && time.Since(session.CreatedAt).Seconds() > float64(session.Timeout) {
		deleteLearnerXiaomiQRSession(sessionID)
		c.JSON(http.StatusRequestTimeout, gin.H{
			"status":  "timeout",
			"message": "qr code expired, please request a new one",
		})
		return
	}

	req, err := http.NewRequest(http.MethodGet, session.LongPollingURL, nil)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	userAgent := generateXiaomiUserAgent()
	req.Header.Set("User-Agent", userAgent)

	checkClient := &http.Client{
		Timeout: 5 * time.Second,
		Jar:     session.Client.Jar,
	}
	resp, err := checkClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			c.JSON(http.StatusOK, gin.H{"status": "waiting", "message": "waiting for scan"})
			return
		}
		writeError(c, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{"status": "waiting", "message": "waiting for scan"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(body), "&&&START&&&")), &payload); err != nil {
		writeError(c, http.StatusBadGateway, domainError("failed to parse xiaomi qr check response"))
		return
	}

	if code, ok := payload["code"].(float64); ok && code != 0 {
		c.JSON(http.StatusOK, gin.H{"status": "waiting", "message": "waiting for scan"})
		return
	}

	if userID, ok := payload["userId"]; ok {
		session.UserID = userID
	}
	session.Ssecurity, _ = payload["ssecurity"].(string)
	session.Location, _ = payload["location"].(string)

	if strings.TrimSpace(session.Location) != "" {
		finalReq, err := http.NewRequest(http.MethodGet, session.Location, nil)
		if err == nil {
			finalReq.Header.Set("User-Agent", userAgent)
			finalReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if finalResp, finalErr := session.Client.Do(finalReq); finalErr == nil {
				for _, cookie := range finalResp.Cookies() {
					if cookie.Name == "serviceToken" {
						session.ServiceToken = cookie.Value
						break
					}
				}
				finalResp.Body.Close()
			}
		}
	}

	xiaomiUserID := xiaomiUserIDString(session.UserID)
	currentConfig, _ := s.service.GetLearnerXiaomiConfig(c.Request.Context(), claims.UserID)
	savedConfig, saveErr := s.service.SaveLearnerXiaomiConfig(c.Request.Context(), claims.UserID, domain.SaveXiaomiConfigInput{
		Username:     currentConfig.Username,
		XiaomiUserID: xiaomiUserID,
		Server:       session.Server,
		Ssecurity:    session.Ssecurity,
		ServiceToken: session.ServiceToken,
		IsActive:     true,
	})
	if saveErr != nil {
		writeError(c, http.StatusBadRequest, saveErr)
		return
	}

	deviceCount := savedConfig.DeviceCount
	devicesSynced := false
	deviceSyncError := ""
	if savedConfig.HasCredentials && savedConfig.DeviceCount == 0 {
		devices, refreshErr := s.service.RefreshLearnerXiaomiDevices(c.Request.Context(), claims.UserID)
		if refreshErr != nil {
			deviceSyncError = refreshErr.Error()
		} else {
			devicesSynced = true
			deviceCount = len(devices)
			if refreshedConfig, err := s.service.GetLearnerXiaomiConfig(c.Request.Context(), claims.UserID); err == nil {
				savedConfig = refreshedConfig
				deviceCount = savedConfig.DeviceCount
			}
		}
	}

	go func(id string) {
		time.Sleep(10 * time.Second)
		deleteLearnerXiaomiQRSession(id)
	}(sessionID)

	message := "xiaomi login succeeded"
	if devicesSynced {
		message = fmt.Sprintf("xiaomi login succeeded, synced %d devices", deviceCount)
	} else if deviceSyncError != "" {
		message = "xiaomi login succeeded, device sync pending"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"status":         "success",
		"message":        message,
		"user_id":        session.UserID,
		"xiaomi_user_id": xiaomiUserID,
		"ssecurity":      session.Ssecurity,
		"service_token":  session.ServiceToken,
		"device_count":   deviceCount,
		"devices_synced": devicesSynced,
		"device_sync_error": func() string {
			return deviceSyncError
		}(),
		"config": savedConfig,
	})
}

func (s *Server) handleLearnerXiaomiHomes(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	items, err := s.service.ListLearnerXiaomiHomes(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleLearnerXiaomiDevices(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	item, err := s.service.ListLearnerXiaomiDevices(c.Request.Context(), claims.UserID, c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true"))
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) handleLearnerRefreshXiaomiDevices(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	items, err := s.service.RefreshLearnerXiaomiDevices(c.Request.Context(), claims.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "device_count": len(items), "devices": items})
}

func (s *Server) handleLearnerSearchXiaomiDevices(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	items, err := s.service.FindLearnerXiaomiDevices(c.Request.Context(), claims.UserID, firstNonEmpty(c.Query("q"), c.Query("query")), limit)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (s *Server) handleLearnerXiaomiDeviceStatus(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	rawProperties := strings.TrimSpace(c.Query("properties"))
	var properties []string
	if rawProperties != "" {
		for _, item := range strings.Split(rawProperties, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				properties = append(properties, trimmed)
			}
		}
	}
	data, err := s.service.GetLearnerXiaomiDeviceStatus(
		c.Request.Context(),
		claims.UserID,
		c.Param("did"),
		properties,
		c.DefaultQuery("include_metadata", "true") != "false",
	)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) handleLearnerControlXiaomiDevice(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.XiaomiControlDeviceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	data, err := s.service.ControlLearnerXiaomiDevice(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) handleLearnerXiaomiPropGet(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.XiaomiPropGetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	data, err := s.service.LearnerXiaomiPropGet(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) handleLearnerXiaomiPropSet(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.XiaomiPropSetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	data, err := s.service.LearnerXiaomiPropSet(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) handleLearnerXiaomiAction(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var input domain.XiaomiActionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	data, err := s.service.LearnerXiaomiAction(c.Request.Context(), claims.UserID, input)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) handleLearnerXiaomiPropGetBatch(c *gin.Context) {
	claims, err := learnerClaimsFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err)
		return
	}
	var payload struct {
		Items []domain.XiaomiBatchPropItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	data, err := s.service.LearnerXiaomiPropGetBatch(c.Request.Context(), claims.UserID, payload.Items)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) handleLearnerXiaomiMiotSpec(c *gin.Context) {
	model := strings.TrimSpace(c.Query("model"))
	if model == "" {
		writeError(c, http.StatusBadRequest, domainError("model is required"))
		return
	}
	spec, raw, err := s.service.GetLearnerMiotSpec(c.Request.Context(), model)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	var parsed interface{}
	_ = json.Unmarshal(raw, &parsed)
	c.JSON(http.StatusOK, gin.H{"spec": parsed, "summary": spec})
}

func normalizeXiaomiServerForHTTP(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cn", "de", "sg", "us", "ru", "tw":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "cn"
	}
}

func xiaomiLoginBaseURL(server string) string {
	switch normalizeXiaomiServerForHTTP(server) {
	case "de":
		return "https://account.de.xiaomi.com"
	case "sg":
		return "https://account.sg.xiaomi.com"
	case "us":
		return "https://account.us.xiaomi.com"
	case "ru":
		return "https://account.ru.xiaomi.com"
	case "tw":
		return "https://account.tw.xiaomi.com"
	default:
		return "https://account.xiaomi.com"
	}
}

func generateXiaomiUserAgent() string {
	randomText := generateRandomString(18)
	agentID := make([]byte, 13)
	for index := range agentID {
		n, _ := rand.Int(rand.Reader, big.NewInt(5))
		agentID[index] = byte(65 + n.Int64())
	}
	return fmt.Sprintf("%s-%s APP/com.xiaomi.mihome APPV/10.5.201", randomText, string(agentID))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	var builder strings.Builder
	builder.Grow(length)
	for index := 0; index < length; index++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		builder.WriteByte(charset[n.Int64()])
	}
	return builder.String()
}

func storeLearnerXiaomiQRSession(session *xiaomiQRLoginSession) {
	learnerXiaomiQRSessions.Lock()
	defer learnerXiaomiQRSessions.Unlock()
	learnerXiaomiQRSessions.items[session.SessionID] = session
}

func loadLearnerXiaomiQRSession(id string) (*xiaomiQRLoginSession, bool) {
	learnerXiaomiQRSessions.Lock()
	defer learnerXiaomiQRSessions.Unlock()
	session, exists := learnerXiaomiQRSessions.items[id]
	return session, exists
}

func deleteLearnerXiaomiQRSession(id string) {
	learnerXiaomiQRSessions.Lock()
	defer learnerXiaomiQRSessions.Unlock()
	delete(learnerXiaomiQRSessions.items, id)
}

func xiaomiUserIDString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}
