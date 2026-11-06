package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mohammadirham37/gopanglima/framework"
	"github.com/mohammadirham37/gopanglima/framework/foundation"
	"github.com/mohammadirham37/gopanglima/framework/logging"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	app, err := newApp(
		framework.WithEnv(foundation.NewEnv(map[string]string{"API_TOKEN": "secret"}, nil)),
		framework.WithLogger(logging.Discard()),
	)
	if err != nil {
		t.Fatal(err)
	}
	h, err := app.Handler()
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func call(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUserCRUD(t *testing.T) {
	h := newTestHandler(t)

	if rec := call(h, "POST", "/api/v1/users", `{"name":"Ana","email":"ana@example.com"}`, ""); rec.Code != 401 {
		t.Fatalf("unauthenticated create = %d", rec.Code)
	}
	if rec := call(h, "POST", "/api/v1/users", `{"name":"Ana","email":"ana@example.com"}`, "wrong"); rec.Code != 401 {
		t.Fatalf("wrong token create = %d", rec.Code)
	}
	if rec := call(h, "POST", "/api/v1/users", `{"name":"A","email":"bad"}`, "secret"); rec.Code != 422 {
		t.Fatalf("invalid create = %d %s", rec.Code, rec.Body)
	}

	rec := call(h, "POST", "/api/v1/users", `{"name":"Ana","email":"ana@example.com"}`, "secret")
	if rec.Code != 201 {
		t.Fatalf("create = %d %s", rec.Code, rec.Body)
	}
	var created struct {
		Data struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.Data.ID != 1 || created.Data.Name != "Ana" {
		t.Fatalf("created = %+v (%v)", created, err)
	}

	if rec := call(h, "GET", "/api/v1/users/1", "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "ana@example.com") {
		t.Fatalf("show = %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", "/api/v1/users", "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"id":1`) {
		t.Fatalf("index = %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", "/api/v1/users/abc", "", ""); rec.Code != 404 {
		t.Fatalf("show invalid id = %d", rec.Code)
	}
	if rec := call(h, "DELETE", "/api/v1/users/1", "", "secret"); rec.Code != 204 {
		t.Fatalf("delete = %d", rec.Code)
	}
	if rec := call(h, "GET", "/api/v1/users/1", "", ""); rec.Code != 404 {
		t.Fatalf("show deleted = %d", rec.Code)
	}
	if rec := call(h, "PATCH", "/api/v1/users/1", "", ""); rec.Code != 405 {
		t.Fatalf("patch = %d", rec.Code)
	}
}
