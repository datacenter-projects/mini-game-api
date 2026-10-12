package membermanagement

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentManagementCore "app/app/core/agent_management"
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/platform/database"

	"gorm.io/gorm"
)

// การแก้ Member — spec: docs/modules/member_management.md M3 · M4 (MGMT-09A, MGMT-21, MGMT-30, MGMT-32M, MGMT-32N, MGMT-60)
// แก้ได้เฉพาะผู้สร้างโดยตรง (MGMT-30 · MGMT-32M): สายล่างที่ไม่ใช่ผู้สร้าง = 402304 · นอกสาย = 402402

// lockMemberChild — lock แถว Member ที่จะแก้ แล้วเช็คว่าผู้เรียกเป็นผู้สร้าง
func lockMemberChild(tx *gorm.DB, actor agentAuthService.Actor, id uint) (models.UserMember, error) {
	m, err := memberManagementPostgres.LockUserMemberRowRepository(tx, id)
	if err != nil {
		return m, err
	}
	if m.ID == 0 {
		return m, apperr.ErrDownlineNotFound
	}
	if m.AgentID == actor.AgentID {
		return m, nil
	}
	return m, agentManagementService.NotDirectChild(tx, actor.AgentID, m.AgentID)
}

// lockMemberPhoneFree — lock เบอร์ (คู่) ภายใต้ผู้สร้าง แล้วเช็คว่า Member อื่นของผู้สร้างคนเดียวกันใช้แล้วไหม (MGMT-08 · lead MQ1)
// ไม่กรอก = ไม่เช็ค · คนละผู้สร้าง / ซ้ำกับ agent ได้ · selfID 0 = Member ใหม่
func lockMemberPhoneFree(tx *gorm.DB, agentID uint, code, phone string, selfID uint) error {
	if phone == "" {
		return nil
	}
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, fmt.Sprintf("member_phone:%d:%s-%s", agentID, code, phone)); err != nil {
		return err
	}
	taken, err := memberManagementPostgres.UserMemberPhoneTakenByOtherRepository(tx, agentID, code, phone, selfID)
	if err != nil {
		return err
	}
	if taken {
		return apperr.ErrPhoneTaken
	}
	return nil
}

// checkMemberPT — กฎที่ต้องดู DB ของ 1 กลุ่ม: pt ไม่เกินค่าที่ผู้สร้างได้รับ (402305) → commission_percent (402309) · ไม่ลงขั้น = 422 (lead MQ3)
func checkMemberPT(group string, v memberManagementDto.MemberPTRequest, received float64) error {
	if err := agentManagementService.MemberPTError(group, agentManagementCore.ValidateMemberPT(v.PT, received), received); err != nil {
		return err
	}
	return agentManagementService.PTError(group, agentManagementCore.ValidateMemberCommission(v.Commission))
}

// UpdateMemberDetailService — POST /manage/members/detail/update (M3 · MGMT-32M · lead M3-1 / MN2)
// ผู้สร้างโดยตรงเท่านั้น · tx เดียว: section ไหนผิดไม่บันทึกเลย · ทำตามลำดับ info → pt · log แยกแถวต่อ section ที่เปลี่ยนจริง
func UpdateMemberDetailService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.MemberDetailUpdateRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMemberChild(tx, actor, req.ID) // ผ่าน = ผู้เรียกเป็นผู้สร้าง (m.AgentID)
		if err != nil {
			return err
		}
		now := time.Now()
		if req.Info != nil {
			if err := applyMemberInfo(ctx, tx, actor, meta, m, *req.Info, now); err != nil {
				return err
			}
		}
		if req.PT != nil {
			return applyMemberPT(ctx, tx, actor, meta, m, req.PT, now)
		}
		return nil
	})
}

// applyMemberInfo — section info (MGMT-08, MGMT-09A) · field ที่ไม่ส่งคงค่าเดิม · ไม่เปลี่ยน = ไม่เขียน log
func applyMemberInfo(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	m models.UserMember, in memberManagementDto.MemberInfoSection, now time.Time) error {
	old := agentManagementService.AgentInfoValue{Name: m.Name, PhoneCountryCode: agentManagementService.StringOrEmpty(m.PhoneCountryCode),
		Phone: agentManagementService.StringOrEmpty(m.Phone)}
	next := old
	if in.Name != nil {
		next.Name = *in.Name
	}
	if in.Phone.Present { // DTO บังคับส่งคู่กับ phone_country_code
		next.PhoneCountryCode, next.Phone = in.PhoneCountryCode.Value, in.Phone.Value
	}
	if next == old {
		return nil
	}
	if next.PhoneCountryCode != old.PhoneCountryCode || next.Phone != old.Phone {
		if err := lockMemberPhoneFree(tx, m.AgentID, next.PhoneCountryCode, next.Phone, m.ID); err != nil {
			return err
		}
	}
	err := memberManagementPostgres.UpdateUserMemberInfoRepository(tx, m.ID, next.Name,
		agentManagementService.OptionalString(next.PhoneCountryCode), agentManagementService.OptionalString(next.Phone), now)
	if errors.Is(err, apperr.ErrConflict) { // unique เดียวของคอลัมน์ที่แก้คือเบอร์ (หลุด advisory lock — เช่น แถวที่ยังไม่ commit ตอนเช็ค)
		return apperr.ErrPhoneTaken
	}
	if err != nil {
		return err
	}
	return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdateInfo, old, next, now)
}

