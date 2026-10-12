// Package membermanagement คือ business logic ของเส้น /manage/members/* — spec: docs/modules/member_management.md
// helper ที่ใช้ร่วมกับ agent (ผู้สร้าง · ยอดตั้งต้น · log) อยู่ใน service/agent_management · import ทางเดียว
package membermanagement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	agentManagementCore "app/app/core/agent_management"
	memberManagementCore "app/app/core/member_management"
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// memberPTValue — ค่า PT ของ Member 1 กลุ่มใน account_change_logs (MGMT-60)
type memberPTValue struct {
	PT         float64 `json:"pt"`
	Remain     float64 `json:"remain"`
	Commission float64 `json:"commission"`
}

// CreateMemberService — POST /api/v1/bo/pr/manage/members/create (M1 · MGMT-02, MGMT-05 – MGMT-15A, MGMT-21, MGMT-60)
// pt ของ Member ไม่เกินค่าที่ผู้สร้างได้รับ (lock แถวผู้สร้าง FOR SHARE ใน LoadCreator) · remain = ค่าที่ผู้สร้างได้รับ − pt
func CreateMemberService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.CreateMemberRequest, meta agentAuthService.RequestMeta) (memberManagementDto.CreateMemberResponse, error) {
	var res memberManagementDto.CreateMemberResponse
	if len(req.BalanceAmounts) > 0 {
		if err := agentManagementService.CheckPermissionService(ctx, actor, agentManagementCore.MenuPayment, agentManagementCore.LevelEdit); err != nil {
			return res, err
		}
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return res, err
	}

	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id, replay, err := agentManagementService.FindReplay(tx, req.RequestID, actor, agentManagementService.TargetMember); err != nil || replay {
			if err != nil {
				return err
			}
			m, err := memberManagementPostgres.GetUserMemberProfileRepository(tx, id)
			if err != nil {
				return err
			}
			res = memberManagementDto.CreateMemberResponse{ID: m.ID, Username: m.Username}
			return nil
		}

		c, err := agentManagementService.LoadCreator(tx, actor)
		if err != nil {
			return err
		}
		// กฎที่ต้องดู DB ไล่ตามลำดับ field ใน body (รูปแบบเช็คครบแล้วใน DTO — หัวข้อ 2.1 · lead U3 / MQ2):
		// ประเภทผู้สร้าง → username → เบอร์ → balance (Seamless → ยอดไม่พอ) → pt → commission_percent → บัญชีธนาคาร
		if !agentManagementCore.CanCreateMember(c.UserType) {
			return apperr.ErrCannotCreateType
		}
		if err := agentManagementService.LockUsernameFree(tx, req.Username); err != nil {
			return err
		}
		if err := lockMemberPhoneFree(tx, c.Agent.ID, req.PhoneCountryCode, req.Phone, 0); err != nil {
			return err
		}
		currencies, _ := agentManagementCore.ResolveCurrencies(c.UserType, agentManagementCore.UserTypeMember, nil, c.Currencies)
		if len(currencies) != 1 { // ผู้สร้าง Member มี 1 สกุลเสมอ (MGMT-11 – MGMT-13) — ไม่ใช่ = ข้อมูลผู้สร้างไม่ครบ
			return apperr.ErrInternal.Wrap(fmt.Errorf("creator %d has %d currencies, want 1", c.Agent.ID, len(currencies)))
		}
		if err := agentManagementService.CheckInitialBalance(req.BalanceAmounts, agentManagementCore.IsSeamless(c.CompanyType), currencies); err != nil {
			return err
		}
		if err := agentManagementService.CheckCreatorBalance(tx, c, req.BalanceAmounts); err != nil {
			return err
		}
		for _, g := range agentManagementService.SortedGroups(req.PT) { // ในกลุ่มไล่ตาม field: pt → commission_percent
			if err := checkMemberPT(g, req.PT[g], c.Received[agentManagementCore.PTGroup(g)]); err != nil {
				return err
			}
		}
		if err := lockMemberPaymentAccountFree(tx, c.Agent.ID, req.Account, 0); err != nil { // บัญชีซ้ำ (MGMT-64) ต่อจาก commission_percent
			return err
		}

		now := time.Now()
		cnf, err := json.Marshal(agentManagementCore.ChildChain(c.Chain, c.Agent.ID, c.Agent.Role)) // สายชั้นบน = สายของผู้สร้าง + ผู้สร้าง
		if err != nil {
			return err
		}
		paymentType := memberManagementCore.StoredPaymentType(req.Payment, agentManagementCore.IsSeamless(c.CompanyType)) // Seamless = CASH (MGMT-63)
		account, err := encodePaymentAccount(req.Account)
		if err != nil {
			return err
		}
		m := models.UserMember{AgentID: c.Agent.ID, Username: req.Username, PasswordHash: hash, Name: req.Name,
			PhoneCountryCode: agentManagementService.OptionalString(req.PhoneCountryCode), Phone: agentManagementService.OptionalString(req.Phone),
			Currency: currencies[0], Status: models.AgentStatusActive, Cnf: string(cnf),
			PaymentType: paymentType, PaymentAccount: account, WinLimitPerDay: req.WinLimitPerDay, CreatedAt: now, UpdatedAt: now}
		if err := createMemberRow(tx, &m, req.Account); err != nil {
			return err
		}

		var settings []models.UserMemberGameSetting
		ptLog := map[string]memberPTValue{}
		for _, g := range agentManagementCore.Groups() {
			v := req.PT[string(g)]
			remain := agentManagementCore.MemberRemain(c.Received[g], *v.OwnPT)
			ptLog[string(g)] = memberPTValue{PT: *v.OwnPT, Remain: remain, Commission: *v.CommissionPercent}
			for _, game := range agentManagementCore.GamesOf(g) {
				settings = append(settings, models.UserMemberGameSetting{UserMemberID: m.ID, GameCode: game.GameCode, ParentID: c.Agent.ID, Category: game.Category,
					PT: *v.OwnPT, Remain: remain, Commission: *v.CommissionPercent,
					CreatedBy: actor.Username, CreatedAt: now, UpdatedBy: actor.Username, UpdatedAt: now})
			}
		}
		if err := memberManagementPostgres.CreateUserMemberGameSettingsRepository(tx, settings); err != nil {
			return err
		}

		if err := agentManagementService.TransferInitialBalance(tx, c, models.BalanceOwnerMember, m.ID, req.BalanceAmounts, req.RequestID, actor, now); err != nil {
			return err
		}
		if err := agentManagementPostgres.CreateCreateRequestRepository(tx, &models.CreateRequest{RequestID: req.RequestID, CreatorType: actor.AccountType,
			CreatorID: actor.AccountID(), TargetType: agentManagementService.TargetMember, TargetID: m.ID, CreatedAt: now}); err != nil {
			return err
		}

		created, err := agentManagementService.ChangeLog(actor, meta, req.RequestID, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeCreate, nil,
			map[string]any{"username": m.Username, "name": req.Name, "phone_country_code": req.PhoneCountryCode, "phone": req.Phone, "agent_id": c.Agent.ID,
				"currency": m.Currency, "pt": ptLog, "payment_type": paymentType, "payment_account": maskedPaymentAccount(req.Account), // เลขบัญชีปิดบางส่วน (P8)
				"win_limit_per_day": req.WinLimitPerDay}, now)
		if err != nil {
			return err
		}
		rows := []models.AccountChangeLog{created}
		if len(req.BalanceAmounts) > 0 {
			row, err := agentManagementService.ChangeLog(actor, meta, req.RequestID, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeInitialBalance, nil,
				map[string]any{"amounts": req.BalanceAmounts}, now)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
		if err := agentManagementPostgres.CreateAccountChangeLogsRepository(tx, rows); err != nil {
			return err
		}

		res = memberManagementDto.CreateMemberResponse{ID: m.ID, Username: m.Username}
		return nil
	})
	return res, err
}

