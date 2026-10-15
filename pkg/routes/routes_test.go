package routes

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"app/pkg/response"

	"github.com/gofiber/fiber/v2"
)

func call(t *testing.T, a *fiber.App, method, path string) (int, []byte) {
	t.Helper()
	resp, err := a.Test(httptest.NewRequest(method, path, nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, body
}

func TestRootRoute(t *testing.T) {
	a := fiber.New(fiber.Config{ErrorHandler: response.FiberErrorHandler})
	SetupRoutes(a)

	status, body := call(t, a, "GET", "/")
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("invalid json: %s", body)
	}
	if status != fiber.StatusOK || env["code"] != float64(200) || len(env) != 2 {
		t.Fatalf("GET / = %d %s, want 200 {code,msg} only", status, body)
	}

	if status, _ := call(t, a, "HEAD", "/"); status != fiber.StatusOK {
		t.Fatalf("HEAD / = %d, want 200", status)
	}

	status, body = call(t, a, "GET", "/no-such-path")
	env = nil
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("invalid json: %s", body)
	}
	if status != fiber.StatusNotFound || env["code"] != float64(404001) {
		t.Fatalf("unknown path = %d %s, want 404 / 404001", status, body)
	}
}
