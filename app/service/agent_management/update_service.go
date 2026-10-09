package agentmanagement

import (
	"context"
	"errors"
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

// InfoValue — ค่าใน log ของ Member (member_management · เบอร์ field เดียว)
type InfoValue struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// AgentInfoValue — ค่าใน log ของ section info ฝั่ง agent (MGMT-08 · เบอร์ 2 field)
type AgentInfoValue struct {
	Name             string `json:"name"`
	PhoneCountryCode string `json:"phone_country_code"`
	Phone            string `json:"phone"`
}

// UpdateAgentDetailService — POST /manage/agents/detail/update (MGMT-32 – MGMT-34 · lead E4)
// ลูกตรงเท่านั้น · tx เดียว: section ไหนผิดไม่บันทึกเลย · ทำตามลำดับ info → pt → status_game · log แยกแถวต่อ section ที่เปลี่ยนจริง
func UpdateAgentDetailService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.AgentDetailUpdateRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		child, err := lockAgentChild(tx, actor, req.ID)
		if err != nil {
			return err
		}
		// ลูกเป็น Share Master (ผู้เรียกคือ CSM) — MGMT-19: pt ส่งได้แค่ commission · ห้ามส่ง status_game · บัญชีอื่นต้องครบ 5 ค่า (7.1 ข้อ 4)
		// เช็คก่อนแก้ section ใดๆ (เดิมเป็นรูปแบบใน DTO)
		shareMaster := isShareMaster(child.Role, child.AgentType)
		shapes := make(map[string]agentManagementDto.ChildPTRequest, len(req.PT))
		for g, v := range req.PT {
			shapes[g] = v.ChildPTRequest
		}
		if err := checkPTShape(shapes, shareMaster); err != nil {
			return err
		}
		if shareMaster && req.StatusGame != nil {
			return shareMasterFieldError("status_game")
		}
		now := time.Now()
		if req.Info != nil {
			if err := applyAgentInfo(ctx, tx, actor, meta, child, *req.Info, now); err != nil {
				return err
			}
		}
		if req.PT != nil {
			if err := applyChildPT(ctx, tx, actor, meta, child, req.PT, now); err != nil {
				return err
			}
		}
		if req.StatusGame != nil {
			return applyGames(ctx, tx, actor, meta, child, req.StatusGame, now)
		}
		return nil
	})
}

// applyAgentInfo — section info (MGMT-08, MGMT-09) · field ที่ไม่ส่งคงค่าเดิม · ไม่เปลี่ยน = ไม่เขียน log
func applyAgentInfo(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	child models.UserAgent, in agentManagementDto.AgentInfoSection, now time.Time) error {
	old := AgentInfoValue{StringOrEmpty(child.Name), StringOrEmpty(child.PhoneCountryCode), StringOrEmpty(child.Phone)}
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
		if err := checkAgentPhone(tx, next.PhoneCountryCode, next.Phone, child.ID); err != nil {
			return err
		}
	}
	if err := agentManagementPostgres.UpdateUserAgentInfoRepository(tx, child.ID, next.Name, OptionalString(next.PhoneCountryCode), OptionalString(next.Phone), now); err != nil {
		return err
	}
	return WriteLog(ctx, tx, actor, meta, targetAgent, child.ID, child.Username, models.ChangeUpdateInfo, old, next, now)
}

// CheckPhoneFree — เบอร์ของ **Member** ห้ามซ้ำกับ Member อื่น (member_management · เบอร์ field เดียว) · ฝั่ง agent ใช้ checkAgentPhone (เบอร์ 2 field)
func CheckPhoneFree(tx *gorm.DB, phone string, selfID uint, member bool) error {
	if !member {
		return apperr.ErrInternal.Wrap(errors.New("CheckPhoneFree: ฝั่ง agent ใช้ checkAgentPhone"))
	}
	if phone == "" {
		return nil
	}
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "phone:"+phone); err != nil {
		return err
	}
	taken, err := memberManagementPostgres.UserMemberPhoneTakenByOtherRepository(tx, phone, selfID)
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

// UpdateAgentStatusService — POST /manage/agents/status/update (MGMT-30, MGMT-31)
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
		if next == a.Status {
			return nil // ค่าเดิม = 200 ไม่เขียน log (MGMT-34 · lead MQ5 / V1)
		}
		if err := agentManagementPostgres.UpdateUserAgentStatusRepository(tx, a.ID, next, now); err != nil {
			return err
		}
		return WriteLog(ctx, tx, actor, meta, targetAgent, a.ID, a.Username, models.ChangeUpdateStatus,
			StatusValue{a.Status}, StatusValue{next}, now)
	})
}

