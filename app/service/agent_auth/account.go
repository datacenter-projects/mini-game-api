package agentauth

import (
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"

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
	// UplineStatus = สถานะที่เข้มที่สุดของผู้สร้าง (กรณี sub) และ upline ทั้งสาย ไม่นับตัวเอง
	// มีค่าหลังเรียก withUplineStatus เท่านั้น (AUTH-53)
	UplineStatus models.AgentStatus

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

// EffectiveStatus — สถานะที่ใช้ตัดสินสิทธิ์ = เข้มที่สุดของตัวเอง ผู้สร้าง และ upline ทั้งสาย (AUTH-53)
func (a account) EffectiveStatus() models.AgentStatus {
	return agentAuthCore.WorstStatus(a.Status, a.UplineStatus)
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
		u, err := agentAuthPostgres.GetUserAgentForLoginRepository(db, username)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := agentAuthPostgres.GetSubaccountForLoginRepository(db, username)
	if err != nil {
		return account{}, err
	}
	creator, err := agentAuthPostgres.GetUserAgentAuthByIDRepository(db, s.AgentID)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, creator), nil
}

// loadAccountAuth โหลดเฉพาะคอลัมน์ที่ middleware ใช้ — ไม่พบคืน apperr.ErrNotFound
func loadAccountAuth(db *gorm.DB, t models.AccountType, id uint) (account, error) {
	if t == models.AccountTypeAgent {
		u, err := agentAuthPostgres.GetUserAgentAuthByIDRepository(db, id)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := agentAuthPostgres.GetSubaccountAuthByIDRepository(db, id)
	if err != nil {
		return account{}, err
	}
	creator, err := agentAuthPostgres.GetUserAgentAuthByIDRepository(db, s.AgentID)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, creator), nil
}

// loadAccountCredentials อ่านรหัสผ่าน/passcode โดยไม่ lock (ไม่มีข้อมูลผู้สร้าง)
func loadAccountCredentials(db *gorm.DB, t models.AccountType, id uint) (account, error) {
	if t == models.AccountTypeAgent {
		u, err := agentAuthPostgres.GetUserAgentCredentialsByIDRepository(db, id)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := agentAuthPostgres.GetSubaccountCredentialsByIDRepository(db, id)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, models.UserAgent{}), nil
}

// lockAccountCredentials — SELECT ... FOR UPDATE แถวบัญชี (AUTH-42) ใช้ใน tx เท่านั้น (ไม่มีข้อมูลผู้สร้าง)
func lockAccountCredentials(tx *gorm.DB, t models.AccountType, id uint) (account, error) {
	if t == models.AccountTypeAgent {
		u, err := agentAuthPostgres.LockUserAgentCredentialsRepository(tx, id)
		if err != nil {
			return account{}, err
		}
		return fromUserAgent(u), nil
	}
	s, err := agentAuthPostgres.LockSubaccountCredentialsRepository(tx, id)
	if err != nil {
		return account{}, err
	}
	return fromSubaccount(s, models.UserAgent{}), nil
}

// withUplineStatus เติม UplineStatus จากผู้สร้าง (กรณี sub) และ upline ทั้งสาย (AUTH-05, AUTH-21, AUTH-27, AUTH-53)
// ใช้กับ account ที่โหลดด้วย loadAccountForLogin / loadAccountAuth (มีข้อมูลผู้สร้าง)
func withUplineStatus(db *gorm.DB, a account) (account, error) {
	statuses, err := agentAuthPostgres.ListAncestorStatusesRepository(db, a.AgentID)
	if err != nil {
		return a, err
	}
	if a.IsSub() {
		statuses = append(statuses, a.CreatorStatus)
	}
	a.UplineStatus = agentAuthCore.WorstStatus(statuses...)
	return a, nil
}

func (a account) UplineLocked() bool { return a.UplineStatus == models.AgentStatusLocked }

func setPasscodeIfEmpty(tx *gorm.DB, a account, hash string, at time.Time) (bool, error) {
	if a.IsSub() {
		return agentAuthPostgres.SetSubaccountPasscodeIfEmptyRepository(tx, a.ID, hash, at)
	}
	return agentAuthPostgres.SetUserAgentPasscodeIfEmptyRepository(tx, a.ID, hash, at)
}

func updatePasscode(tx *gorm.DB, a account, hash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	if a.IsSub() {
		return agentAuthPostgres.UpdateSubaccountPasscodeRepository(tx, a.ID, hash, mustChange, tempExpiresAt, at)
	}
	return agentAuthPostgres.UpdateUserAgentPasscodeRepository(tx, a.ID, hash, mustChange, tempExpiresAt, at)
}

func updatePassword(tx *gorm.DB, a account, hash string, mustChange bool, tempExpiresAt *time.Time, at time.Time) error {
	if a.IsSub() {
		return agentAuthPostgres.UpdateSubaccountPasswordRepository(tx, a.ID, hash, a.PasswordHash, mustChange, tempExpiresAt, at)
	}
	return agentAuthPostgres.UpdateUserAgentPasswordRepository(tx, a.ID, hash, a.PasswordHash, mustChange, tempExpiresAt, at)
}

func updateLastLogin(db *gorm.DB, a account, ip string, at time.Time) error {
	if a.IsSub() {
		return agentAuthPostgres.UpdateSubaccountLastLoginRepository(db, a.ID, ip, at)
	}
	return agentAuthPostgres.UpdateUserAgentLastLoginRepository(db, a.ID, ip, at)
}