// createMemberRow — INSERT แถว Member (MGMT-05, MGMT-08, MGMT-64)
// username / เบอร์ / บัญชีถูกเช็คใต้ advisory lock แล้ว แต่ถ้ายังชน unique (เช่น แถวที่ยังไม่ commit ตอนเช็ค) ให้ตอบ error ของ field แทน 409:
// ย้อนไป savepoint แล้วดูตามลำดับ field — เบอร์ถูกใช้ = 402403 · บัญชีถูกใช้ = 402405 · ไม่ใช่ทั้งคู่ = username (unique ที่เหลือของตาราง) 402401
func createMemberRow(tx *gorm.DB, m *models.UserMember, account models.MemberPaymentAccount) error {
	const sp = "create_member_row"
	if err := tx.SavePoint(sp).Error; err != nil {
		return err
	}
	err := memberManagementPostgres.CreateUserMemberRepository(tx, m)
	if !errors.Is(err, apperr.ErrConflict) {
		return err
	}
	if err := tx.RollbackTo(sp).Error; err != nil {
		return err
	}
	if m.Phone != nil && m.PhoneCountryCode != nil {
		taken, err := memberManagementPostgres.UserMemberPhoneTakenByOtherRepository(tx, m.AgentID, *m.PhoneCountryCode, *m.Phone, 0)
		if err != nil {
			return err
		}
		if taken {
			return apperr.ErrPhoneTaken
		}
	}
	if memberManagementCore.HasPaymentAccount(account) {
		taken, err := memberManagementPostgres.UserMemberPaymentAccountTakenByOtherRepository(tx, m.AgentID, account.BankCode, account.AccountNumber, 0)
		if err != nil {
			return err
		}
		if taken {
			return apperr.ErrPaymentAccountTaken
		}
	}
	return apperr.ErrUsernameTaken
}
