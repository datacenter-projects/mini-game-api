// Package playerauth คือ request / response ของเส้น /transfer/* (operator สาย Transfer) — spec: docs/modules/player_auth.md หัวข้อ 5
// ชื่อ field เป็น snake_case เหมือน API ทั้งระบบ (lead Q9) · ข้อความ error ใช้ชื่อ field ดิบ เพราะผู้เรียกเป็น server ของ operator
package playerauth

import (
	"strings"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	playerAuthCore "app/app/core/player_auth"
	"app/pkg/apperr"
)

const (
	maxAgentUsername = 100
	maxKey           = 200
	maxPassword      = 72 // bcrypt รับได้สูงสุด 72 byte
)

// OperatorCreds — ส่วนที่ middleware operator อ่านจาก body ทุกเส้น (PA-10)
type OperatorCreds struct {
	AgentUsername string `json:"agent_username"`
	Key           string `json:"key"`
}

func (r *OperatorCreds) Validate() error {
	if strings.TrimSpace(r.AgentUsername) == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก agent_username", "agent_username is required")
	}
	if len(r.AgentUsername) > maxAgentUsername {
		return apperr.ErrValidation.WithMessage("agent_username ยาวเกิน 100 ตัวอักษร", "agent_username must not exceed 100 characters")
	}
	if r.Key == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก key", "key is required")
	}
	if len(r.Key) > maxKey {
		return apperr.ErrValidation.WithMessage("key ไม่ถูกต้อง", "key is invalid")
	}
	return nil
}

func validatePlayerUsername(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return apperr.ErrValidation.WithMessage("กรุณากรอก player_username", "player_username is required")
	}
	if !playerAuthCore.IsPlayerUsername(playerAuthCore.NormalizeUsername(raw)) {
		return apperr.ErrValidation.WithMessage("player_username ต้องเป็น a-z, 0-9 ยาว 3–32 ตัว", "player_username must be 3–32 characters of a-z and 0-9")
	}
	return nil
}

// validateCurrencyFormat — รูปแบบเท่านั้น (ว่าง = ไม่ส่ง · 3–4 ตัวอักษร) · ว่าอยู่ในสกุลของ agent ไหมตรวจที่ service
func validateCurrencyFormat(c string) error {
	c = strings.TrimSpace(c)
	if c == "" {
		return nil
	}
	if len(c) < 3 || len(c) > 4 {
		return apperr.ErrValidation.WithMessage("currency ไม่ถูกต้อง", "currency is invalid")
	}
	return nil
}

// TransferLoginRequest — L2 POST /api/v1/transfer/login · currency / language ไม่บังคับ
type TransferLoginRequest struct {
	AgentUsername  string `json:"agent_username"`
	Key            string `json:"key"`
	PlayerUsername string `json:"player_username"`
	Currency       string `json:"currency"`
	Language       string `json:"language"`
}

func (r *TransferLoginRequest) Validate() error {
	if err := (&OperatorCreds{AgentUsername: r.AgentUsername, Key: r.Key}).Validate(); err != nil {
		return err
	}
	if err := validatePlayerUsername(r.PlayerUsername); err != nil {
		return err
	}
	if err := validateCurrencyFormat(r.Currency); err != nil {
		return err
	}
	if _, ok := playerAuthCore.NormalizeLanguage(r.Language); !ok {
		return apperr.ErrValidation.WithMessage("language ต้องเป็น th หรือ en", "language must be th or en")
	}
	return nil
}

// TransferLoginResponse — L2 · Link = lobby URL + ?token=…&lang=… (LQ-c)
type TransferLoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Link      string    `json:"link"`
}

// TransferRegisterRequest — TR POST /api/v1/transfer/register · password ไม่บังคับ (ไม่ส่งหรือส่ง "" = ผู้เล่นเข้าทาง operator อย่างเดียว)
type TransferRegisterRequest struct {
	AgentUsername  string `json:"agent_username"`
	Key            string `json:"key"`
	PlayerUsername string `json:"player_username"`
	Currency       string `json:"currency"`
	Password       string `json:"password"`
}

func (r *TransferRegisterRequest) Validate() error {
	if err := (&OperatorCreds{AgentUsername: r.AgentUsername, Key: r.Key}).Validate(); err != nil {
		return err
	}
	if err := validatePlayerUsername(r.PlayerUsername); err != nil {
		return err
	}
	if err := validateCurrencyFormat(r.Currency); err != nil {
		return err
	}
	if r.Password == "" {
		return nil
	}
	if len(r.Password) > maxPassword {
		return apperr.ErrValidation.WithMessage("password ยาวเกิน 72 ตัวอักษร", "password must not exceed 72 characters")
	}
	if v := agentAuthCore.CheckPasswordPolicy(r.Password); v != agentAuthCore.PasswordOK {
		th, en := agentAuthCore.PasswordViolationMessage(v)
		return apperr.ErrValidation.WithMessage("password "+th, "password "+en)
	}
	return nil
}

// TransferRegisterResponse — TR · ไม่ออก token / link · ไม่คืนรหัส
type TransferRegisterResponse struct {
	Username string `json:"username"`
	Currency string `json:"currency"`
}
