//go:build integration

package testutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"app/pkg/configs"
	"app/platform/storage"
)

// FakeS3 คือ S3 จำลองในตัว (httptest) สำหรับ test ที่ต้องอัปไฟล์ — เก็บแค่ว่ามี object ไหนและ header อะไร
// ใช้ผ่าน S3_ENDPOINT เหมือน MinIO จึงเรียก platform/storage จริง ไม่ mock อะไร
type FakeS3 struct {
	srv *httptest.Server

	mu      sync.Mutex
	objects map[string]FakeObject
	puts    int
	deletes int
	failPut bool
}

// FakeObject — object ที่อัปเข้ามา
type FakeObject struct {
	ContentType  string
	CacheControl string
}

const fakeBucket = "test-bucket"

// NewFakeS3 เปิด S3 จำลอง ตั้ง configs.Cfg.Storage ให้ชี้มาที่มัน แล้ว storage.Init — คืนค่าทั้งหมดตอน test จบ
func NewFakeS3(t *testing.T) *FakeS3 {
	t.Helper()
	f := &FakeS3{objects: map[string]FakeObject{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))

	t.Setenv("AWS_ACCESS_KEY_ID", "test") // ให้ default credential chain ของ SDK ไม่ไปหา IMDS / ไฟล์ในเครื่อง
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	original := configs.Cfg.Storage
	configs.Cfg.Storage = configs.StorageConfig{Region: "ap-southeast-1", Bucket: fakeBucket,
		CDNBaseURL: "https://cdn.example.com", UploadTimeout: original.UploadTimeout, Endpoint: f.srv.URL}
	if err := storage.Init(context.Background(), configs.Cfg.Storage); err != nil {
		t.Fatalf("fake s3: storage init: %v", err)
	}
	t.Cleanup(func() {
		f.srv.Close()
		configs.Cfg.Storage = original
		_ = storage.Init(context.Background(), configs.StorageConfig{}) // กลับเป็น "ไม่พร้อม" (ไม่ตั้ง bucket)
	})
	return f
}

func (f *FakeS3) handle(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/"+fakeBucket+"/")
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method {
	case http.MethodPut:
		_, _ = io.Copy(io.Discard, r.Body)
		if f.failPut {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		f.puts++
		f.objects[key] = FakeObject{ContentType: r.Header.Get("Content-Type"), CacheControl: r.Header.Get("Cache-Control")}
		w.Header().Set("ETag", `"fake"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		f.deletes++
		delete(f.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unsupported", http.StatusMethodNotAllowed)
	}
}

// FailPuts — ทำให้การอัปครั้งถัดไปล้มเหลว (ทดสอบ 505802)
func (f *FakeS3) FailPuts(fail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failPut = fail
}

// Object — object ตาม key · ok = false เมื่อไม่มี
func (f *FakeS3) Object(key string) (FakeObject, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.objects[key]
	return o, ok
}

// Count — จำนวน object ที่มีอยู่
func (f *FakeS3) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.objects)
}

// Puts / Deletes — จำนวนคำขออัป / ลบที่เข้ามาถึง storage
func (f *FakeS3) Puts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.puts
}

func (f *FakeS3) Deletes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deletes
}