// applyChildPT — section pt (MGMT-18 – MGMT-25) · กลุ่มที่ค่าเหมือนเดิมไม่เขียน · ไม่มีกลุ่มเปลี่ยน = ไม่เขียน log
// ลูกเป็น Share Master (ผู้เรียกคือ CSM): แก้ได้แค่ Commission ค่าอื่นคงตาม CSM (MGMT-19)
// ลูกเป็น CSM (ผู้เรียกคือ Superadmin): ระบบตั้งค่าของ Share Master ทุกคนตาม ใน tx เดียว (MGMT-19 · ข้อยกเว้นของ MGMT-24)
// lock: ผู้สร้าง (SHARE ใน loadCreator) → ลูก (UPDATE) → ลูกของลูก (SHARE · UPDATE เมื่อลูกเป็น CSM) → ค่าของ Member (UPDATE — R3)
func applyChildPT(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	child models.UserAgent, req map[string]agentManagementDto.UpdateChildPTGroup, now time.Time) error {
	c, err := LoadCreator(tx, actor)
	if err != nil {
		return err
	}
	childIsCSM := isCSM(child.Role, child.AgentType)
	childIsShareMaster := isShareMaster(child.Role, child.AgentType)
	childSettings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, []uint{child.ID}, "UPDATE")
	if err != nil {
		return err
	}
	grandIDs, err := agentManagementPostgres.ListAgentChildIDsRepository(tx, child.ID)
	if err != nil {
		return err
	}
	grandLock := "SHARE"
	if childIsCSM {
		grandLock = "UPDATE" // Share Master ถูกปรับตาม CSM
	}
	grandSettings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, grandIDs, grandLock)
	if err != nil {
		return err
	}
	// R3: ค่าของ Member lock หลัง agent_game_settings เสมอ · ลูกเป็น CSM = Member ของ Share Master ทุกคน (CSM สร้าง Member ไม่ได้)
	var codes []string
	for _, g := range SortedGroups(req) {
		codes = append(codes, GameCodes(agentManagementCore.PTGroup(g))...)
	}
	memberOwners := []uint{child.ID}
	if childIsCSM {
		memberOwners = grandIDs
	}
	memberSettings, err := memberManagementPostgres.LockUserMemberGameSettingsByAgentsRepository(tx, memberOwners, codes)
	if err != nil {
		return err
	}

	creatorIsMaster := c.UserType == agentManagementCore.UserTypeCompanySeamlessMaster
	oldLog, newLog := map[string]agentManagementCore.ChildPT{}, map[string]agentManagementCore.ChildPT{}
	var remains []models.UserMemberGameSetting
	var followerLogs []models.AccountChangeLog
	for _, g := range SortedGroups(req) {
		group := agentManagementCore.PTGroup(g)
		cur, ok := groupSetting(childSettings, group)
		if !ok {
			continue // บัญชีไม่มีแถวของกลุ่มนี้ (ข้อมูลก่อน module ②) — ไม่มีอะไรให้แก้
		}
		v := req[g].Parsed
		var status bool
		if childIsShareMaster { // CSM แก้ได้แค่ Commission (MGMT-19)
			v = agentManagementCore.ChildPT{PTFromParent: cur.PTFromParent, Force: cur.Force, Remain: cur.Remain, Commission: v.Commission}
			status = cur.Status
		} else {
			status = *req[g].Status
		}
		// ไล่ตาม field: pt_from_parent (เพดาน → ต่ำสุดที่ลูกใช้ MGMT-24) → force → remain_quota → commission_percent
		is := agentManagementCore.CheckChildPT(v, c.Received[group], creatorIsMaster)
		if is.Field == "pt_from_parent" {
			return ChildPTError(g, is)
		}
		var grand []float64
		if !childIsCSM { // ลูกเป็น CSM: Share Master ตามค่าใหม่เอง ไม่นับเป็นค่าต่ำสุด
			for _, s := range grandSettings {
				if gg, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && gg == group {
					grand = append(grand, s.PTFromParent)
				}
			}
		}
		var memberPTs []float64
		for _, s := range memberSettings {
			if gg, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && gg == group {
				memberPTs = append(memberPTs, s.PT)
			}
		}
		if min := agentManagementCore.MinPTFromParent(grand, memberPTs); v.PTFromParent < min { // MGMT-24 · R1 · MGMT-19
			p := utils.FormatNum(min)
			return apperr.ErrPTBelowChildUsage.WithMessage(
				"pt."+g+".pt_from_parent ต่ำกว่าที่ลูกใช้อยู่ ตั้งได้ต่ำสุด "+p,
				"pt."+g+".pt_from_parent is lower than what the child uses, minimum is "+p)
		}
		if err := ChildPTError(g, is); err != nil {
			return err
		}
		if cur.PTFromParent == v.PTFromParent && cur.Force == v.Force && cur.Remain == v.Remain && cur.Commission == v.Commission && cur.Status == status {
			continue // ค่าเหมือนเดิม (MGMT-34)
		}
		if err := agentManagementPostgres.UpdateChildPTRepository(tx, child.ID, GameCodes(group), models.AgentGameSetting{
			PTFromParent: v.PTFromParent, Force: v.Force, Remain: v.Remain, Commission: v.Commission,
			Status: status}, actor.Username, now); err != nil {
			return err
		}
		if v.PTFromParent != cur.PTFromParent { // R2: remain_quota ของ Member = ค่าที่ได้รับใหม่ − pt
			for i, s := range memberSettings {
				if gg, ok := agentManagementCore.GroupOfGame(s.GameCode); ok && gg == group {
					memberSettings[i].Remain = agentManagementCore.MemberRemain(v.PTFromParent, s.PT)
					remains = append(remains, memberSettings[i])
				}
			}
		}
		if childIsCSM && (v.PTFromParent != cur.PTFromParent || status != cur.Status) {
			rows, err := syncFollowersPT(ctx, tx, actor, meta, child.ID, grandIDs, grandSettings, group, v.PTFromParent, status, now)
			if err != nil {
				return err
			}
			followerLogs = append(followerLogs, rows...)
		}
		oldLog[g] = agentManagementCore.ChildPT{PTFromParent: cur.PTFromParent, Force: cur.Force, Remain: cur.Remain, Commission: cur.Commission}
		newLog[g] = v
	}
	if len(newLog) == 0 {
		return nil
	}
	if err := memberManagementPostgres.UpdateUserMemberRemainsRepository(tx, remains); err != nil {
		return err
	}
	if err := agentManagementPostgres.CreateAccountChangeLogsRepository(tx, followerLogs); err != nil {
		return err
	}
	return WriteLog(ctx, tx, actor, meta, targetAgent, child.ID, child.Username, models.ChangeUpdatePT, oldLog, newLog, now)
}

