// miniredis เปิด Redis จำลอง (รองรับคำสั่งของ Redis 7) สำหรับรัน integration test บนเครื่องที่ไม่มี Redis 7
// เช่น Windows Server ที่ใช้ Docker ไม่ได้ — docs/TESTING.md
//
//	go run ./scripts/miniredis                       # 127.0.0.1:56379
//	go run ./scripts/miniredis -addr 127.0.0.1:6380
//
// ใช้แทน Redis จริงเพื่อทดสอบเท่านั้น: ข้อมูลอยู่ใน memory ปิดแล้วหาย · ผลที่ยืนยันได้จริงต้องรันกับ Redis 7 (CI)
// miniredis ไม่ลด TTL ตามเวลาจริงเอง จึงเลื่อนเวลาตามนาฬิกาทุก 50ms
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:56379", "address ที่จะ listen")
	flag.Parse()

	m := miniredis.NewMiniRedis()
	if err := m.StartAddr(*addr); err != nil {
		log.Fatal(err)
	}
	defer m.Close()
	log.Println("miniredis listening on", m.Addr())

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	last := time.Now()
	for {
		select {
		case now := <-ticker.C:
			m.FastForward(now.Sub(last))
			last = now
		case <-stop:
			return
		}
	}
}
