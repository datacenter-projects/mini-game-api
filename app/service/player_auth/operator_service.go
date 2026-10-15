package playerauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	agentManagementCore "app/app/core/agent_management"
	memberManagementCore "app/app/core/member_management"
	playerAuthCore "app/app/core/player_auth"
	transferDto "app/app/externals/transfer/dto/player_auth"
	"app/app/models"
	accountPostgres "app/app/repository/postgres/account"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	playerAuthPostgres "app/app/repository/postgres/player_auth"
	playerAuthRedis "app/app/repository/redis/player_auth"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// Operator คือ agent เจ้าของ Key ที่เรียก transfer/* — middleware OperatorAuthenticated set ไว้ใน Locals
type Operator struct {
	AgentID  uint
	Username string
	UserType agentManagementCore.UserType
	Gate     playerAuthCore.OperatorGate // OK / Suspended (LOCKED ไม่ผ่าน middleware)
}

// OperatorRateLimitService จำกัดจำนวนครั้งที่ IP เรียก transfer/* (PA-42) — เรียกก่อนอ่าน / ตรวจ body
func OperatorRateLimitService(ctx context.Context, ip string) error {
	count, err := playerAuthRedis.IncrOperatorIPRepository(ctx, ip)
	if err != nil {
		return err
	}
	if count > configs.Cfg.Auth.OperatorIPLimit {
		return apperr.ErrTooMany
	}
	return nil
}

// OperatorAuthenticateService ตรวจ agent_username + key + IP whitelist + สถานะ + ประเภทผู้สร้าง ของ transfer/* ทุกเส้น (PA-10 – PA-14)
// ลำดับ: Key (407206) → IP (407305) → สถานะ LOCKED ของ agent / สายบน (407302) → ประเภท (407306)
// agent_username ไม่มี / ไม่มี Key / Key ไม่ตรง ตอบ 407206 เหมือนกัน ไม่บอกว่าผิดที่ไหน (PA-11) · ห้าม log key (PA-12)
func OperatorAuthenticateService(ctx context.Context, creds transferDto.OperatorCreds, ip string) (Operator, error) {
	db := database.DBConn.WithContext(ctx)

	a, err := playerAuthPostgres.GetOperatorAgentRepository(db, playerAuthCore.NormalizeUsername(creds.AgentUsername))
	if errors.Is(err, apperr.ErrNotFound) {
		return Operator{}, apperr.ErrOperatorInvalidKey
	}
	if err != nil {
		return Operator{}, err
	}
	cred, err := accountPostgres.GetAPICredentialRepository(db, a.ID)
	if errors.Is(err, apperr.ErrNotFound) {
		return Operator{}, apperr.ErrOperatorInvalidKey
	}
	if err != nil {
		return Operator{}, err
	}
	if !playerAuthCore.KeyEqual(cred.APIKey, creds.Key) {
		return Operator{}, apperr.ErrOperatorInvalidKey
	}

	cidrs, err := accountPostgres.ListAPIAllowedIPsRepository(db, a.ID)
	if err != nil {
		return Operator{}, err
	}
	if !playerAuthCore.IPAllowed(ip, cidrs) { // whitelist ว่าง = ปฏิเสธทุก IP (ACC-07)
		return Operator{}, apperr.ErrOperatorIPNotAllowed
	}

	upline, err := agentAuthPostgres.ListAncestorStatusesRepository(db, a.ID)
	if err != nil {
		return Operator{}, err
	}
	gate := playerAuthCore.OperatorGateOf(a.Status, upline...)
	if gate == playerAuthCore.OperatorLocked { // PA-13: agent หรือสายบน LOCKED → ปฏิเสธทุก transfer/*
		return Operator{}, apperr.ErrPlayerUplineLocked
	}
	userType := agentManagementCore.UserTypeOf(a.Role, a.AgentType)
	if !playerAuthCore.CanUseTransfer(userType) { // PA-14
		return Operator{}, apperr.ErrOperatorNotAllowed
	}
	return Operator{AgentID: a.ID, Username: a.Username, UserType: userType, Gate: gate}, nil
}

