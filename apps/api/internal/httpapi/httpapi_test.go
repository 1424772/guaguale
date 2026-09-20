package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/1424772/guaguale/apps/api/internal/service"
	"github.com/1424772/guaguale/apps/api/internal/store/memory"
)

func TestRegisterRequiresAgeConfirmationAndPurchaseIsIdempotent(t *testing.T) {
	memoryStore := memory.New()
	handler := New(service.New(memoryStore), memoryStore, slog.New(slog.NewTextHandler(io.Discard, nil)), false)

	missingAge := postJSON(t, handler, "/api/v1/auth/register", map[string]any{
		"username": "接口玩家",
		"password": "correct-horse-42",
	}, nil, "")
	if missingAge.Code != http.StatusBadRequest {
		t.Fatalf("expected missing age confirmation to fail, got %d", missingAge.Code)
	}

	registered := postJSON(t, handler, "/api/v1/auth/register", map[string]any{
		"username":     "接口玩家",
		"password":     "correct-horse-42",
		"ageConfirmed": true,
	}, nil, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("expected registration success, got %d: %s", registered.Code, registered.Body.String())
	}
	cookies := registered.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly {
		t.Fatalf("expected secure session cookie, got %#v", cookies)
	}

	firstPurchase := postJSON(t, handler, "/api/v1/cards/lingqian-ticket/purchase", nil, cookies[0], "purchase-http-001")
	if firstPurchase.Code != http.StatusCreated {
		t.Fatalf("expected purchase success, got %d: %s", firstPurchase.Code, firstPurchase.Body.String())
	}
	var first struct {
		User struct {
			Balance int64 `json:"balance"`
		} `json:"user"`
		Ticket struct {
			ID      string   `json:"id"`
			Symbols []string `json:"symbols"`
		} `json:"ticket"`
		Idempotent bool `json:"idempotent"`
	}
	if err := json.Unmarshal(firstPurchase.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if first.User.Balance != 950 || len(first.Ticket.Symbols) != 0 || first.Idempotent {
		t.Fatalf("unexpected first purchase response: %#v", first)
	}

	retry := postJSON(t, handler, "/api/v1/cards/lingqian-ticket/purchase", nil, cookies[0], "purchase-http-001")
	var second struct {
		User struct {
			Balance int64 `json:"balance"`
		} `json:"user"`
		Ticket struct {
			ID string `json:"id"`
		} `json:"ticket"`
		Idempotent bool `json:"idempotent"`
	}
	if err := json.Unmarshal(retry.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	if retry.Code != http.StatusCreated || second.User.Balance != 950 || second.Ticket.ID != first.Ticket.ID || !second.Idempotent {
		t.Fatalf("purchase retry was not idempotent: %#v", second)
	}

	reveal := postJSON(t, handler, "/api/v1/tickets/"+first.Ticket.ID+"/reveal", nil, cookies[0], "")
	if reveal.Code != http.StatusOK || !bytes.Contains(reveal.Body.Bytes(), []byte(`"state":"purchased"`)) || !bytes.Contains(reveal.Body.Bytes(), []byte(`"symbols"`)) {
		t.Fatalf("ticket reveal should expose the fixed outcome without completing the scratch: %d %s", reveal.Code, reveal.Body.String())
	}
	tickets := getJSON(t, handler, "/api/v1/tickets", cookies[0])
	if tickets.Code != http.StatusOK || !bytes.Contains(tickets.Body.Bytes(), []byte(`"state":"purchased"`)) || bytes.Contains(tickets.Body.Bytes(), []byte(`"symbols"`)) {
		t.Fatalf("revealing a ticket changed its state or leaked the outcome in listings: %d %s", tickets.Code, tickets.Body.String())
	}

	leaderboard := getJSON(t, handler, "/api/v1/leaderboard", cookies[0])
	if leaderboard.Code != http.StatusOK || bytes.Contains(leaderboard.Body.Bytes(), []byte("接口玩家")) || bytes.Contains(leaderboard.Body.Bytes(), []byte("userId")) {
		t.Fatalf("leaderboard leaked private identity: %d %s", leaderboard.Code, leaderboard.Body.String())
	}
	history := getJSON(t, handler, "/api/v1/history", cookies[0])
	if history.Code != http.StatusOK || !bytes.Contains(history.Body.Bytes(), []byte("购买刮刮乐")) {
		t.Fatalf("history endpoint did not return purchase event: %d %s", history.Code, history.Body.String())
	}
}

func TestAdminCanSearchAndAdjustBalanceIdempotently(t *testing.T) {
	memoryStore := memory.New()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewWithAdmin(
		service.New(memoryStore), memoryStore, slog.New(slog.NewTextHandler(io.Discard, nil)), false,
		AdminConfig{Username: "admin", PasswordHash: passwordHash, SessionSecret: bytes.Repeat([]byte{7}, 32)},
	)

	registered := postJSON(t, handler, "/api/v1/auth/register", map[string]any{
		"username": "managed_player", "password": "correct-horse-42", "ageConfirmed": true,
	}, nil, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register player: %d %s", registered.Code, registered.Body.String())
	}

	unauthorized := getJSON(t, handler, "/api/v1/admin/users", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("admin users should require login, got %d", unauthorized.Code)
	}
	login := postJSON(t, handler, "/api/v1/admin/login", map[string]any{
		"username": "admin", "password": "admin-test-password",
	}, nil, "")
	if login.Code != http.StatusOK {
		t.Fatalf("admin login failed: %d %s", login.Code, login.Body.String())
	}
	adminCookies := login.Result().Cookies()
	if len(adminCookies) != 1 || adminCookies[0].Name != adminCookieName || !adminCookies[0].HttpOnly {
		t.Fatalf("expected HttpOnly admin cookie, got %#v", adminCookies)
	}

	users := getJSON(t, handler, "/api/v1/admin/users?query=managed", adminCookies[0])
	if users.Code != http.StatusOK || !bytes.Contains(users.Body.Bytes(), []byte("managed_player")) {
		t.Fatalf("admin search failed: %d %s", users.Code, users.Body.String())
	}
	adjust := postJSON(t, handler, "/api/v1/admin/users/1/balance", map[string]any{
		"mode": "add", "amount": 20000,
	}, adminCookies[0], "admin-test-adjust-0001")
	if adjust.Code != http.StatusOK || !bytes.Contains(adjust.Body.Bytes(), []byte(`"balance":21000`)) {
		t.Fatalf("admin adjustment failed: %d %s", adjust.Code, adjust.Body.String())
	}
	retry := postJSON(t, handler, "/api/v1/admin/users/1/balance", map[string]any{
		"mode": "add", "amount": 20000,
	}, adminCookies[0], "admin-test-adjust-0001")
	if retry.Code != http.StatusOK || !bytes.Contains(retry.Body.Bytes(), []byte(`"balance":21000`)) || !bytes.Contains(retry.Body.Bytes(), []byte(`"idempotent":true`)) {
		t.Fatalf("admin adjustment retry was not idempotent: %d %s", retry.Code, retry.Body.String())
	}
}

func getJSON(t *testing.T, handler http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func postJSON(t *testing.T, handler http.Handler, path string, body any, cookie *http.Cookie, idempotencyKey string) *httptest.ResponseRecorder {
	t.Helper()
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
