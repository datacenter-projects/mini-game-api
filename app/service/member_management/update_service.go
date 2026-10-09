package membermanagement

import (
	"context"
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

// การแก้ Member (phase 4) — spec: docs/modules/agent_management.md MGMT-09A, MGMT-21, MGMT-30, MGMT-60
// แก้ได้เฉพาะผู้สร้างโดยตรง (MGMT-23, MGMT-30): สายล่างที่ไม่ใช่ลูกตรง = 402304 · นอกสาย = 402402

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

// UpdateMemberInfoService — POST /manage/members/update-info (MGMT-09A)
func UpdateMemberInfoService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.UpdateInfoRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMemberChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		if err := agentManagementService.CheckPhoneFree(tx, req.Phone.Value, m.ID, true); err != nil {
			return err
		}
		now := time.Now()
		if err := memberManagementPostgres.UpdateUserMemberInfoRepository(tx, m.ID, req.Name, agentManagementService.OptionalString(req.Phone.Value), now); err != nil {
			return err
		}
		return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdateInfo,
			agentManagementService.InfoValue{Name: m.Name, Phone: agentManagementService.StringOrEmpty(m.Phone)},
			agentManagementService.InfoValue{Name: req.Name, Phone: req.Phone.Value}, now)
	})
}

// UpdateMemberStatusService — POST /manage/members/update-status (MGMT-30)
func UpdateMemberStatusService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.UpdateStatusRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMemberChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		now := time.Now()
		next := models.AgentStatus(req.Status)
		if err := memberManagementPostgres.UpdateUserMemberStatusRepository(tx, m.ID, next, now); err != nil {
			return err
		}
		return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdateStatus,
			agentManagementService.StatusValue{Status: m.Status}, agentManagementService.StatusValue{Status: next}, now)
	})
}

// UpdateMemberPTService — POST /manage/members/update-pt (MGMT-21 แก้ 2026-10-09)
// lock: แถว Member → ค่าที่ผู้สร้างได้รับ (agent_game_settings FOR SHARE) → แถวของ Member (FOR UPDATE)
// ลำดับ agent_game_settings ก่อน user_member_game_settings เหมือน agents/update-pt — กัน deadlock
func UpdateMemberPTService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.UpdateMemberPTRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m, err := lockMemberChild(tx, actor, req.ID) // ผ่าน = ผู้เรียกเป็นผู้สร้าง (m.AgentID)
		if err != nil {
			return err
		}
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
		now := time.Now()
		oldLog, newLog := map[string]memberPTValue{}, map[string]memberPTValue{}
		for _, g := range agentManagementService.SortedGroups(req.PT) {
			v := req.PT[g]
			group := agentManagementCore.PTGroup(g)
			if err := agentManagementService.MemberPTError(g, agentManagementCore.ValidateMemberPT(v.PT, received[group]), received[group]); err != nil {
				return err
			}
			if err := agentManagementService.PTError(g, agentManagementCore.ValidateMemberCommission(v.Commission)); err != nil {
				return err
			}
			for _, s := range settings {
				if gg, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && gg == group {
					oldLog[g] = memberPTValue{PT: s.PT, Remain: s.Remain, Commission: s.Commission}
				}
			}
			remain := agentManagementCore.MemberRemain(received[group], v.PT)
			if err := memberManagementPostgres.UpdateUserMemberPTRepository(tx, m.ID, agentManagementService.GameCodes(group),
				v.PT, remain, v.Commission, actor.Username, now); err != nil {
				return err
			}
			newLog[g] = memberPTValue{PT: v.PT, Remain: remain, Commission: v.Commission}
		}
		return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdatePT,
			oldLog, newLog, now)
	})
}
