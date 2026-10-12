package asset

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // register decoder สำหรับ image.DecodeConfig
	_ "image/jpeg" // register decoder
	_ "image/png"  // register decoder
	"strings"
)

// MaxFileSize — ขนาดไฟล์สูงสุดของรูป (AST-16 · lead GR3)
const MaxFileSize = 5 * 1024 * 1024

// ImageKind — ชนิดรูปที่รับ (AST-16) · SVG และอื่นๆ ไม่รับ
type ImageKind string

const (
	ImagePNG  ImageKind = "png"
	ImageJPEG ImageKind = "jpeg"
	ImageGIF  ImageKind = "gif"
	ImageWebP ImageKind = "webp"
)

// ErrUnsupportedImage — เนื้อไฟล์ไม่ใช่ png / jpeg / gif / webp (ไม่เชื่อ Content-Type ที่ client ส่ง)
var ErrUnsupportedImage = errors.New("unsupported image type")

// ImageInfo — ผลตรวจเนื้อไฟล์ · SizeKnown = false เมื่ออ่านขนาดภาพไม่ได้ (fail-open — AST-17)
type ImageInfo struct {
	Kind        ImageKind
	ContentType string
	Width       int
	Height      int
	SizeKnown   bool
}

// Ext — นามสกุลของ storage key (AST-16)
func (i ImageInfo) Ext() string { return string(i.Kind) }

// DetectImage — ตรวจชนิดจาก magic bytes ของเนื้อไฟล์ แล้วอ่านขนาดภาพ (AST-16, AST-17)
func DetectImage(data []byte) (ImageInfo, error) {
	var info ImageInfo
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		info = ImageInfo{Kind: ImagePNG, ContentType: "image/png"}
	case bytes.HasPrefix(data, []byte("\xff\xd8\xff")):
		info = ImageInfo{Kind: ImageJPEG, ContentType: "image/jpeg"}
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		info = ImageInfo{Kind: ImageGIF, ContentType: "image/gif"}
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		info = ImageInfo{Kind: ImageWebP, ContentType: "image/webp"}
	default:
		return ImageInfo{}, ErrUnsupportedImage
	}

	if info.Kind == ImageWebP {
		info.Width, info.Height, info.SizeKnown = webpSize(data)
		return info, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil { // อ่านขนาดไม่ได้ = ผ่าน (fail-open — ขนาดภาพเป็นเรื่องหน้าตา ไม่ใช่ความปลอดภัย)
		return info, nil
	}
	info.Width, info.Height, info.SizeKnown = cfg.Width, cfg.Height, cfg.Width > 0 && cfg.Height > 0
	return info, nil
}

// webpSize — อ่านขนาดจาก header ของ WebP (VP8 lossy · VP8L lossless · VP8X extended) ไม่ถอดรหัสภาพ
func webpSize(d []byte) (w, h int, ok bool) {
	if len(d) < 30 {
		return 0, 0, false
	}
	switch string(d[12:16]) {
	case "VP8 ": // lossy: start code 9d 01 2a ที่ 23..25 · กว้าง / สูง 14 บิต
		if d[23] != 0x9d || d[24] != 0x01 || d[25] != 0x2a {
			return 0, 0, false
		}
		w = int(binary.LittleEndian.Uint16(d[26:28]) & 0x3fff)
		h = int(binary.LittleEndian.Uint16(d[28:30]) & 0x3fff)
	case "VP8L": // lossless: signature 0x2f ที่ 20 · กว้าง / สูง 14 บิตต่อ − 1
		if d[20] != 0x2f {
			return 0, 0, false
		}
		w = 1 + (int(d[21]) | int(d[22]&0x3f)<<8)
		h = 1 + (int(d[22]>>6) | int(d[23])<<2 | int(d[24]&0x0f)<<10)
	case "VP8X": // extended: canvas กว้าง / สูง 24 บิต − 1 ที่ 24..29
		w = 1 + (int(d[24]) | int(d[25])<<8 | int(d[26])<<16)
		h = 1 + (int(d[27]) | int(d[28])<<8 | int(d[29])<<16)
	default:
		return 0, 0, false
	}
	return w, h, w > 0 && h > 0
}

// Size — ขนาดภาพที่ช่องรับ (AST-12)
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ParseAllowedSizes — อ่าน asset_types.allowed_sizes (JSONB) · "" / "[]" = ไม่ตรวจขนาด
func ParseAllowedSizes(raw string) ([]Size, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var sizes []Size
	if err := json.Unmarshal([]byte(raw), &sizes); err != nil {
		return nil, fmt.Errorf("allowed_sizes: %w", err)
	}
	return sizes, nil
}

// SizeAllowed — ขนาดภาพตรงขนาดใดขนาดหนึ่งที่ช่องรับ · ช่องไม่มีสเปก = ผ่าน (AST-17)
func SizeAllowed(allowed []Size, w, h int) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, s := range allowed {
		if s.Width == w && s.Height == h {
			return true
		}
	}
	return false
}

// FormatSizes — ข้อความขนาดที่รับ เช่น "32×32, 40×40" (ใช้ใน msg 405101)
func FormatSizes(allowed []Size) string {
	parts := make([]string, len(allowed))
	for i, s := range allowed {
		parts[i] = fmt.Sprintf("%d×%d", s.Width, s.Height)
	}
	return strings.Join(parts, ", ")
}