// TransferLoginService — L2 POST /api/v1/transfer/login (PA-07, PA-08A, PA-08B, PA-13, PA-22, LQ-c)
// ไม่มีผู้เล่น = สร้างให้อัตโนมัติ · มีแล้ว = login เลย · ส่ง token + link lobby (token ใน URL)
func TransferLoginService(ctx context.Context, op Operator, req transferDto.TransferLoginRequest, meta RequestMeta) (transferDto.TransferLoginResponse, error) {
	var res transferDto.TransferLoginResponse
	lobbyURL := configs.Cfg.PlayerLobbyURL
	if lobbyURL == "" { // env ไม่ถูกตั้ง = ระบบยังไม่พร้อมออก link
		return res, apperr.ErrUnavailable
	}
	lang, _ := playerAuthCore.NormalizeLanguage(req.Language) // Validate() ผ่านแล้ว
	// N5: ตรวจว่าสร้าง link ได้ **ก่อน**สร้างผู้เล่น / ออก session (เตะ session เดิม) — ค่าผิดรูปแบบต้องไม่ทำให้ผู้เล่นโดนเตะโดยไม่ได้ link
	// (ปกติ configs.Load กันไว้แล้ว ที่นี่กันซ้ำเผื่อค่าถูกแก้หลัง boot)
	if _, ok := playerAuthCore.LobbyLink(lobbyURL, "check", lang); !ok {
		return res, apperr.ErrUnavailable.Wrap(fmt.Errorf("PLAYER_LOBBY_URL is not a valid absolute URL"))
	}
	player := playerAuthCore.NormalizeUsername(req.PlayerUsername)
	username := playerAuthCore.ComposeUsername(op.Username, player)

	db := database.DBConn.WithContext(ctx)
	m, err := playerAuthPostgres.GetMemberByUsernameRepository(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		if op.Gate == playerAuthCore.OperatorSuspended { // PA-13: agent SUSPENDED สร้างผู้เล่นใหม่ไม่ได้
			return res, apperr.ErrPlayerCreatorSuspended
		}
		m, err = createPlayer(ctx, op, meta.IP, username, player, req.Currency, "")
		if errors.Is(err, apperr.ErrPlayerExists) { // PA-08B: แพ้การสร้างพร้อมกัน → อ่านผู้เล่นที่สร้างแล้วทำต่อ ไม่ error
			m, err = playerAuthPostgres.GetMemberByUsernameRepository(db, username)
		}
	}
	if err != nil {
		return res, err
	}
	if m.AgentID != op.AgentID { // username มี prefix ของ agent จึงไม่ควรเกิด — กันไว้ไม่ให้ใช้ Key ของคนอื่นเข้าผู้เล่นของ agent อื่น
		return res, apperr.ErrOperatorNotAllowed
	}
	if !playerAuthCore.CurrencyMatchesPlayer(m.Currency, req.Currency) { // PA-08A
		return res, apperr.ErrValidation.WithMessage("สกุลเงินไม่ตรงกับผู้เล่น", "currency does not match the player")
	}
	st, err := loadMemberState(db, m)
	if err != nil {
		return res, err
	}
	if err := st.gate(); err != nil { // ผู้เล่นเองถูกล็อก (agent / สายบนถูกล็อกถูกกันที่ middleware แล้ว)
		return res, err
	}

	s, err := issueSession(ctx, db, m.ID, meta.IP) // PA-22: เตะ session เดิม
	if err != nil {
		return res, err
	}
	link, ok := playerAuthCore.LobbyLink(lobbyURL, s.Token, lang)
	if !ok {
		return res, apperr.ErrUnavailable.Wrap(fmt.Errorf("PLAYER_LOBBY_URL is not a valid absolute URL"))
	}
	return transferDto.TransferLoginResponse{Token: s.Token, ExpiresAt: s.ExpiresAt, Link: link}, nil
}

// TransferRegisterService — TR POST /api/v1/transfer/register (PA-08, PA-08A, PA-13)
// สร้างผู้เล่นอย่างเดียว ไม่ออก token · มีแล้ว = 407401 · password (ไม่บังคับ) = ผู้เล่น login ตรงที่ PA1 ได้
func TransferRegisterService(ctx context.Context, op Operator, req transferDto.TransferRegisterRequest, meta RequestMeta) (transferDto.TransferRegisterResponse, error) {
	var res transferDto.TransferRegisterResponse
	player := playerAuthCore.NormalizeUsername(req.PlayerUsername)
	username := playerAuthCore.ComposeUsername(op.Username, player)

	db := database.DBConn.WithContext(ctx)
	if _, err := playerAuthPostgres.GetMemberByUsernameRepository(db, username); err == nil {
		return res, apperr.ErrPlayerExists
	} else if !errors.Is(err, apperr.ErrNotFound) {
		return res, err
	}
	if op.Gate == playerAuthCore.OperatorSuspended { // PA-13
		return res, apperr.ErrPlayerCreatorSuspended
	}
	var hash string
	if req.Password != "" {
		h, err := utils.HashPassword(req.Password)
		if err != nil {
			return res, err
		}
		hash = h
	}
	m, err := createPlayer(ctx, op, meta.IP, username, player, req.Currency, hash)
	if err != nil {
		return res, err
	}
	return transferDto.TransferRegisterResponse{Username: m.Username, Currency: m.Currency}, nil
}

