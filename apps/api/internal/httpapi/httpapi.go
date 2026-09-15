package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
	"github.com/1424772/guaguale/apps/api/internal/service"
	"github.com/1424772/guaguale/apps/api/internal/store"
)

const sessionCookieName = "guaguale_session"

type API struct {
	service      *service.Service
	store        store.Store
	logger       *slog.Logger
	cookieSecure bool
}

type contextKey string

const (
	userContextKey  contextKey = "user"
	tokenContextKey contextKey = "token"
)

type credentialsRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	AgeConfirmed bool   `json:"ageConfirmed,omitempty"`
}

type placementRequest struct {
	Location  domain.TicketLocation `json:"location"`
	DeskX     float64               `json:"deskX"`
	DeskY     float64               `json:"deskY"`
	Rotation  float64               `json:"rotation"`
	ZIndex    int                   `json:"zIndex"`
	SlotIndex *int                  `json:"slotIndex"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func New(service *service.Service, store store.Store, logger *slog.Logger, cookieSecure bool) http.Handler {
	api := &API{service: service, store: store, logger: logger, cookieSecure: cookieSecure}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /api/v1/health", api.health)
	mux.HandleFunc("GET /api/v1/ready", api.ready)
	mux.HandleFunc("POST /api/v1/auth/register", api.register)
	mux.HandleFunc("POST /api/v1/auth/login", api.login)
	mux.Handle("POST /api/v1/auth/logout", api.requireUser(http.HandlerFunc(api.logout)))
	mux.Handle("GET /api/v1/me", api.requireUser(http.HandlerFunc(api.me)))
	mux.Handle("GET /api/v1/cards", api.requireUser(http.HandlerFunc(api.cards)))
	mux.Handle("POST /api/v1/cards/{code}/purchase", api.requireUser(http.HandlerFunc(api.purchase)))
	mux.Handle("GET /api/v1/shop", api.requireUser(http.HandlerFunc(api.shop)))
	mux.Handle("POST /api/v1/shop/{code}/upgrade", api.requireUser(http.HandlerFunc(api.upgradeItem)))
	mux.Handle("GET /api/v1/tickets", api.requireUser(http.HandlerFunc(api.tickets)))
	mux.Handle("POST /api/v1/tickets/{id}/scratch", api.requireUser(http.HandlerFunc(api.scratch)))
	mux.Handle("POST /api/v1/tickets/{id}/redeem", api.requireUser(http.HandlerFunc(api.redeem)))
	mux.Handle("PATCH /api/v1/tickets/{id}/placement", api.requireUser(http.HandlerFunc(api.placeTicket)))
	mux.Handle("POST /api/v1/tickets/{id}/discard", api.requireUser(http.HandlerFunc(api.discardTicket)))
	mux.Handle("GET /api/v1/daily", api.requireUser(http.HandlerFunc(api.dailyStatus)))
	mux.Handle("POST /api/v1/daily/login-claim", api.requireUser(http.HandlerFunc(api.claimDailyLogin)))
	mux.Handle("POST /api/v1/daily/plates/start", api.requireUser(http.HandlerFunc(api.startPlate)))
	mux.Handle("POST /api/v1/daily/plates/{id}/complete", api.requireUser(http.HandlerFunc(api.completePlate)))
	mux.Handle("POST /api/v1/daily/wheel/spin", api.requireUser(http.HandlerFunc(api.spinDailyWheel)))
	return api.securityHeaders(api.limitBody(mux))
}

func (api *API) health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "guaguale-api",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (api *API) ready(response http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if err := api.store.Ping(ctx); err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"status": "ready"})
}

func (api *API) register(response http.ResponseWriter, request *http.Request) {
	var body credentialsRequest
	if err := decodeJSON(request, &body); err != nil {
		api.writeError(response, request, service.ErrInvalidInput)
		return
	}
	result, err := api.service.Register(request.Context(), body.Username, body.Password, body.AgeConfirmed)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	api.setSessionCookie(response, result.Token, result.ExpiresAt)
	writeJSON(response, http.StatusCreated, map[string]any{"user": result.User})
}

func (api *API) login(response http.ResponseWriter, request *http.Request) {
	var body credentialsRequest
	if err := decodeJSON(request, &body); err != nil {
		api.writeError(response, request, service.ErrInvalidCredentials)
		return
	}
	result, err := api.service.Login(request.Context(), body.Username, body.Password)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	api.setSessionCookie(response, result.Token, result.ExpiresAt)
	writeJSON(response, http.StatusOK, map[string]any{"user": result.User})
}

func (api *API) logout(response http.ResponseWriter, request *http.Request) {
	token, _ := request.Context().Value(tokenContextKey).(string)
	if err := api.service.Logout(request.Context(), token); err != nil {
		api.writeError(response, request, err)
		return
	}
	http.SetCookie(response, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   api.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	response.WriteHeader(http.StatusNoContent)
}

func (api *API) me(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{"user": currentUser(request)})
}

func (api *API) cards(response http.ResponseWriter, request *http.Request) {
	user := currentUser(request)
	writeJSON(response, http.StatusOK, map[string]any{"cards": api.service.Cards(user.Balance)})
}

func (api *API) purchase(response http.ResponseWriter, request *http.Request) {
	result, err := api.service.Purchase(
		request.Context(),
		currentUser(request),
		request.PathValue("code"),
		request.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusCreated, result)
}

func (api *API) shop(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{"shop": api.service.Shop(currentUser(request))})
}

func (api *API) upgradeItem(response http.ResponseWriter, request *http.Request) {
	result, err := api.service.UpgradeItem(
		request.Context(), currentUser(request), request.PathValue("code"), request.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusCreated, result)
}

func (api *API) tickets(response http.ResponseWriter, request *http.Request) {
	tickets, err := api.service.Tickets(request.Context(), currentUser(request).ID)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"tickets": tickets})
}

func (api *API) scratch(response http.ResponseWriter, request *http.Request) {
	ticket, err := api.service.Scratch(request.Context(), currentUser(request).ID, request.PathValue("id"))
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"ticket": ticket})
}

func (api *API) redeem(response http.ResponseWriter, request *http.Request) {
	result, err := api.service.Redeem(request.Context(), currentUser(request).ID, request.PathValue("id"))
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (api *API) placeTicket(response http.ResponseWriter, request *http.Request) {
	var body placementRequest
	if err := decodeJSON(request, &body); err != nil {
		api.writeError(response, request, service.ErrInvalidInput)
		return
	}
	ticket, err := api.service.PlaceTicket(request.Context(), currentUser(request).ID, request.PathValue("id"), store.TicketPlacement{
		Location: body.Location, DeskX: body.DeskX, DeskY: body.DeskY,
		Rotation: body.Rotation, ZIndex: body.ZIndex, SlotIndex: body.SlotIndex,
	})
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"ticket": ticket})
}

func (api *API) discardTicket(response http.ResponseWriter, request *http.Request) {
	ticket, err := api.service.DiscardTicket(request.Context(), currentUser(request).ID, request.PathValue("id"))
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"ticket": ticket})
}

func (api *API) dailyStatus(response http.ResponseWriter, request *http.Request) {
	status, err := api.service.DailyStatus(request.Context(), currentUser(request))
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"daily": status})
}

func (api *API) claimDailyLogin(response http.ResponseWriter, request *http.Request) {
	result, err := api.service.ClaimDailyLogin(request.Context(), currentUser(request).ID)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (api *API) startPlate(response http.ResponseWriter, request *http.Request) {
	daily, idempotent, err := api.service.StartPlate(request.Context(), currentUser(request))
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]any{"daily": daily, "idempotent": idempotent})
}

func (api *API) completePlate(response http.ResponseWriter, request *http.Request) {
	result, err := api.service.CompletePlate(request.Context(), currentUser(request).ID, request.PathValue("id"))
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (api *API) spinDailyWheel(response http.ResponseWriter, request *http.Request) {
	result, err := api.service.SpinDailyWheel(request.Context(), currentUser(request).ID)
	if err != nil {
		api.writeError(response, request, err)
		return
	}
	writeJSON(response, http.StatusCreated, result)
}

func (api *API) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeAPIError(response, http.StatusUnauthorized, "unauthorized", "请先登录")
			return
		}
		user, err := api.service.UserByToken(request.Context(), cookie.Value)
		if err != nil {
			writeAPIError(response, http.StatusUnauthorized, "unauthorized", "登录已失效，请重新登录")
			return
		}
		ctx := context.WithValue(request.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, tokenContextKey, cookie.Value)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func (api *API) setSessionCookie(response http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(response, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   api.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (api *API) writeError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeAPIError(response, http.StatusBadRequest, "invalid_input", "请检查输入内容")
	case errors.Is(err, service.ErrInvalidCredentials):
		writeAPIError(response, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
	case errors.Is(err, store.ErrUsernameTaken):
		writeAPIError(response, http.StatusConflict, "username_taken", "用户名已被使用")
	case errors.Is(err, store.ErrInsufficientFunds):
		writeAPIError(response, http.StatusConflict, "insufficient_funds", "金币不足")
	case errors.Is(err, service.ErrCardUnavailable):
		writeAPIError(response, http.StatusConflict, "card_unavailable", "该卡片暂未开放")
	case errors.Is(err, service.ErrItemUnavailable):
		writeAPIError(response, http.StatusConflict, "item_unavailable", "该道具已经满级或暂不可升级")
	case errors.Is(err, store.ErrUpgradeConflict):
		writeAPIError(response, http.StatusConflict, "upgrade_conflict", "道具等级已变化，请刷新后重试")
	case errors.Is(err, store.ErrNotFound):
		writeAPIError(response, http.StatusNotFound, "not_found", "没有找到对应内容")
	case errors.Is(err, store.ErrInvalidState):
		writeAPIError(response, http.StatusConflict, "invalid_state", "当前状态不能执行该操作")
	case errors.Is(err, store.ErrNotWinner):
		writeAPIError(response, http.StatusUnprocessableEntity, "not_winner", "该卡未中奖，无法兑奖")
	case errors.Is(err, store.ErrSlotOccupied):
		writeAPIError(response, http.StatusConflict, "slot_occupied", "这个固定卡槽已经有卡片")
	case errors.Is(err, store.ErrProtected):
		writeAPIError(response, http.StatusConflict, "ticket_protected", "固定卡槽中的卡片不能丢弃")
	case errors.Is(err, store.ErrDailyLimit):
		writeAPIError(response, http.StatusConflict, "daily_limit", "今天的次数已经用完")
	case errors.Is(err, store.ErrTooEarly):
		writeAPIError(response, http.StatusTooEarly, "too_early", "操作尚未完成，请稍候")
	case errors.Is(err, store.ErrWheelUnavailable):
		writeAPIError(response, http.StatusConflict, "wheel_unavailable", "至少持有50金币后才能转动")
	default:
		api.logger.Error("request failed", "method", request.Method, "path", request.URL.Path, "error", err)
		writeAPIError(response, http.StatusInternalServerError, "internal_error", "服务暂时不可用")
	}
}

func (api *API) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request.Body = http.MaxBytesReader(response, request.Body, 64<<10)
		next.ServeHTTP(response, request)
	})
}

func (api *API) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(response, request)
	})
}

func currentUser(request *http.Request) domain.User {
	user, _ := request.Context().Value(userContextKey).(domain.User)
	return user
}

func decodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return errors.New("request contains multiple JSON values")
	}
	return nil
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeAPIError(response http.ResponseWriter, status int, code, message string) {
	var body errorResponse
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(response, status, body)
}

func RequestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(response, request)
		logger.Info("request completed",
			"method", request.Method,
			"path", request.URL.Path,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"remote_ip", clientIP(request),
		)
	})
}

func clientIP(request *http.Request) string {
	if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return request.RemoteAddr
}
