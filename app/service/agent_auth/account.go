package agentauth

import (
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/app/repository/postgres"

	"gorm.io/gorm"
)

// account รวมบัญชีหลังบ้าน 2 ตาราง (user_agents / subaccounts) ให้ logic ของ auth ใช้ร่วมกัน
// ค่าที่ไม่ได้ select มาจะเป็นค่าว่าง — ดูว่าโหลดด้วยฟังก์ชันไหน
type account struct {
	Type     models.AccountType
	ID       uint
	Username string
	Status   models.AgentStatus // สถานะของบัญชีนี้เอง

	// AgentID = agent ที่บัญชีนี้ทำงานในนาม: agent เอง หรือผู้สร้างถ้าเป็น sub (AUTH-26)
	AgentID  uint
	ParentID *uint            // parent ของ agent (sub = parent ของผู้สร้าง)
	Role     models.AgentRole // sub = role ของผู้สร้าง (AUTH-25)

	CreatorStatus models.AgentStatus // เฉพาะ sub

	PasswordHash          string
	PreviousPasswordHash  *string
	PasscodeHash          *string
	MustChangePassword    bool
	MustChangePasscode    bool
	TempPasswordExpiresAt *time.Time
	TempPasscodeExpiresAt *time.Time
}

func (a account) IsSub() bool       { return a.Type == models.AccountTypeSub }
func (a account) PasscodeSet() bool { return a.PasscodeHash != nil }

func (a account) EffectiveStatus() models.AgentStatus {
	return agentAuthCore.EffectiveStatus(a.Status, a.CreatorStatus)
}

func (a account) Gate() agentAuthCore.Gate {
	return agentAuthCore.CurrentGate(a.MustChangePassword, a.MustChangePasscode, a.PasscodeSet())
}

func fromUserAgent(u models.UserAgent) account {
	return account{
		Type: models.AccountTypeAgent, ID: u.ID, Username: u.Username, Status: u.Status,
		AgentID: u.ID, ParentID: u.ParentID, Role: u.Role,
		PasswordHash: u.PasswordHash, PreviousPasswordHash: u.PreviousPasswordHash, PasscodeHash: u.PasscodeHash,
		MustChangePassword: u.MustChangePassword, MustChangePasscode: u.MustChangePasscode,
		TempPasswordExpiresAt: u.TempPasswordExpiresAt, TempPasscodeExpiresAt: u.TempPasscodeExpiresAt,
	}
}

func fromSubaccount(s models.Subaccount, creator models.UserAgent) account {
	return account{
		Type: models.AccountTypeSub, ID: s.ID, Username: s.Username, Status: s.Status,
		AgentID: s.AgentID, ParentID: creator.ParentID, Role: creator.Role, CreatorStatus: creator.Status,
		PasswordHash: s.PasswordHash, PreviousPasswordHash: s.PreviousPasswordHash, PasscodeHash: s.PasscodeHash,
		MustChangePassword: s.MustChangePassword, MustChangePasscode: s.MustChangePasscode,
		TempPasswordExpiresAt: s.TempPasswordExpiresAt, TempPasscodeExpiresAt: s.TempPasscodeExpiresAt,
	}
}

// loadAccountForLogin หาบัญชีจาก username ที่ normalize แล้ว (AUTH-18) — ไม่พบคืน apperr.ErrNotFound
// sub ที่ไม่มีผู้สร้าง (ข้อมูลเสีย) ถือว่าไม่พบ
func loadAccountForLogin(db *gorm.DB, username string) (account, error) {
	if !agentAuthCore.IsSubaccountUsername(username) {
		u, err := postgres.GetUserAgentForLoginRepository(db, username)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := postgres.GetSubaccountForLoginRepository(db, username)
	if err != nil {
		return account{}, err
	}
	creator, err := postgres.GetUserAgentAuthByIDRepository(db, s.AgentID)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, creator), nil
}

// loadAccountAuth โหลดเฉพาะคอลัมน์ที่ middleware ใช้ — ไม่พบคืน apperr.ErrNotFound
func loadAccountAuth(db *gorm.DB, t models.AccountType, id uint) (account, error) {
	if t == models.AccountTypeAgent {
		u, err := postgres.GetUserAgentAuthByIDRepository(db, id)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := postgres.GetSubaccountAuthByIDRepository(db, id)
	if err != nil {
		return account{}, err
	}
	creator, err := postgres.GetUserAgentAuthByIDRepository(db, s.AgentID)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, creator), nil
}

// loadAccountCredentials อ่านรหัสผ่าน/passcode โดยไม่ lock (ไม่มีข้อมูลผู้สร้าง)
func loadAccountCredentials(db *gorm.DB, t models.AccountType, id uint) (account, error) {
	if t == models.AccountTypeAgent {
		u, err := postgres.GetUserAgentCredentialsByIDRepository(db, id)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := postgres.GetSubaccountCredentialsByIDRepository(db, id)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, models.UserAgent{}), nil
}

// lockAccountCredentials — SELECT ... FOR UPDATE แถวบัญชี (AUTH-42) ใช้ใน tx เท่านั้น (ไม่มีข้อมูลผู้สร้าง)
func lockAccountCredentials(tx *gorm.DB, t models.AccountType, id uint) (account, error) {
	if t == models.AccountTypeAgent {
		u, err := postgres.LockUserAgentCredentialsRepository(tx, id)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := postgres.LockSubaccountCredentialsRepository(tx, id)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, models.UserAgent{}), nil
}

// isLockedByChain — ผู้สร้าง (กรณี sub) หรือ upline คนใดถูก LOCKED (AUTH-21, AUTH-27, Phase 1 AUTH-05)
// ไม่นับสถานะของบัญชีนี้เอง
func isLockedByChain(db *gorm.DB, a account) (bool, error) {
	if a.IsSub() && a.CreatorStatus == models.AgentStatusLocked {
		return true, nil
	}
	return postgres.HasLockedAncestorRepository(db, a.AgentID)
}

func setPasscodeIfEmpty(tx *gorm.DB, a account, hash string, at time.Time) (bool, error) {
	if a.IsSub() {
		return postgres.SetSubaccountPasscodeIfEmptyRepository(tx, a.ID, hash, at)
	}
	return postgres.SetUserAgentPasscodeIfEmptyRepository(tx, a.ID, hash, at)
}

func updatePasscode(tx *gorm.DB, a account, hash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	if a.IsSub() {
		return postgres.UpdateSubaccountPasscodeRepository(tx, a.ID, hash, mustChange, tempExpiresAt, at)
	}
	return postgres.UpdateUserAgentPasscodeRepository(tx, a.ID, hash, mustChange, tempExpiresAt, at)
}

func updatePassword(tx *gorm.DB, a account, hash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	if a.IsSub() {
		return postgres.UpdateSubaccountPasswordRepository(tx, a.ID, hash, a.PasswordHash, mustChange, tempExpiresAt, at)
	}
	return postgres.UpdateUserAgentPasswordRepository(tx, a.ID, hash, a.PasswordHash, mustChange, tempExpiresAt, at)
}

func updateLastLogin(db *gorm.DB, a account, ip string, at time.Time) error {
	if a.IsSub() {
		return postgres.UpdateSubaccountLastLoginRepository(db, a.ID, ip, at)
	}
	return postgres.UpdateUserAgentLastLoginRepository(db, a.ID, ip, at)
}
