// Package account คือ helper ร่วมของ script ที่สร้างบัญชีหลังบ้าน (create_superadmin, create_admin)
package account

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	agentAuthCore "app/app/core/agent_auth"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"golang.org/x/term"
)

// Create สร้างบัญชี role ที่กำหนด (ไม่มี parent) — รหัสผ่านอ่านจาก stdin และตรวจตาม AUTH-36
func Create(rawUsername string, role models.AgentRole) error {
	username := agentAuthCore.NormalizeUsername(rawUsername)
	if username == "" {
		return fmt.Errorf("ต้องระบุ -username")
	}

	password, err := readPassword()
	if err != nil {
		return err
	}
	if err := agentAuthDto.PasswordPolicyError("password", agentAuthCore.CheckPasswordPolicy(password)); err != nil {
		return fmt.Errorf("%s", apperr.From(err).MsgTH)
	}

	if err := configs.Load(); err != nil {
		return err
	}
	utils.SetPasswordCost(configs.Cfg.Auth.PasswordCost)
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	logger.InitLogger(true)
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}

	agent := models.UserAgent{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		Status:       models.AgentStatusActive,
	}
	if err := postgres.CreateUserAgentRepository(database.DBConn, &agent); err != nil {
		return err
	}
	fmt.Printf("สร้าง %s %q สำเร็จ (id=%d)\n", role, agent.Username, agent.ID)
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
