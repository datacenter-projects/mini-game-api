// fakes3 เปิด S3 จำลองในเครื่อง (PUT / GET / HEAD / DELETE แบบ path-style) สำหรับลองอัปรูป asset
// บนเครื่องที่ไม่มี Docker / MinIO — MinIO ปิดการแจกไบนารีแล้ว (docs/TESTING.md)
//
//	go run ./scripts/fakes3                          # 127.0.0.1:9000 · bucket minigame-asset-local · เก็บที่ .testenv/fakes3
//	go run ./scripts/fakes3 -addr 127.0.0.1:9100 -bucket my-bucket -dir ./tmp/s3
//
// ตั้งค่าฝั่ง API (.env):
//
//	S3_ENDPOINT=http://127.0.0.1:9000
//	S3_BUCKET=minigame-asset-local
//	S3_CDN_BASE_URL=http://127.0.0.1:9000/minigame-asset-local   # เปิด URL ของรูปในเบราว์เซอร์ได้
//	AWS_ACCESS_KEY_ID=local / AWS_SECRET_ACCESS_KEY=local        # ค่าอะไรก็ได้ — ไม่ตรวจลายเซ็น
//
// ใช้ทดสอบเท่านั้น: ไม่ตรวจ credential / ลายเซ็น · listen เฉพาะ loopback · ผลที่ยืนยันจริงคือ S3 จริง
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type meta struct {
	ContentType  string `json:"content_type"`
	CacheControl string `json:"cache_control"`
}

type server struct {
	bucket string
	dir    string
}

func main() {
	addr := flag.String("addr", "127.0.0.1:9000", "address ที่จะ listen (ต้องเป็น loopback)")
	bucket := flag.String("bucket", "minigame-asset-local", "ชื่อ bucket ที่รับ")
	dir := flag.String("dir", filepath.Join(".testenv", "fakes3"), "โฟลเดอร์เก็บไฟล์")
	flag.Parse()

	host, _, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatal(err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		log.Fatalf("addr ต้องเป็น loopback เท่านั้น (ได้ %q) — S3 จำลองไม่ตรวจสิทธิ์", host)
	}

	root, err := filepath.Abs(filepath.Join(*dir, *bucket))
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		log.Fatal(err)
	}
	s := &server{bucket: *bucket, dir: root}
	log.Printf("fakes3 listening on http://%s · bucket %q · เก็บที่ %s", *addr, *bucket, root)
	log.Fatal(http.ListenAndServe(*addr, s))
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	bucket, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if bucket != s.bucket {
		s.fail(w, http.StatusNotFound, "NoSuchBucket", "bucket not found")
		return
	}
	file, ok := s.resolve(key)
	if !ok {
		s.fail(w, http.StatusBadRequest, "InvalidArgument", "invalid key")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.put(w, r, file, key)
	case http.MethodGet, http.MethodHead:
		s.get(w, r, file)
	case http.MethodDelete:
		_ = os.Remove(file)
		_ = os.Remove(file + ".meta")
		w.WriteHeader(http.StatusNoContent)
		log.Printf("DELETE %s", key)
	default:
		s.fail(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not supported")
	}
}

// resolve แปลง key เป็นพาธไฟล์ใต้ root — ปฏิเสธ key ว่าง / พยายามออกนอก root
func (s *server) resolve(key string) (string, bool) {
	if key == "" || strings.Contains(key, "\\") || strings.HasSuffix(key, ".meta") {
		return "", false
	}
	clean := path.Clean("/" + key)
	if clean == "/" || strings.Contains(key, "..") {
		return "", false
	}
	full := filepath.Join(s.dir, filepath.FromSlash(clean))
	if rel, err := filepath.Rel(s.dir, full); err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return full, true
}

func (s *server) put(w http.ResponseWriter, r *http.Request, file, key string) {
	var body io.Reader = r.Body
	if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") || strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") {
		body = decodeAWSChunked(r.Body) // SDK ส่งแบบ aws-chunked (มี checksum trailer) — ถอดเอาเฉพาะเนื้อไฟล์
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		s.fail(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	tmp := file + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	n, err := io.Copy(f, body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		s.fail(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	if err := os.Rename(tmp, file); err != nil {
		_ = os.Remove(tmp)
		s.fail(w, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	m, _ := json.Marshal(meta{ContentType: r.Header.Get("Content-Type"), CacheControl: r.Header.Get("Cache-Control")})
	_ = os.WriteFile(file+".meta", m, 0o644)
	w.Header().Set("ETag", `"fakes3"`)
	w.WriteHeader(http.StatusOK)
	log.Printf("PUT %s (%d bytes, %s)", key, n, r.Header.Get("Content-Type"))
}

func (s *server) get(w http.ResponseWriter, r *http.Request, file string) {
	st, err := os.Stat(file)
	if err != nil || st.IsDir() {
		s.fail(w, http.StatusNotFound, "NoSuchKey", "key not found")
		return
	}
	if raw, err := os.ReadFile(file + ".meta"); err == nil {
		var m meta
		if json.Unmarshal(raw, &m) == nil {
			if m.ContentType != "" {
				w.Header().Set("Content-Type", m.ContentType)
			}
			if m.CacheControl != "" {
				w.Header().Set("Cache-Control", m.CacheControl)
			}
		}
	}
	f, err := os.Open(file)
	if err != nil {
		s.fail(w, http.StatusNotFound, "NoSuchKey", "key not found")
		return
	}
	defer f.Close()
	http.ServeContent(w, r, "", st.ModTime(), f)
}

func (s *server) fail(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`, code, msg)
}

// decodeAWSChunked ถอด body แบบ aws-chunked: "<hex size>[;chunk-signature=...]\r\n<data>\r\n" ... "0\r\n" + trailer
// เป็น io.Reader ธรรมดา (ไม่ใช้ goroutine) — handler เลิกอ่านกลางทางเมื่อไรก็ไม่มีอะไรค้าง
func decodeAWSChunked(r io.Reader) io.Reader {
	return &chunkedReader{br: bufio.NewReader(r)}
}

type chunkedReader struct {
	br     *bufio.Reader
	remain int64 // byte ที่เหลือของ chunk ปัจจุบัน
	done   bool
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if c.done {
		return 0, io.EOF
	}
	if c.remain == 0 {
		line, err := c.br.ReadString('\n')
		if err != nil {
			return 0, unexpectedEOF(err) // body จบก่อนเจอ chunk สุดท้าย (0) = ไม่ครบ ไม่ใช่จบปกติ
		}
		sizeText, _, _ := strings.Cut(strings.TrimSpace(line), ";")
		size, err := strconv.ParseInt(sizeText, 16, 64)
		if err != nil || size < 0 {
			return 0, errors.New("bad aws-chunked size")
		}
		if size == 0 {
			c.done = true
			_, _ = io.Copy(io.Discard, c.br) // trailer (x-amz-checksum-*) ทิ้ง
			return 0, io.EOF
		}
		c.remain = size
	}
	if int64(len(p)) > c.remain {
		p = p[:c.remain]
	}
	n, err := c.br.Read(p)
	c.remain -= int64(n)
	if c.remain == 0 && err == nil {
		_, err = c.br.Discard(2) // \r\n หลังข้อมูลแต่ละ chunk
	}
	return n, unexpectedEOF(err)
}

// unexpectedEOF — ระหว่างถอด chunk ยังไม่ถึง chunk สุดท้าย io.EOF แปลว่า body ขาด
func unexpectedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
