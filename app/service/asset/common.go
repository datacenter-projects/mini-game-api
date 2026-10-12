// Package asset คือ business logic ของหน้าตาเกม / หน้าเว็บ — spec: docs/modules/asset.md
package asset

import (
	"encoding/json"
	"errors"
	"time"

	accountCore "app/app/core/account"
	agentManagementCore "app/app/core/agent_management"
	assetCore "app/app/core/asset"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/configs"

	"gorm.io/gorm"
)

// errTargetNotResolved — route ลืมใส่ ResolveAssetTarget → ปฏิเสธ (fail-closed — AST-09) · เป็น bug ของ route จึงตอบ 500
var errTargetNotResolved = apperr.ErrInternal.Wrap(errors.New("asset: target not resolved (ResolveAssetTarget missing on route)"))

// loadOwner — บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ — AUTH-26)
func loadOwner(db *gorm.DB, actor agentAuthService.Actor) (models.UserAgent, error) {
	owner, err := agentManagementPostgres.GetAgentProfileRepository(db, actor.AgentID)
	if errors.Is(err, apperr.ErrNotFound) {
		return owner, apperr.ErrSessionEnded
	}
	return owner, err
}

// parentIDs — id ของชั้นบนจาก cnf เรียงจาก Superadmin ลงมาถึงผู้สร้างตรง ไม่รวมตัวเอง (MGMT-61)
func parentIDs(cnf string) ([]uint, error) {
	var c agentManagementCore.Chain
	if err := json.Unmarshal([]byte(cnf), &c); err != nil {
		return nil, err
	}
	ids := make([]uint, len(c.Parent))
	for i, n := range c.Parent {
		ids[i] = n.ID
	}
	return ids, nil
}

// chainOf — สายของบัญชี agentID: ตัวเอง → ชั้นบนจากใกล้ → ไกล (AST-57) · ไม่พบบัญชี = apperr.ErrDownlineNotFound
func chainOf(db *gorm.DB, agentID uint) ([]uint, error) {
	cnf, err := agentManagementPostgres.GetAgentChainRepository(db, agentID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, apperr.ErrDownlineNotFound
	}
	if err != nil {
		return nil, err
	}
	parents, err := parentIDs(cnf)
	if err != nil {
		return nil, err
	}
	return assetCore.ChainIDs(agentID, parents), nil
}

// targetChain — สายสำหรับไล่ค่าของเป้าหมาย: GLOBAL = ว่าง (เอาเฉพาะค่ากลาง) ·
// AGENT = สายที่ RequireAssetTarget โหลดไว้แล้ว (ไม่มี = อ่านเอง)
func targetChain(db *gorm.DB, t assetCore.Target) ([]uint, error) {
	if !t.Resolved {
		return nil, errTargetNotResolved
	}
	if t.Scope == assetCore.ScopeGlobal {
		return nil, nil
	}
	if len(t.Chain) > 0 {
		return t.Chain, nil
	}
	return chainOf(db, t.AgentID)
}

// targetAgentID — id ของเจ้าของแถวที่จะเขียน: nil = GLOBAL
func targetAgentID(t assetCore.Target) *uint {
	if t.Scope == assetCore.ScopeGlobal {
		return nil
	}
	id := t.AgentID
	return &id
}

// findSlot — ช่องตาม (game_code, asset_type) ต้องตรง catalog ทุกตัวอักษร (AST-14) · ไม่พบ = 405401
// อ่าน catalog ไม่สำเร็จ = คืน error เดิม (500) ไม่ข้ามไปบันทึกต่อ (fail-closed — AST-18)
func findSlot(db *gorm.DB, gameCode, assetType string) (models.AssetType, error) {
	if !assetCore.IsGameCode(gameCode) {
		return models.AssetType{}, apperr.ErrAssetSlotNotFound
	}
	t, err := assetPostgres.FindAssetTypeRepository(db, gameCode, assetType)
	if errors.Is(err, apperr.ErrNotFound) {
		return t, apperr.ErrAssetSlotNotFound
	}
	return t, err
}

// requireGame — รหัสเกมต้องตรงทุกตัวอักษร (AST-10, AST-14)
func requireGame(gameCode string) error {
	if !assetCore.IsGameCode(gameCode) {
		return apperr.ErrAssetSlotNotFound
	}
	return nil
}

// urlOf — URL จาก storage key (AST-49) · key ว่าง = ""
func urlOf(key string) string {
	return assetCore.ObjectURL(configs.Cfg.Storage.CDNBaseURL, key)
}

func timeText(t time.Time) string { return accountCore.OptionalTime(&t) }

// usernames — username ของบัญชีที่ให้ค่ามา (`inherited_from` — AST-53) · query เดียว
func usernames(db *gorm.DB, ids []uint) (map[uint]string, error) {
	return assetPostgres.ListAgentUsernamesRepository(db, ids)
}

// resolveUsed — used ที่ใกล้ที่สุดต่อช่องของเกม ตามสาย (AST-25 · query เดียว + pure function)
func resolveUsed(db *gorm.DB, gameCode string, chain []uint) (map[uint]assetCore.ResolvedUsed, error) {
	rows, err := assetPostgres.ListUsedCandidatesRepository(db, gameCode, chain)
	if err != nil {
		return nil, err
	}
	cands := make([]assetCore.UsedCandidate, len(rows))
	for i, r := range rows {
		cands[i] = assetCore.UsedCandidate{SlotID: r.AssetTypeID, OwnerID: r.AgentID, ConfigID: r.AssetConfigID, StorageKey: r.StorageKey}
	}
	return assetCore.ResolveUsed(chain, cands), nil
}

func isNotFound(err error) bool { return errors.Is(err, apperr.ErrNotFound) }
