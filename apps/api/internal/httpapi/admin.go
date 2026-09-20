package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/1424772/guaguale/apps/api/internal/store"
)

const adminCookieName = "guaguale_admin"

type AdminConfig struct {
	Username      string
	PasswordHash  []byte
	SessionSecret []byte
}

func (config AdminConfig) enabled() bool {
	return config.Username != "" && len(config.PasswordHash) > 0 && len(config.SessionSecret) >= 32
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminBalanceRequest struct {
	Mode   string `json:"mode"`
	Amount int64  `json:"amount"`
}

func (api *API) adminLogin(response http.ResponseWriter, request *http.Request) {
	if !api.admin.enabled() {
		writeAPIError(response, http.StatusNotFound, "admin_unavailable", "管理员后台尚未启用")
		return
	}
	var body adminLoginRequest
	if decodeJSON(request, &body) != nil || len(body.Password) > 256 {
		writeAPIError(response, http.StatusUnauthorized, "invalid_credentials", "管理员账号或密码错误")
		return
	}
	usernameOK := subtle.ConstantTimeCompare([]byte(body.Username), []byte(api.admin.Username)) == 1
	passwordOK := bcrypt.CompareHashAndPassword(api.admin.PasswordHash, []byte(body.Password)) == nil
	if !usernameOK || !passwordOK {
		api.logger.Warn("admin login rejected", "remote_ip", clientIP(request))
		writeAPIError(response, http.StatusUnauthorized, "invalid_credentials", "管理员账号或密码错误")
		return
	}
	expiresAt := time.Now().UTC().Add(8 * time.Hour)
	http.SetCookie(response, &http.Cookie{
		Name:     adminCookieName,
		Value:    api.adminToken(expiresAt),
		Path:     "/api/v1/admin",
		HttpOnly: true,
		Secure:   api.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		Expires:  expiresAt,
		MaxAge:   int((8 * time.Hour).Seconds()),
	})
	api.logger.Info("admin login succeeded", "remote_ip", clientIP(request))
	writeJSON(response, http.StatusOK, map[string]any{"authenticated": true, "expiresAt": expiresAt})
}

func (api *API) adminLogout(response http.ResponseWriter, _ *http.Request) {
	http.SetCookie(response, &http.Cookie{
		Name:     adminCookieName,
		Value:    "",
		Path:     "/api/v1/admin",
		HttpOnly: true,
		Secure:   api.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	response.WriteHeader(http.StatusNoContent)
}

func (api *API) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if !api.admin.enabled() {
			writeAPIError(response, http.StatusNotFound, "admin_unavailable", "管理员后台尚未启用")
			return
		}
		cookie, err := request.Cookie(adminCookieName)
		if err != nil || !api.validAdminToken(cookie.Value) {
			writeAPIError(response, http.StatusUnauthorized, "admin_unauthorized", "管理员登录已失效")
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (api *API) adminUsers(response http.ResponseWriter, request *http.Request) {
	query := strings.TrimSpace(request.URL.Query().Get("query"))
	if len([]rune(query)) > 32 {
		writeAPIError(response, http.StatusBadRequest, "invalid_input", "搜索内容过长")
		return
	}
	users, err := api.store.AdminUsers(request.Context(), query, 50)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"users": users})
}

func (api *API) adminBalance(response http.ResponseWriter, request *http.Request) {
	userID, err := strconv.ParseUint(request.PathValue("id"), 10, 64)
	if err != nil || userID == 0 {
		writeAPIError(response, http.StatusBadRequest, "invalid_input", "账号编号不正确")
		return
	}
	var body adminBalanceRequest
	if decodeJSON(request, &body) != nil || (body.Mode != "add" && body.Mode != "subtract" && body.Mode != "set") || body.Amount < 0 || body.Amount > 1_000_000_000_000 {
		writeAPIError(response, http.StatusBadRequest, "invalid_input", "请选择正确操作并输入有效金币数")
		return
	}
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if len(idempotencyKey) < 16 || len(idempotencyKey) > 64 {
		writeAPIError(response, http.StatusBadRequest, "invalid_idempotency_key", "操作标识无效，请刷新后重试")
		return
	}
	user, idempotent, err := api.store.AdminAdjustBalance(request.Context(), store.AdminBalanceInput{
		UserID: userID, Mode: body.Mode, Amount: body.Amount, IdempotencyKey: "admin:" + idempotencyKey,
	})
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	api.logger.Info("admin balance adjusted",
		"remote_ip", clientIP(request), "user_id", userID, "mode", body.Mode,
		"amount", body.Amount, "balance", user.Balance, "idempotent", idempotent,
	)
	writeJSON(response, http.StatusOK, map[string]any{"user": user, "idempotent": idempotent})
}

func (api *API) adminToken(expiresAt time.Time) string {
	payload := strconv.FormatInt(expiresAt.Unix(), 10)
	mac := hmac.New(sha256.New, api.admin.SessionSecret)
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (api *API) validAdminToken(token string) bool {
	payload, signature, found := strings.Cut(token, ".")
	if !found {
		return false
	}
	expiresUnix, err := strconv.ParseInt(payload, 10, 64)
	if err != nil || time.Now().UTC().Unix() >= expiresUnix {
		return false
	}
	provided, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, api.admin.SessionSecret)
	_, _ = mac.Write([]byte(payload))
	return hmac.Equal(provided, mac.Sum(nil))
}
