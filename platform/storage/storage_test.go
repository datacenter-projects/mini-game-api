package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"app/pkg/configs"
)

func TestNotConfigured(t *testing.T) { // AST-30: ไม่ตั้ง bucket = ไม่พร้อม แต่ boot ได้ (ไม่ error)
	if err := Init(context.Background(), configs.StorageConfig{}); err != nil {
		t.Fatalf("Init ที่ไม่ตั้ง bucket ต้องไม่ error: %v", err)
	}
	if Available() {
		t.Fatal("ไม่ตั้ง bucket ต้องไม่พร้อม")
	}
	if err := Put(context.Background(), "k", []byte("x"), "image/png", "x"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Put: want ErrUnavailable got %v", err)
	}
	if err := Delete(context.Background(), "k"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Delete: want ErrUnavailable got %v", err)
	}
}

func TestInitConfigured(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Cleanup(func() { _ = Init(context.Background(), configs.StorageConfig{}) })

	// ตั้ง bucket แล้วพร้อม (ยังไม่ติดต่อ S3 จนกว่าจะเรียก Put / Delete) · endpoint ที่ติดต่อไม่ได้ = อัปล้ม ไม่ใช่ "ไม่พร้อม"
	err := Init(context.Background(), configs.StorageConfig{Region: "ap-southeast-1", Bucket: "b", Endpoint: "http://127.0.0.1:1", UploadTimeout: 500 * time.Millisecond})
	if err != nil || !Available() {
		t.Fatalf("Init: err=%v available=%v", err, Available())
	}
	if err := Put(context.Background(), "k", []byte("x"), "image/png", "x"); !errors.Is(err, ErrUploadFailed) {
		t.Errorf("Put ไปยัง endpoint ที่ติดต่อไม่ได้: want ErrUploadFailed got %v", err)
	}
	// Init ใหม่โดยไม่ตั้ง bucket = กลับเป็นไม่พร้อม
	if err := Init(context.Background(), configs.StorageConfig{}); err != nil || Available() {
		t.Errorf("Init ว่าง: err=%v available=%v", err, Available())
	}
}

func TestInitResetsTimeout(t *testing.T) { // Init ครั้งหลังที่ไม่ตั้ง timeout ต้องกลับเป็นค่าตั้งต้น ไม่ใช้ค่าของครั้งก่อน
	t.Cleanup(func() { _ = Init(context.Background(), configs.StorageConfig{}) })
	_ = Init(context.Background(), configs.StorageConfig{UploadTimeout: 500 * time.Millisecond})
	if timeout != 500*time.Millisecond {
		t.Fatalf("timeout = %v, want 500ms", timeout)
	}
	_ = Init(context.Background(), configs.StorageConfig{})
	if timeout != defaultTimeout {
		t.Fatalf("Init ที่ไม่ตั้ง timeout: timeout = %v, want %v", timeout, defaultTimeout)
	}
}
