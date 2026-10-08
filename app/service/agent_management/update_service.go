package agentmanagement

import (
	"context"
	"sort"
	"time"

	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// การแก้บัญชี (phase 4) — spec: docs/modules/agent_management.md MGMT-09 – MGMT-31, MGMT-60
// แก้ได้เฉพาะผู้สร้างโดยตรง (MGMT-23, MGMT-30): สายล่างที่ไม่ใช่ลูกตรง = 402304 · นอกสาย = 402402

// lockAgentChild — lock แถวบัญชีฝั่ง agent ที่จะแก้ แล้วเช็คว่าเป็นลูกตรงของผู้เรียก
func lockAgentChild(tx *gorm.DB, actor agentAuthService.Actor, id uint) (models.UserAgent, error) {
	a, err := agentManagementPostgres.LockUserAgentRowRepository(tx, id)
	if err != nil {
		return a, err
	}
	if a.ID == 0 {
		return a, apperr.ErrDownlineNotFound
	}
	if a.ParentID != nil && *a.ParentID == actor.AgentID {
		return a, nil
	}
	return a, NotDirectChild(tx, actor.AgentID, id)
}

// NotDirectChild — บัญชีในสายล่างแต่ไม่ใช่ลูกตรง = 402304 · นอกสาย = 402402
func NotDirectChild(tx *gorm.DB, actorAgentID, agentID uint) error {
	ok, err := agentManagementPostgres.IsInDownlineRepository(tx, actorAgentID, agentID)
	if err != nil {
		return err
	}
	if ok || agentID == actorAgentID {
		return apperr.ErrNotDirectCreator
	}
	return apperr.ErrDownlineNotFound
}

type InfoValue struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// UpdateAgentInfoService — POST /manage/agents/update-info (MGMT-08, MGMT-09)
func UpdateAgentInfoService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.UpdateInfoRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, err := lockAgentChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		if err := CheckPhoneFree(tx, req.Phone.Value, a.ID, false); err != nil {
			return err
		}
		now := time.Now()
		if err := agentManagementPostgres.UpdateUserAgentInfoRepository(tx, a.ID, req.Name, OptionalString(req.Phone.Value), now); err != nil {
			return err
		}
		return WriteLog(ctx, tx, actor, meta, targetAgent, a.ID, a.Username, models.ChangeUpdateInfo,
			InfoValue{StringOrEmpty(a.Name), StringOrEmpty(a.Phone)}, InfoValue{req.Name, req.Phone.Value}, now)
	})
}

// CheckPhoneFree — เบอร์ห้ามซ้ำกับบัญชีอื่นในตารางเดียวกัน (MGMT-08) · lock key เดียวกับเส้นสร้าง
func CheckPhoneFree(tx *gorm.DB, phone string, selfID uint, member bool) error {
	if phone == "" {
		return nil
	}
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "phone:"+phone); err != nil {
		return err
	}
	var taken bool
	var err error
	if member {
		taken, err = memberManagementPostgres.UserMemberPhoneTakenByOtherRepository(tx, phone, selfID)
	} else {
		taken, err = agentManagementPostgres.AgentPhoneTakenByOtherRepository(tx, phone, selfID)
	}
	if err != nil {
		return err
	}
	if taken {
		return apperr.ErrPhoneTaken
	}
	return nil
}

type StatusValue struct {
	Status models.AgentStatus `json:"status"`
}

// UpdateAgentStatusService — POST /manage/agents/update-status (MGMT-30, MGMT-31)
// แก้สถานะที่ตั้งกับบัญชีนั้นเอง · สายล่างได้ผลผ่านสถานะที่ใช้งานจริง ไม่แก้แถวของชั้นล่าง
func UpdateAgentStatusService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.UpdateStatusRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a, err := lockAgentChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		now := time.Now()
		next := models.AgentStatus(req.Status)
		if err := agentManagementPostgres.UpdateUserAgentStatusRepository(tx, a.ID, next, now); err != nil {
			return err
		}
		return WriteLog(ctx, tx, actor, meta, targetAgent, a.ID, a.Username, models.ChangeUpdateStatus,
			StatusValue{a.Status}, StatusValue{next}, now)
	})
}

