package agentmanagement

import (
	"context"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// helper สร้างรูปแบบแสดงบัญชี — ใช้ร่วมกับ account Profile (ACC-16, ACC-19)

// AgentPTViews — ค่าหุ้นส่วนชุดเดียวต่อกลุ่ม + เปิด / ปิดทีละเกม จากแถวต่อเกม (MGMT-16, MGMT-20)
func AgentPTViews(settings []models.AgentGameSetting) (map[string]agentManagementDto.PTGroupView, map[string]bool) {
	pt := map[string]agentManagementDto.PTGroupView{}
	statusGame := map[string]bool{}
	for _, s := range settings {
		statusGame[s.GameCode] = s.StatusGame
		g, ok := agentManagementCore.GroupOfGame(s.GameCode)
		if !ok {
			continue
		}
		pt[string(g)] = agentManagementDto.PTGroupView{ // ค่าในกลุ่มเท่ากันทุกเกม
			PTFromParent: utils.Percent(s.PTFromParentBP), PT: utils.Percent(s.PTBP), Force: utils.Percent(s.ForceBP),
			RemainQuota: utils.Percent(s.RemainBP), CommissionPercent: utils.Percent(s.CommissionBP), Status: s.Status,
		}
	}
	return pt, statusGame
}

// MemberPTViews — Member มีแค่ Commission ต่อกลุ่ม (MGMT-21)
func MemberPTViews(settings []models.MemberGameSetting) map[string]agentManagementDto.MemberPTGroupView {
	pt := map[string]agentManagementDto.MemberPTGroupView{}
	for _, s := range settings {
		if g, ok := agentManagementCore.GroupOfGame(s.GameCode); ok {
			pt[string(g)] = agentManagementDto.MemberPTGroupView{CommissionPercent: utils.Percent(s.CommissionBP)}
		}
	}
	return pt
}

// BalanceViews — 1 รายการต่อสกุลของบัญชี · ไม่มียอด = 0 (account ACC-19)
func BalanceViews(currencies []string, amount map[string]int64) []agentManagementDto.BalanceView {
	out := make([]agentManagementDto.BalanceView, len(currencies))
	for i, c := range currencies {
		out[i] = agentManagementDto.BalanceView{Currency: c, Amount: utils.Money(amount[c])}
	}
	return out
}

// canSeePT — มีสิทธิ์ pt ≥ view ไหม · ไม่มี = response ไม่มี field pt (MGMT-51)
func canSeePT(ctx context.Context, actor agentAuthService.Actor) (bool, error) {
	perms, err := PermissionsOf(ctx, actor)
	if err != nil {
		return false, err
	}
	return perms[agentManagementCore.MenuPT].Allows(agentManagementCore.LevelView), nil
}

// chainStatus — สถานะที่ใช้งานจริงของบัญชีฝั่ง agent = เข้มที่สุดของตัวเองและหัวสายทุกชั้น (ACC-30 · AUTH-53)
func chainStatus(db *gorm.DB, agentID uint, own models.AgentStatus) (models.AgentStatus, error) {
	ancestors, err := postgres.ListAncestorStatusesRepository(db, agentID)
	if err != nil {
		return "", err
	}
	return agentAuthCore.WorstStatus(append(ancestors, own)...), nil
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// optionalTime — "" แทน null (account ACC-32) · รูปแบบเดียวกับ encoding/json ของ time.Time
func optionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
