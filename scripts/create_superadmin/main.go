// create_superadmin สร้างบัญชี SUPERADMIN (บัญชีแรกของระบบ)
//
//	go run ./scripts/create_superadmin -username admin
//
// รหัสผ่านพิมพ์ทาง stdin (ไม่แสดงบนจอ) — ไม่รับผ่าน argument เพื่อไม่ให้ค้างใน shell history
// รันใน pipeline ได้ด้วย: echo "$PASSWORD" | go run ./scripts/create_superadmin -username admin
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/term"
)

const minPasswordLen = 12

func main() {
	username := flag.String("username", "", "username ของ superadmin (a-z, 0-9)")
	flag.Parse()

	if err := run(agentAuthCore.NormalizeUsername(*username)); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(username string) error {
	if username == "" {
		return fmt.Errorf("ต้องระบุ -username")
	}

	password, err := readPassword()
	if err != nil {
		return err
	}
	if len(password) < minPasswordLen {
		return fmt.Errorf("รหัสผ่านต้องยาวอย่างน้อย %d ตัวอักษร", minPasswordLen)
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	if err := configs.Load(); err != nil {
		return err
	}
	logger.InitLogger(true)
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}

	agent := models.UserAgent{
		Username:     username,
		PasswordHash: hash,
		Role:         models.AgentRoleSuperAdmin,
		Status:       models.AgentStatusActive,
	}
	if err := postgres.CreateUserAgentRepository(database.DBConn, &agent); err != nil {
		return err
	}
	fmt.Printf("สร้าง SUPERADMIN %q สำเร็จ (id=%d)\n", agent.Username, agent.ID)
	return nil
}

func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	fmt.Fprint(os.Stderr, "Password: ")
	p1, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "Confirm password: ")
	p2, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(p1) != string(p2) {
		return "", fmt.Errorf("รหัสผ่านไม่ตรงกัน")
	}
	return string(p1), nil
}
