package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
