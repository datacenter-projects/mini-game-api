// create_admin สร้างบัญชี ADMIN (ทีม support) — ช่องทางเดียวที่สร้าง ADMIN ได้ (AUTH-43)
//
//	go run ./scripts/create_admin -username support01
//
// รหัสผ่านพิมพ์ทาง stdin (ไม่แสดงบนจอ) — ไม่รับผ่าน argument เพื่อไม่ให้ค้างใน shell history
// รันใน pipeline ได้ด้วย: echo "$PASSWORD" | go run ./scripts/create_admin -username support01
// รหัสผ่านต้องผ่านกติกา AUTH-36 (docs/modules/agent_auth_phase2.md)
package main

import (
	"flag"
	"fmt"
	"os"

	"app/app/models"
	"app/scripts/internal/account"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	username := flag.String("username", "", "username ของ admin (a-z, 0-9)")
	flag.Parse()

	if err := account.Create(*username, models.AgentRoleAdmin); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
