package utils

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"app/pkg/apperr"

	"github.com/gofiber/fiber/v2"
)

func formApp(limit int64, read *[]byte, readErr *error) *fiber.App {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		data, err := ReadFormFile(c, "file", limit)
		*read, *readErr = data, err
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func postFormFile(t *testing.T, app *fiber.App, field string, content []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, "x.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(content)
	_ = mw.WriteField("other", "text")
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if _, err := app.Test(req, -1); err != nil {
		t.Fatal(err)
	}
}

func TestReadFormFile(t *testing.T) {
	var data []byte
	var err error
	app := formApp(10, &data, &err)

	postFormFile(t, app, "file", []byte("hello"))
	if err != nil || string(data) != "hello" {
		t.Errorf("ไฟล์เล็ก: got (%q, %v)", data, err)
	}

	// อ่านไม่เกิน limit — ไฟล์ที่ใหญ่กว่าถูกตัดที่ limit (ผู้เรียกส่ง เพดาน+1 แล้วเทียบ len เอง)
	postFormFile(t, app, "file", []byte(strings.Repeat("a", 100)))
	if err != nil || len(data) != 10 {
		t.Errorf("ไฟล์ใหญ่: ต้องอ่านแค่ limit ไบต์ got (%d, %v)", len(data), err)
	}

	postFormFile(t, app, "file", nil) // ไฟล์ว่าง = อ่านได้ 0 ไบต์ (service ตัดสินเอง)
	if err != nil || len(data) != 0 {
		t.Errorf("ไฟล์ว่าง: got (%d, %v)", len(data), err)
	}

	// ไม่แนบไฟล์ใน field ที่ขอ = 422 บอก field
	postFormFile(t, app, "other_name", []byte("x"))
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != 422 || !strings.Contains(ae.MsgEN, "file") {
		t.Errorf("ไม่แนบไฟล์: want 422 ที่บอก field got %v", err)
	}
}
