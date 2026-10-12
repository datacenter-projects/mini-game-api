package asset

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"reflect"
	"testing"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func gifBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.Black, color.White})
	if err := gif.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// webp ที่มีแค่ header พอให้อ่านขนาดได้ (ไม่ใช่ภาพที่ถอดรหัสได้)
func webpHeader(kind string, w, h int) []byte {
	d := make([]byte, 40)
	copy(d[0:], "RIFF")
	binary.LittleEndian.PutUint32(d[4:], 32)
	copy(d[8:], "WEBP")
	copy(d[12:], kind)
	switch kind {
	case "VP8 ":
		d[23], d[24], d[25] = 0x9d, 0x01, 0x2a
		binary.LittleEndian.PutUint16(d[26:], uint16(w)) //nolint:gosec // ขนาดใน test อยู่ในช่วง 0–16383
		binary.LittleEndian.PutUint16(d[28:], uint16(h)) //nolint:gosec // ขนาดใน test อยู่ในช่วง 0–16383
	case "VP8L":
		d[20] = 0x2f
		x, y := w-1, h-1
		d[21] = byte(x & 0xff)
		d[22] = byte((x>>8)&0x3f) | byte((y&0x3)<<6)
		d[23] = byte((y >> 2) & 0xff)
		d[24] = byte((y >> 10) & 0x0f)
	case "VP8X":
		x, y := w-1, h-1
		d[24], d[25], d[26] = byte(x), byte(x>>8), byte(x>>16) //nolint:gosec // ตั้งใจตัดเหลือไบต์ล่าง (little-endian 24 บิต)
		d[27], d[28], d[29] = byte(y), byte(y>>8), byte(y>>16) //nolint:gosec // ตั้งใจตัดเหลือไบต์ล่าง (little-endian 24 บิต)
	}
	return d
}

func TestDetectImage(t *testing.T) { // AST-16 · AST-17
	tests := []struct {
		name string
		data []byte
		kind ImageKind
		ct   string
		w, h int
		size bool
	}{
		{"png", pngBytes(t, 425, 425), ImagePNG, "image/png", 425, 425, true},
		{"jpeg", jpegBytes(t, 608, 1080), ImageJPEG, "image/jpeg", 608, 1080, true},
		{"gif", gifBytes(t, 32, 32), ImageGIF, "image/gif", 32, 32, true},
		{"webp lossy (VP8)", webpHeader("VP8 ", 502, 514), ImageWebP, "image/webp", 502, 514, true},
		{"webp lossless (VP8L)", webpHeader("VP8L", 1920, 1080), ImageWebP, "image/webp", 1920, 1080, true},
		{"webp extended (VP8X)", webpHeader("VP8X", 544, 851), ImageWebP, "image/webp", 544, 851, true},
	}
	for _, tt := range tests {
		got, err := DetectImage(tt.data)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if got.Kind != tt.kind || got.ContentType != tt.ct || got.Ext() != string(tt.kind) {
			t.Errorf("%s: kind/ct got %s/%s", tt.name, got.Kind, got.ContentType)
		}
		if got.SizeKnown != tt.size || got.Width != tt.w || got.Height != tt.h {
			t.Errorf("%s: size got %dx%d known=%v, want %dx%d", tt.name, got.Width, got.Height, got.SizeKnown, tt.w, tt.h)
		}
	}
}

func TestDetectImageRejects(t *testing.T) { // AST-16: ตรวจจากเนื้อไฟล์ · ห้าม SVG
	tests := []struct {
		name string
		data []byte
	}{
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)},
		{"svg มี xml header", []byte(`<?xml version="1.0"?><svg></svg>`)},
		{"pdf", []byte("%PDF-1.4\n...")},
		{"html", []byte("<html></html>")},
		{"ว่าง", nil},
		{"สั้นเกิน", []byte("\x89PN")},
		{"RIFF แต่ไม่ใช่ WEBP (wav)", append([]byte("RIFF\x00\x00\x00\x00WAVE"), make([]byte, 30)...)},
		{"ข้อความที่นามสกุลอาจเป็น .png", []byte("hello")},
	}
	for _, tt := range tests {
		if _, err := DetectImage(tt.data); !errors.Is(err, ErrUnsupportedImage) {
			t.Errorf("%s: ต้อง ErrUnsupportedImage ได้ %v", tt.name, err)
		}
	}
}