// UpdateChildPTService — POST /manage/agents/update-pt (MGMT-18 – MGMT-25)
// lock: ผู้สร้าง (SHARE ใน loadCreator) → ลูก (UPDATE) → ลูกของลูก (SHARE) — id เรียงจากชั้นบนลงล่างเสมอ
func UpdateChildPTService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.UpdateChildPTRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child, err := lockAgentChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		c, err := LoadCreator(tx, actor)
		if err != nil {
			return err
		}
		childSettings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, []uint{child.ID}, "UPDATE")
		if err != nil {
			return err
		}
		grandIDs, err := agentManagementPostgres.ListAgentChildIDsRepository(tx, child.ID)
		if err != nil {
			return err
		}
		grandSettings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, grandIDs, "SHARE")
		if err != nil {
			return err
		}

		creatorIsMaster := c.UserType == agentManagementCore.UserTypeCompanySeamlessMaster
		now := time.Now()
		oldLog, newLog := map[string]agentManagementCore.ChildPT{}, map[string]agentManagementCore.ChildPT{}
		for _, g := range SortedGroups(req.PT) {
			v := req.PT[g].Parsed
			group := agentManagementCore.PTGroup(g)
			if err := PTError(g, agentManagementCore.ValidateChildPT(v, c.ReceivedBP[group], creatorIsMaster)); err != nil {
				return err
			}
			cur, ok := groupSetting(childSettings, group)
			if !ok {
				continue // บัญชีไม่มีแถวของกลุ่มนี้ (ข้อมูลก่อน module ②) — ไม่มีอะไรให้แก้
			}
			var grand []int
			for _, s := range grandSettings {
				if gg, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && gg == group {
					grand = append(grand, s.PTFromParentBP)
				}
			}
			if min := agentManagementCore.MinPTFromParent(cur.PTBP, grand); v.PTFromParentBP < min { // MGMT-24
				p, _ := utils.Percent(min).MarshalJSON()
				return apperr.ErrPTBelowChildUsage.WithMessage(
					"pt."+g+".pt_from_parent ต่ำกว่าที่ลูกใช้อยู่ ตั้งได้ต่ำสุด "+string(p),
					"pt."+g+".pt_from_parent is lower than what the child uses, minimum is "+string(p))
			}
			if err := agentManagementPostgres.UpdateChildPTRepository(tx, child.ID, GameCodes(group), models.AgentGameSetting{
				PTFromParentBP: v.PTFromParentBP, ForceBP: v.ForceBP, RemainBP: v.RemainBP, CommissionBP: v.CommissionBP,
				Status: *req.PT[g].Status}, actor.Username, now); err != nil {
				return err
			}
			oldLog[g] = agentManagementCore.ChildPT{PTFromParentBP: cur.PTFromParentBP, ForceBP: cur.ForceBP, RemainBP: cur.RemainBP, CommissionBP: cur.CommissionBP}
			newLog[g] = v
		}
		return WriteLog(ctx, tx, actor, meta, targetAgent, child.ID, child.Username, models.ChangeUpdatePT, oldLog, newLog, now)
	})
}

