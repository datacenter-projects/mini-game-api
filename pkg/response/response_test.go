package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"app/pkg/apperr"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

func call(t *testing.T, h fiber.Handler, lang string) (int, map[string]any) {
	t.Helper()
	app := fiber.New(fiber.Config{ErrorHandler: FiberErrorHandler})
	app.Get("/", h)
	req := httptest.NewRequest("GET", "/", nil)
	if lang != "" {
		req.Header.Set("X-Lang", lang)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid json: %s", body)
	}
	return resp.StatusCode, m
}

func TestOK(t *testing.T) {
	status, m := call(t, func(c *fiber.Ctx) error { return OK(c, fiber.Map{"id": 1}) }, "en")
	if status != 200 || m["code"].(float64) != 200 || m["msg"] != "Success" {
		t.Fatalf("got %d %v", status, m)
	}
}

func TestErrorKnownCode(t *testing.T) {
	status, m := call(t, func(c *fiber.Ctx) error { return Error(c, apperr.ErrForbidden) }, "")
	if status != 200 || m["code"].(float64) != 403 || m["msg"] != "ไม่มีสิทธิ์เข้าถึง" {
		t.Fatalf("got %d %v", status, m)
	}
	if _, ok := m["data"]; !ok {
		t.Fatal("ต้องมี key data เสมอ")
	}
}

func TestErrorUnknownHidesDetail(t *testing.T) {
	status, m := call(t, func(c *fiber.Ctx) error { return Error(c, errors.New("pq: secret column")) }, "en")
	if status != 500 || m["code"].(float64) != 500 || m["msg"] != "Internal server error" {
		t.Fatalf("got %d %v", status, m)
	}
}

func TestPageData(t *testing.T) {
	p := utils.Page{Page: 2, Limit: 10}
	got := NewPageData([]int{1, 2, 3}, p, 25)
	if got.TotalPage != 3 || !got.HasNext || !got.HasPrev || got.TotalCount != 25 {
		t.Fatalf("got %+v", got)
	}
	empty := NewPageData[int](nil, utils.Page{Page: 1, Limit: 10}, 0)
	if empty.Data == nil || empty.TotalPage != 0 || empty.HasNext || empty.HasPrev {
		t.Fatalf("empty got %+v", empty)
	}
}
