// reset_credentials รีเซ็ตรหัสผ่าน / passcode ของ SUPERADMIN หรือ ADMIN (กู้บัญชีที่ลืมรหัส — AUTH-51)
//
//	go run ./scripts/reset_credentials -username superadmin1 -password
//	go run ./scripts/reset_credentials -username superadmin1 -passcode
//	go run ./scripts/reset_credentials -username superadmin1 -password -passcode
//
// บัญชีอื่น (Company / Share / Agent / sub) ใช้ API admin reset แทน
// ค่าชั่วคราวสุ่มให้ แสดงบนจอครั้งเดียว หมดอายุตาม TEMP_CREDENTIAL_TTL (ค่าเริ่มต้น 24 ชม.) · login แล้วถูกบังคับเปลี่ยน
// บันทึก auth_audit_logs (actor = SCRIPT · actor_username = user ของเครื่องที่รัน)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"time"

	adminManagementService "app/app/service/admin_management"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	username := flag.String("username", "", "username ของ SUPERADMIN / ADMIN")
	password := flag.Bool("password", false, "รีเซ็ตรหัสผ่าน")
	passcode := flag.Bool("passcode", false, "รีเซ็ต passcode")
	flag.Parse()

	if err := run(*username, *password, *passcode); err != nil {
		if ae := apperr.From(err); ae.Code != apperr.ErrInternal.Code {
			err = fmt.Errorf("%s", ae.MsgTH)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(username string, password, passcode bool) error {
	if username == "" {
		return fmt.Errorf("ต้องระบุ -username")
	}
	if err := configs.Load(); err != nil {
		return err
	}
	utils.SetPasswordCost(configs.Cfg.Auth.PasswordCost)
	logger.InitLogger(true)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}
	if err := database.RedisConnection(ctx, configs.Cfg.Redis); err != nil {
		return err
	}

	res, err := adminManagementService.ScriptResetCredentialsService(ctx, adminManagementService.ScriptResetRequest{
		Username: username, Password: password, Passcode: passcode, Operator: operator(),
	})
	if err != nil {
		return err
	}

	fmt.Printf("รีเซ็ต %q สำเร็จ — ค่าชั่วคราวแสดงครั้งเดียว ส่งให้เจ้าของบัญชีทางช่องทางที่ปลอดภัย\n", res.Username)
	if res.TempPassword != "" {
		fmt.Println("  รหัสผ่านชั่วคราว:", res.TempPassword)
	}
	if res.TempPasscode != "" {
		fmt.Println("  passcode ชั่วคราว:", res.TempPasscode)
	}
	fmt.Println("  หมดอายุ:", res.TempExpiresAt.Format(time.RFC3339))
	return nil
}

// operator — user ของเครื่องที่รัน script (เก็บใน audit log) · หาไม่ได้ = ""
func operator() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.Username
}