// UpdateOwnHoldService — POST /manage/agents/update-hold (MGMT-19, MGMT-22)
// ค่าถือจาก Member ใต้ตัวเอง · ไม่เกินค่าที่ได้รับ · Company Seamless Master ล็อก 0
func UpdateOwnHoldService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.UpdateOwnPTRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		me, err := agentManagementPostgres.GetAgentProfileRepository(tx, actor.AgentID)
		if err != nil {
			return err
		}
		settings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, []uint{me.ID}, "UPDATE")
		if err != nil {
			return err
		}
		isMaster := agentManagementCore.UserTypeOf(me.Role, me.AgentType) == agentManagementCore.UserTypeCompanySeamlessMaster
		now := time.Now()
		oldLog, newLog := map[string]int{}, map[string]int{}
		for _, g := range SortedGroups(req.PT) {
			group := agentManagementCore.PTGroup(g)
			cur, ok := groupSetting(settings, group)
			if !ok {
				return PTError(g, agentManagementCore.PTExceedsReceived) // ไม่มีค่าที่ได้รับในกลุ่มนี้
			}
			bp := req.PT[g].PTBP
			if err := PTError(g, agentManagementCore.ValidateOwnPT(bp, cur.PTFromParentBP, isMaster)); err != nil {
				return err
			}
			if err := agentManagementPostgres.UpdateOwnPTRepository(tx, me.ID, GameCodes(group), bp, actor.Username, now); err != nil {
				return err
			}
			oldLog[g], newLog[g] = cur.PTBP, bp
		}
		return WriteLog(ctx, tx, actor, meta, targetAgent, me.ID, me.Username, models.ChangeUpdatePT,
			map[string]any{"pt_bp": oldLog}, map[string]any{"pt_bp": newLog}, now)
	})
}

// groupSetting — แถวของเกมแรกในกลุ่ม (ค่าในกลุ่มเท่ากันทุกเกม — MGMT-16)
func groupSetting(settings []models.AgentGameSetting, group agentManagementCore.PTGroup) (models.AgentGameSetting, bool) {
	for _, s := range settings {
		if g, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && g == group {
			return s, true
		}
	}
	return models.AgentGameSetting{}, false
}

func GameCodes(group agentManagementCore.PTGroup) []string {
	games := agentManagementCore.GamesOf(group)
	out := make([]string, len(games))
	for i, g := range games {
		out[i] = g.GameCode
	}
	return out
}

func SortedGroups[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// WriteLog — 1 แถว account_change_logs ใน tx เดียวกับการแก้ (MGMT-60)
func WriteLog(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta, targetType string,
	targetID uint, targetUsername string, action models.AccountChangeAction, oldValue, newValue any, now time.Time) error {
	row, err := ChangeLog(actor, meta, logger.RequestID(ctx), targetType, targetID, targetUsername, action, oldValue, newValue, now)
	if err != nil {
		return err
	}
	return agentManagementPostgres.CreateAccountChangeLogsRepository(tx, []models.AccountChangeLog{row})
}

// UpdateGamesService — POST /manage/agents/update-games (MGMT-20)
// เปิด / ปิดเกมรายบัญชีให้ลูกตรง · ไม่ส่งต่อลงสายล่าง — ตอนเล่นเช็คทั้งสาย (core.IsGameOpen)
func UpdateGamesService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.UpdateGamesRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child, err := lockAgentChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		settings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, []uint{child.ID}, "UPDATE")
		if err != nil {
			return err
		}
		cur := make(map[string]bool, len(settings))
		for _, s := range settings {
			cur[s.GameCode] = s.StatusGame
		}
		codes := make([]string, 0, len(req.StatusGame))
		for code := range req.StatusGame {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		var on, off []string
		oldLog, newLog := map[string]bool{}, map[string]bool{}
		for _, code := range codes {
			was, ok := cur[code]
			if !ok {
				continue // บัญชีไม่มีแถวของเกมนี้ (ข้อมูลก่อน module ②) — ไม่มีอะไรให้แก้
			}
			next := req.StatusGame[code]
			if next {
				on = append(on, code)
			} else {
				off = append(off, code)
			}
			oldLog[code], newLog[code] = was, next
		}
		if err := agentManagementPostgres.UpdateStatusGameRepository(tx, child.ID, on, true); err != nil {
			return err
		}
		if err := agentManagementPostgres.UpdateStatusGameRepository(tx, child.ID, off, false); err != nil {
			return err
		}
		return WriteLog(ctx, tx, actor, meta, targetAgent, child.ID, child.Username, models.ChangeUpdateGames,
			map[string]any{"status_game": oldLog}, map[string]any{"status_game": newLog}, time.Now())
	})
}