// applyMemberPT — section pt (MGMT-21 · MGMT-32M / MGMT-32N) · กลุ่มที่ค่าเหมือนเดิมไม่เขียน · ไม่มีกลุ่มเปลี่ยน = ไม่เขียน log
// lock: แถว Member (lockMemberChild) → ค่าที่ผู้สร้างได้รับ (agent_game_settings FOR SHARE) → แถวค่าตั้งของ Member (FOR UPDATE)
// ลำดับ agent_game_settings ก่อน user_member_game_settings เหมือน agents/detail/update — กัน deadlock
func applyMemberPT(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	m models.UserMember, req map[string]memberManagementDto.MemberPTRequest, now time.Time) error {
	creatorSettings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, []uint{m.AgentID}, "SHARE")
	if err != nil {
		return err
	}
	received := map[agentManagementCore.PTGroup]float64{}
	for _, s := range creatorSettings { // ค่าของเกมในกลุ่มเท่ากันเสมอ (MGMT-16)
		if g, ok := agentManagementCore.GroupOfGame(s.GameCode); ok {
			received[g] = s.PTFromParent
		}
	}
	settings, err := memberManagementPostgres.LockUserMemberGameSettingsRepository(tx, m.ID)
	if err != nil {
		return err
	}
	oldLog, newLog := map[string]memberPTValue{}, map[string]memberPTValue{}
	for _, g := range agentManagementService.SortedGroups(req) {
		v := req[g]
		group := agentManagementCore.PTGroup(g)
		cur, ok := memberGroupSetting(settings, group)
		if !ok {
			continue // Member ไม่มีแถวของกลุ่มนี้ (ข้อมูลก่อน module ②) — ข้ามเงียบ ไม่เขียน log (MGMT-32N · lead MQ4)
		}
		if err := checkMemberPT(g, v, received[group]); err != nil {
			return err
		}
		remain := agentManagementCore.MemberRemain(received[group], v.PT)
		if cur.PT == v.PT && cur.Commission == v.Commission && cur.Remain == remain {
			continue // ค่าเหมือนเดิม (MGMT-32M)
		}
		if err := memberManagementPostgres.UpdateUserMemberPTRepository(tx, m.ID, agentManagementService.GameCodes(group),
			v.PT, remain, v.Commission, actor.Username, now); err != nil {
			return err
		}
		oldLog[g] = memberPTValue{PT: cur.PT, Remain: cur.Remain, Commission: cur.Commission}
		newLog[g] = memberPTValue{PT: v.PT, Remain: remain, Commission: v.Commission}
	}
	if len(newLog) == 0 {
		return nil
	}
	return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdatePT,
		oldLog, newLog, now)
}

// memberGroupSetting — แถวของเกมแรกในกลุ่ม (ค่าในกลุ่มเท่ากันทุกเกม — MGMT-16)
func memberGroupSetting(settings []models.UserMemberGameSetting, group agentManagementCore.PTGroup) (models.UserMemberGameSetting, bool) {
	for _, s := range settings {
		if g, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && g == group {
			return s, true
		}
	}
	return models.UserMemberGameSetting{}, false
}

// UpdateMemberStatusService — POST /manage/members/status/update (M4 · MGMT-30) · ส่งสถานะเดิม = 200 ไม่เขียน log (lead MQ5)
func UpdateMemberStatusService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.UpdateStatusRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMemberChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		next := models.AgentStatus(req.Status)
		if next == m.Status {
			return nil
		}
		now := time.Now()
		if err := memberManagementPostgres.UpdateUserMemberStatusRepository(tx, m.ID, next, now); err != nil {
			return err
		}
		return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdateStatus,
			agentManagementService.StatusValue{Status: m.Status}, agentManagementService.StatusValue{Status: next}, now)
	})
}