func TestDetectImageFailOpenWhenSizeUnreadable(t *testing.T) { // AST-17: อ่านขนาดไม่ได้ = ผ่าน
	trunc := pngBytes(t, 10, 10)[:20] // magic ถูก แต่ถูกตัดจน decoder อ่านขนาดไม่ได้
	got, err := DetectImage(trunc)
	if err != nil {
		t.Fatalf("ต้องไม่ error: %v", err)
	}
	if got.Kind != ImagePNG || got.SizeKnown {
		t.Errorf("ต้องเป็น png ที่ไม่รู้ขนาด: %+v", got)
	}

	badWebp := webpHeader("VP8 ", 100, 100)
	badWebp[23] = 0 // start code ผิด
	got, err = DetectImage(badWebp)
	if err != nil || got.Kind != ImageWebP || got.SizeKnown {
		t.Errorf("webp อ่านขนาดไม่ได้ ต้องผ่านแบบไม่รู้ขนาด: %+v err=%v", got, err)
	}
	unknown := webpHeader("VP8 ", 1, 1)
	copy(unknown[12:], "ABCD")
	if got, err = DetectImage(unknown); err != nil || got.SizeKnown {
		t.Errorf("chunk ที่ไม่รู้จัก ต้องผ่านแบบไม่รู้ขนาด: %+v err=%v", got, err)
	}
}

func TestSizeAllowed(t *testing.T) { // AST-17
	icon := []Size{{32, 32}, {40, 40}}
	tests := []struct {
		name    string
		allowed []Size
		w, h    int
		want    bool
	}{
		{"ช่องไม่มีสเปก ขนาดใดก็ได้", nil, 123, 45, true},
		{"สเปกว่าง", []Size{}, 1, 1, true},
		{"ตรงขนาดแรก", icon, 32, 32, true},
		{"ตรงขนาดที่ 2", icon, 40, 40, true},
		{"ไม่ตรงทั้งคู่", icon, 36, 36, false},
		{"กว้างตรง สูงไม่ตรง", icon, 32, 40, false},
	}
	for _, tt := range tests {
		if got := SizeAllowed(tt.allowed, tt.w, tt.h); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseAllowedSizes(t *testing.T) { // AST-12
	got, err := ParseAllowedSizes(`[{"width":32,"height":32},{"width":40,"height":40}]`)
	if err != nil || !reflect.DeepEqual(got, []Size{{32, 32}, {40, 40}}) {
		t.Errorf("got %v err=%v", got, err)
	}
	for _, empty := range []string{"", "  ", "[]", "null"} {
		got, err := ParseAllowedSizes(empty)
		if err != nil || len(got) != 0 {
			t.Errorf("%q: ต้องไม่มีสเปก ได้ %v err=%v", empty, got, err)
		}
	}
	if _, err := ParseAllowedSizes("{not json"); err == nil {
		t.Error("JSON พังต้อง error")
	}
}

func TestFormatSizes(t *testing.T) {
	if got := FormatSizes([]Size{{32, 32}, {40, 40}}); got != "32×32, 40×40" {
		t.Errorf("got %q", got)
	}
	if got := FormatSizes(nil); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestMaxFileSize(t *testing.T) { // AST-16 · lead GR3
	if MaxFileSize != 5*1024*1024 {
		t.Errorf("MaxFileSize ต้อง 5 MB ได้ %d", MaxFileSize)
	}
}