// followerPTValue — ค่าใน log SYNC_FROM_CSM ของ section pt
type followerPTValue struct {
	PTFromParent float64 `json:"pt_from_parent"`
	Force        float64 `json:"force"`
	Remain       float64 `json:"remain_quota"`
	Status       bool    `json:"status"`
}

// syncFollowersPT — ตั้ง Share Master ทุกคนใต้ CSM ตามค่าใหม่ของ CSM (MGMT-19): pt_from_parent · force / remain = 0 · status · commission คงเดิม
// เขียน statement เดียว · คืนแถว log SYNC_FROM_CSM ต่อ Share Master (ผู้ทำ = คนที่แก้ CSM · request_id เดียวกับการแก้ CSM · new_value มี csm_id — lead B5 / V6)
func syncFollowersPT(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta, csmID uint, followerIDs []uint,
	followerSettings []models.AgentGameSetting, group agentManagementCore.PTGroup, ptFromParent float64, status bool, now time.Time) ([]models.AccountChangeLog, error) {
	if len(followerIDs) == 0 {
		return nil, nil
	}
	if err := agentManagementPostgres.SyncFollowerPTRepository(tx, followerIDs, GameCodes(group), ptFromParent, status, actor.Username, now); err != nil {
		return nil, err
	}
	names, err := agentManagementPostgres.ListUsernamesByIDsRepository(tx, followerIDs)
	if err != nil {
		return nil, err
	}
	byID := map[uint][]models.AgentGameSetting{}
	for _, s := range followerSettings {
		byID[s.AgentID] = append(byID[s.AgentID], s)
	}
	rows := make([]models.AccountChangeLog, 0, len(followerIDs))
	for _, id := range followerIDs {
		cur, _ := groupSetting(byID[id], group)
		row, err := ChangeLog(actor, meta, logger.RequestID(ctx), targetAgent, id, names[id], models.ChangeSyncFromCSM,
			map[string]followerPTValue{string(group): {cur.PTFromParent, cur.Force, cur.Remain, cur.Status}},
			map[string]any{"csm_id": csmID, string(group): followerPTValue{ptFromParent, 0, 0, status}}, now)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
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

// applyGames — section status_game (MGMT-20) · เกมที่ค่าเหมือนเดิมข้าม · ไม่มีเกมเปลี่ยน = ไม่เขียน log
// ลูกเป็น CSM: Share Master ทุกคนได้ค่าเดียวกันใน tx เดียว (MGMT-19)
// เปิด / ปิดเกมรายบัญชีให้ลูกตรง · ไม่ส่งต่อลงสายล่าง — ตอนเล่นเช็คทั้งสาย (core.IsGameOpen)
func applyGames(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	child models.UserAgent, req map[string]bool, now time.Time) error {
	settings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, []uint{child.ID}, "UPDATE")
	if err != nil {
		return err
	}
	cur := make(map[string]bool, len(settings))
	for _, s := range settings {
		cur[s.GameCode] = s.StatusGame
	}
	codes := make([]string, 0, len(req))
	for code := range req {
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
		next := req[code]
		if next == was {
			continue // ค่าเหมือนเดิม (MGMT-34)
		}
		if next {
			on = append(on, code)
		} else {
			off = append(off, code)
		}
		oldLog[code], newLog[code] = was, next
	}
	if len(newLog) == 0 {
		return nil
	}
	if err := agentManagementPostgres.UpdateStatusGameRepository(tx, child.ID, on, true); err != nil {
		return err
	}
	if err := agentManagementPostgres.UpdateStatusGameRepository(tx, child.ID, off, false); err != nil {
		return err
	}
	if isCSM(child.Role, child.AgentType) { // Share Master ทุกคนเปิด / ปิดเกมตาม CSM (MGMT-19)
		if err := syncFollowersGames(ctx, tx, actor, meta, child.ID, on, off, now); err != nil {
			return err
		}
	}
	return WriteLog(ctx, tx, actor, meta, targetAgent, child.ID, child.Username, models.ChangeUpdateGames,
		map[string]any{"status_game": oldLog}, map[string]any{"status_game": newLog}, now)
}

// syncFollowersGames — Share Master ทุกคนใต้ CSM เปิด / ปิดเกมตาม CSM (MGMT-19) · log SYNC_FROM_CSM เฉพาะคนที่ค่าเปลี่ยน
func syncFollowersGames(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	csmID uint, on, off []string, now time.Time) error {
	followerIDs, err := agentManagementPostgres.ListAgentChildIDsRepository(tx, csmID)
	if err != nil || len(followerIDs) == 0 {
		return err
	}
	settings, err := agentManagementPostgres.LockAgentGameSettingsRepository(tx, followerIDs, "UPDATE")
	if err != nil {
		return err
	}
	next := map[string]bool{}
	for _, code := range on {
		next[code] = true
	}
	for _, code := range off {
		next[code] = false
	}
	oldBy, newBy := map[uint]map[string]bool{}, map[uint]map[string]bool{}
	for _, s := range settings {
		want, ok := next[s.GameCode]
		if !ok || s.StatusGame == want {
			continue
		}
		if oldBy[s.AgentID] == nil {
			oldBy[s.AgentID], newBy[s.AgentID] = map[string]bool{}, map[string]bool{}
		}
		oldBy[s.AgentID][s.GameCode], newBy[s.AgentID][s.GameCode] = s.StatusGame, want
	}
	if err := agentManagementPostgres.UpdateStatusGameManyRepository(tx, followerIDs, on, true); err != nil {
		return err
	}
	if err := agentManagementPostgres.UpdateStatusGameManyRepository(tx, followerIDs, off, false); err != nil {
		return err
	}
	names, err := agentManagementPostgres.ListUsernamesByIDsRepository(tx, followerIDs)
	if err != nil {
		return err
	}
	var rows []models.AccountChangeLog
	for _, id := range followerIDs {
		if oldBy[id] == nil {
			continue
		}
		row, err := ChangeLog(actor, meta, logger.RequestID(ctx), targetAgent, id, names[id], models.ChangeSyncFromCSM,
			map[string]any{"status_game": oldBy[id]}, map[string]any{"csm_id": csmID, "status_game": newBy[id]}, now)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return agentManagementPostgres.CreateAccountChangeLogsRepository(tx, rows)
}