// createPlayer สร้างผู้เล่นให้ operator ใน tx เดียว (PA-07 / PA-08 / PA-09)
// ค่าตั้งต้น: name = player_username · credit 0 · CASH · pt = ค่าเต็มที่ agent ได้รับ (remain 0) · commission 0 · win limit ตาม MGMT-65
// ชน unique ของ username (สร้างพร้อมกัน / มีแล้ว) → ErrPlayerExists
func createPlayer(ctx context.Context, op Operator, ip, username, name, requestedCurrency, passwordHash string) (models.UserMember, error) {
	var m models.UserMember
	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		c, err := agentManagementService.LoadCreatorByAgentID(tx, op.AgentID)
		if err != nil {
			return err
		}
		currency, v := playerAuthCore.ResolveCurrency(c.Currencies, requestedCurrency)
		switch v {
		case playerAuthCore.CurrencyRequired:
			return apperr.ErrValidation.WithMessage("ต้องส่ง currency (agent นี้มีหลายสกุล)", "currency is required (this agent has several currencies)")
		case playerAuthCore.CurrencyNotAgents:
			return apperr.ErrValidation.WithMessage("currency ไม่ใช่สกุลของ agent", "currency is not a currency of this agent")
		case playerAuthCore.CurrencyNoneConfig:
			return apperr.ErrInternal.Wrap(fmt.Errorf("agent %d has no currency", c.Agent.ID))
		}
		cnf, err := json.Marshal(agentManagementCore.ChildChain(c.Chain, c.Agent.ID, c.Agent.Role)) // สายชั้นบน = สายของผู้สร้าง + ผู้สร้าง (MGMT-61)
		if err != nil {
			return err
		}
		now := time.Now()
		m = models.UserMember{AgentID: c.Agent.ID, Username: username, PasswordHash: passwordHash, Name: name, Currency: currency,
			Status: models.AgentStatusActive, Cnf: string(cnf), PaymentType: models.PaymentTypeCash,
			WinLimitPerDay: defaultWinLimit(), CreatedAt: now, UpdatedAt: now}
		if err := memberManagementPostgres.CreateUserMemberRepository(tx, &m); err != nil {
			if errors.Is(err, apperr.ErrConflict) {
				return apperr.ErrPlayerExists
			}
			return err
		}
		var settings []models.UserMemberGameSetting
		ptLog := map[string]playerPTValue{}
		for _, g := range agentManagementCore.Groups() {
			received := c.Received[g]
			pt := received // agent ถือสู้เองทั้งหมด
			remain := agentManagementCore.MemberRemain(received, pt)
			ptLog[string(g)] = playerPTValue{PT: pt, Remain: remain, Commission: 0}
			for _, game := range agentManagementCore.GamesOf(g) {
				settings = append(settings, models.UserMemberGameSetting{UserMemberID: m.ID, GameCode: game.GameCode, ParentID: c.Agent.ID,
					Category: game.Category, PT: pt, Remain: remain, Commission: 0,
					CreatedBy: op.Username, CreatedAt: now, UpdatedBy: op.Username, UpdatedAt: now})
			}
		}
		if err := memberManagementPostgres.CreateUserMemberGameSettingsRepository(tx, settings); err != nil {
			return err
		}

		// S2 (lead รอบ 6): เขียน account_change_logs CREATE เหมือน members/create — ผู้ทำ = agent เจ้าของ key · ip = IP ของ operator · value ห้ามมีรหัสผ่าน
		actor := agentAuthService.Actor{AccountType: models.AccountTypeAgent, AgentID: op.AgentID, Username: op.Username}
		row, err := agentManagementService.ChangeLog(actor, agentAuthService.RequestMeta{IP: ip}, "", agentManagementService.TargetMember, m.ID, m.Username,
			models.ChangeCreate, nil, map[string]any{"username": m.Username, "name": m.Name, "agent_id": c.Agent.ID, "currency": m.Currency, "pt": ptLog,
				"payment_type": m.PaymentType, "win_limit_per_day": m.WinLimitPerDay, "has_password": passwordHash != "", "source": "transfer_api"}, now)
		if err != nil {
			return err
		}
		return agentManagementPostgres.CreateAccountChangeLogsRepository(tx, []models.AccountChangeLog{row})
	})
	return m, err
}

// playerPTValue — ค่า PT ของผู้เล่น 1 กลุ่มใน account_change_logs (รูปเดียวกับ members/create — MGMT-60)
type playerPTValue struct {
	PT         float64 `json:"pt"`
	Remain     float64 `json:"remain"`
	Commission float64 `json:"commission"`
}

// defaultWinLimit — ค่าตั้งต้น win_limit_per_day ของผู้เล่นที่ระบบสร้าง (PA-09 · lead R4-2 + member-payment รอบ 2.1)
// หน่วยบาท (THB) เสมอ ไม่มีสกุลเงิน — ไม่ต้องแปลงสกุล / ไม่ใช้ currency_rate (การแปลงอยู่ที่ module เดิมพัน) · เท่ากับที่ members/create ให้เมื่อไม่ส่งค่า
// ห้ามพึ่งค่า DEFAULT ของคอลัมน์ใน DB — ส่งค่านี้ชัดเจนตอน insert เสมอ
func defaultWinLimit() float64 {
	return float64(memberManagementCore.DefaultWinLimitPerDay)
}
