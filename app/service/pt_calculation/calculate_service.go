// Package ptcalculation คือ service คิด PT สุดท้าย + % Commission ทุกชั้นในสายของ Member — spec: docs/modules/pt_calculation.md
// module เดิมพันเรียก CalculateMemberPT ด้วย tx ของ bet · เส้น dev เรียก CalculateMemberPTService
package ptcalculation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	agentManagementCore "app/app/core/agent_management"
	ptCalculationCore "app/app/core/pt_calculation"
	"app/app/models"
	ptCalculationPostgres "app/app/repository/postgres/pt_calculation"
	"app/pkg/apperr"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// PositionMember — position ของแถว Member (PTC-23) · ฝั่ง agent ใช้ position จาก cnf
const PositionMember = "member"

// Layer — ผลของชั้นหนึ่ง เรียง Superadmin → … → Member (PTC-22) · PT = nil เฉพาะแถว Member
type Layer struct {
	ID         uint
	Position   string
	PT         *float64
	Commission float64
}

// CalculateMemberPTService — เส้น dev POST /api/v1/bo/pr/dev/pt/calculate (PTC-40)
func CalculateMemberPTService(ctx context.Context, memberID uint, gameCode string) ([]Layer, error) {
	return CalculateMemberPT(ctx, database.DBConn.WithContext(ctx), memberID, gameCode)
}

// CalculateMemberPT — คิด PT สุดท้าย (PTC-02 – PTC-08) + Commission (PTC-11) ของ Member ในเกมนี้
// db = connection หรือ tx ของผู้เรียก · อ่านอย่างเดียว 3 query (PTC-30) · ไม่พบ Member = 408401 · ข้อมูลในสายผิด = 508601
func CalculateMemberPT(ctx context.Context, db *gorm.DB, memberID uint, gameCode string) ([]Layer, error) {
	m, err := ptCalculationPostgres.GetPTCMemberRepository(db, memberID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, apperr.ErrPTCMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	var chain agentManagementCore.Chain
	if err := json.Unmarshal([]byte(m.Cnf), &chain); err != nil {
		return nil, invalidChain(fmt.Errorf("member %d cnf: %w", m.ID, err))
	}
	nodes := chain.Parent
	if len(nodes) == 0 || nodes[0].Position != strings.ToLower(string(models.AgentRoleSuperAdmin)) || nodes[len(nodes)-1].ID != m.AgentID {
		return nil, invalidChain(fmt.Errorf("member %d cnf %s does not start at superadmin and end at creator %d", m.ID, m.Cnf, m.AgentID))
	}

	ms, found, err := ptCalculationPostgres.GetPTCMemberSettingRepository(db, m.ID, gameCode)
	if err != nil {
		return nil, err
	}
	if !found || ms.ParentID != m.AgentID {
		return nil, invalidChain(fmt.Errorf("member %d game %s setting missing or parent_id mismatch", m.ID, gameCode))
	}

	ids := make([]uint, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	rows, err := ptCalculationPostgres.ListPTCChainSettingsRepository(db, ids, gameCode)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]ptCalculationPostgres.ChainSettingRow, len(rows))
	for _, r := range rows {
		byID[r.AgentID] = r
	}

	layers := make([]ptCalculationCore.Layer, len(nodes))
	settings := make([]ptCalculationPostgres.ChainSettingRow, len(nodes))
	for i, n := range nodes {
		r, ok := byID[n.ID]
		if !ok || !parentMatches(r.ParentID, nodes, i) || strings.ToLower(string(r.Role)) != n.Position {
			return nil, invalidChain(fmt.Errorf("member %d game %s layer %d (agent %d) setting missing or not matching cnf", m.ID, gameCode, i, n.ID))
		}
		settings[i] = r
		layers[i] = ptCalculationCore.Layer{
			Received: r.PTFromParent, Force: r.Force, Remain: r.Remain, AcceptsPT: r.Status,
			IsCSM: r.Role == models.AgentRoleCompany && r.AgentType != nil && *r.AgentType == models.AgentTypeSeamlessMaster,
		}
	}

	res, err := ptCalculationCore.Calculate(ptCalculationCore.Input{Layers: layers, MemberPT: ms.PT})
	if err != nil {
		return nil, invalidChain(fmt.Errorf("member %d game %s: %w", m.ID, gameCode, err))
	}
	for _, w := range res.Warnings { // PTC-04A / PTC-10 — คิดต่อได้ แต่ log ไว้ตรวจข้อมูล
		logger.Ctx(ctx).Warnw("pt calculation warning", "member_id", m.ID, "game_code", gameCode, "kind", warningName(w.Kind),
			"agent_id", nodes[w.Layer].ID, "want", w.Want, "got", w.Got)
	}

	out := make([]Layer, 0, len(nodes)+1)
	for i, n := range nodes {
		pt := res.PT[i]
		commission := settings[i].Commission
		if i == 0 { // Superadmin ไม่มีผู้สร้าง (PTC-11)
			commission = 0
		}
		out = append(out, Layer{ID: n.ID, Position: n.Position, PT: &pt, Commission: commission})
	}
	out = append(out, Layer{ID: m.ID, Position: PositionMember, Commission: ms.Commission})
	return out, nil
}

// parentMatches — parent_id ของแถวชั้น i ต้องเป็นชั้น i−1 ใน cnf · Superadmin = NULL (MGMT-62 · PTC-09)
func parentMatches(parentID *uint, nodes []agentManagementCore.ChainNode, i int) bool {
	if i == 0 {
		return parentID == nil
	}
	return parentID != nil && *parentID == nodes[i-1].ID
}

// invalidChain — 508601 + แนบรายละเอียดไว้ log (client ไม่เห็น — PTC-08 / PTC-09)
func invalidChain(cause error) error { return apperr.ErrPTCInvalidChain.Wrap(cause) }

func warningName(k ptCalculationCore.WarningKind) string {
	switch k {
	case ptCalculationCore.WarnCSMForce:
		return "csm_force_skipped"
	case ptCalculationCore.WarnCappedByLeft:
		return "capped_by_left"
	default:
		return "unknown"
	}
}
