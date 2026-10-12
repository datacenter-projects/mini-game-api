// Package storage คือ client ของ object storage (S3 / MinIO) สำหรับไฟล์ asset — docs/modules/asset.md
//
// เรียก Init ตอน boot · ไม่พร้อม (ไม่ตั้ง bucket / สร้าง client ไม่ได้) = ระบบยัง boot ได้ (AST-30):
// เส้นที่ต้องแตะ storage ตอบ ErrUnavailable ส่วนเส้นอื่นทำงานปกติ
// credential ใช้ default chain ของ AWS SDK (IRSA บน k8s) — ห้ามเก็บ key ใน config ของเรา
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"app/pkg/configs"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	// ErrUnavailable — storage ยังไม่พร้อม (ไม่ได้ตั้ง bucket หรือสร้าง client ไม่สำเร็จ) → 505801
	ErrUnavailable = errors.New("storage: not available")
	// ErrUploadFailed — เรียก storage แล้วไม่สำเร็จ / timeout → 505802
	ErrUploadFailed = errors.New("storage: request failed")
)

var (
	client  *s3.Client
	bucket  string
	timeout = 30 * time.Second
)

// Init สร้าง client ตาม config · คืน error เมื่อสร้างไม่ได้ (ผู้เรียก log แล้ว boot ต่อ — storage จะไม่พร้อม)
// ไม่ตั้ง Bucket = ไม่พร้อมโดยไม่ถือเป็น error (เครื่อง local ที่ไม่ใช้ storage)
func Init(ctx context.Context, c configs.StorageConfig) error {
	client, bucket = nil, ""
	if c.UploadTimeout > 0 {
		timeout = c.UploadTimeout
	}
	if c.Bucket == "" {
		return nil
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(c.Region))
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}
	client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if c.Endpoint != "" { // MinIO / S3 จำลอง — ใช้ path-style
			o.BaseEndpoint = aws.String(c.Endpoint)
			o.UsePathStyle = true
		}
	})
	bucket = c.Bucket
	return nil
}

// Available — พร้อมอัป / ลบไฟล์ไหม
func Available() bool { return client != nil && bucket != "" }

// Put อัป object ตาม key · contentType / cacheControl ตั้งที่ object (AST-16)
func Put(ctx context.Context, key string, body []byte, contentType, cacheControl string) error {
	if !Available() {
		return ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(body),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String(cacheControl),
	})
	if err != nil {
		return fmt.Errorf("%w: put %s: %w", ErrUploadFailed, key, err)
	}
	return nil
}

// Delete ลบ object ตาม key (ใช้ย้อนกลับเมื่อบันทึก DB ไม่สำเร็จ — AST-19) · ไม่มี object ก็ไม่ error (S3 ตอบสำเร็จ)
func Delete(ctx context.Context, key string) error {
	if !Available() {
		return ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("%w: delete %s: %w", ErrUploadFailed, key, err)
	}
	return nil
}
