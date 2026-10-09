package agentmanagement

import (
	"context"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentAuthService "app/app/service/agent_auth"

	"gorm.io/gorm"
)

// helper สร้างรูปแบบแสดงบัญชี — ใช้ร่วมกับ account Profile (ACC-16, ACC-19)

// AgentPTViews — ค่าหุ้นส่วนชุดเดียวต่อระบบ + เปิด / ปิดทีละเกม จากแถวต่อเกม (MGMT-16, MGMT-20)
// ค่าในระบบเท่ากันทุกเกม · ผู้แก้ล่าสุด = แถวที่ updated_at ใหม่สุด
func AgentPTViews(settings []models.AgentGameSetting) (map[string]agentManagementDto.PTGroupView, map[string]bool) {
	pt := map[string]agentManagementDto.PTGroupView{}
	latest := map[string]time.Time{}
	statusGame := map[string]bool{}
	for _, s := range settings {
		statusGame[s.GameCode] = s.StatusGame
		g, ok := agentManagementCore.GroupOfGame(s.GameCode)
		if !ok {
			continue
		}
		if t, seen := latest[string(g)]; seen && !s.UpdatedAt.After(t) {
			continue
		}
		latest[string(g)] = s.UpdatedAt
		pt[string(g)] = agentManagementDto.PTGroupView{
			PTFromParent: s.PTFromParent, Force: s.Force,
			RemainQuota: s.Remain, CommissionPercent: s.Commission, Status: s.Status,
			CreatedAt: formatTime(s.CreatedAt), CreatedBy: s.CreatedBy, UpdatedAt: formatTime(s.UpdatedAt), UpdatedBy: s.UpdatedBy,
		}
	}
	return pt, statusGame
}

// MemberPTViews — pt · remain_quota · Commission ของ Member ชุดเดียวต่อระบบ (MGMT-21 แก้ 2026-10-09)
func MemberPTViews(settings []models.UserMemberGameSetting) map[string]agentManagementDto.MemberPTGroupView {
	pt := map[string]agentManagementDto.MemberPTGroupView{}
	latest := map[string]time.Time{}
	for _, s := range settings {
		g, ok := agentManagementCore.GroupOfGame(s.GameCode)
		if !ok {
			continue
		}
		if t, seen := latest[string(g)]; seen && !s.UpdatedAt.After(t) {
			continue
		}
		latest[string(g)] = s.UpdatedAt
		pt[string(g)] = agentManagementDto.MemberPTGroupView{PT: s.PT, RemainQuota: s.Remain, CommissionPercent: s.Commission,
			CreatedAt: formatTime(s.CreatedAt), CreatedBy: s.CreatedBy, UpdatedAt: formatTime(s.UpdatedAt), UpdatedBy: s.UpdatedBy}
	}
	return pt
}

// BalanceViews — 1 รายการต่อสกุลของบัญชี · ไม่มียอด = 0 (account ACC-19)
func BalanceViews(currencies []string, amount map[string]float64) []agentManagementDto.BalanceView {
	out := make([]agentManagementDto.BalanceView, len(currencies))
	for i, c := range currencies {
		out[i] = agentManagementDto.BalanceView{Currency: c, Amount: amount[c]}
	}
	return out
}

// CanSeePT — มีสิทธิ์ pt ≥ view ไหม · ไม่มี = response ไม่มี field pt (MGMT-51)
func CanSeePT(ctx context.Context, actor agentAuthService.Actor) (bool, error) {
	perms, err := PermissionsOf(ctx, actor)
	if err != nil {
		return false, err
	}
	return perms[agentManagementCore.MenuPT].Allows(agentManagementCore.LevelView), nil
}

// ChainStatus — สถานะที่ใช้งานจริงของบัญชีฝั่ง agent = เข้มที่สุดของตัวเองและหัวสายทุกชั้น (ACC-30 · AUTH-53)
func ChainStatus(db *gorm.DB, agentID uint, own models.AgentStatus) (models.AgentStatus, error) {
	ancestors, err := agentAuthPostgres.ListAncestorStatusesRepository(db, agentID)
	if err != nil {
		return "", err
	}
	return agentAuthCore.WorstStatus(append(ancestors, own)...), nil
}

func StringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// OptionalTime — "" แทน null (account ACC-32) · รูปแบบเดียวกับ encoding/json ของ time.Time
func OptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func formatTime(t time.Time) string { return t.Format(time.RFC3339Nano) }
