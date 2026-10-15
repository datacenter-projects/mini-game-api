// seed_loadtest สร้างบัญชี AGENT สำหรับ load test (tests/load) — ใช้กับ test env เท่านั้น (docs/TESTING.md)
//
//	go run ./scripts/seed_loadtest -count 100              # load0001 … load0100
//	go run ./scripts/seed_loadtest -count 500 -prefix lt
//
// env มาจาก environment (เช่น scripts/testenv.ps1 seed ที่โหลด .env.test ให้) · APP_ENV dev/uat/prod ไม่ยอมรัน
// รันซ้ำได้: username ที่มีอยู่แล้วถูกข้าม · ทุกบัญชีใช้รหัสผ่านเดียวกันและตั้ง passcode แล้ว (ผ่านด่านหลัง login)
package main

import (
	"flag"
	"fmt"
	"os"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm/clause"
)

// ค่าเริ่มต้นตรงกับ tests/load/lib/config.js — เป็นรหัสของบัญชีทดสอบเท่านั้น
const (
	defaultPassword = "LoadTest1!"
	defaultPasscode = "123456"
)

func main() {
	count := flag.Int("count", 100, "จำนวนบัญชี")
	prefix := flag.String("prefix", "load", "ขึ้นต้น username (a-z 0-9)")
	password := flag.String("password", defaultPassword, "รหัสผ่านของทุกบัญชี")
	passcode := flag.String("passcode", defaultPasscode, "passcode ของทุกบัญชี")
	flag.Parse()

	if err := run(*count, *prefix, *password, *passcode); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(count int, prefix, password, passcode string) error {
	if count < 1 || count > 9999 {
		return fmt.Errorf("-count ต้องอยู่ระหว่าง 1–9999")
	}
	if agentAuthCore.CheckPasswordPolicy(password) != agentAuthCore.PasswordOK {
		return fmt.Errorf("-password ไม่ผ่านกติการหัสผ่าน (AUTH-36)")
	}
	if !agentAuthCore.IsValidPasscode(passcode) {
		return fmt.Errorf("-passcode ต้องเป็นตัวเลข 6 หลัก")
	}

	if err := configs.Load(); err != nil {
		return err
	}
	switch configs.Cfg.AppEnv {
	case "dev", "uat", "prod":
		return fmt.Errorf("APP_ENV=%s: seed_loadtest ใช้กับ test env เท่านั้น", configs.Cfg.AppEnv)
	}
	logger.InitLogger(true)
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}

	// bcrypt ครั้งเดียวแล้วใช้ร่วมกันทุกบัญชี (cost 12 ~250ms ต่อครั้ง)
	pwHash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	pcHash, err := utils.HashPassword(passcode)
	if err != nil {
		return err
	}

	agents := make([]models.UserAgent, 0, count)
	for i := 1; i <= count; i++ {
		agents = append(agents, models.UserAgent{
			Username:     fmt.Sprintf("%s%04d", prefix, i),
			PasswordHash: pwHash,
			PasscodeHash: &pcHash,
			Role:         models.AgentRoleAgent,
			Status:       models.AgentStatusActive,
		})
	}
	res := database.DBConn.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&agents, 500)
	if res.Error != nil {
		return res.Error
	}
	fmt.Printf("seed_loadtest: สร้างใหม่ %d บัญชี (%s0001 – %s%04d)\n", res.RowsAffected, prefix, prefix, count)
	return nil
}
