// Package asset คือ request / response ของ module asset — docs/modules/asset.md หัวข้อ 5
package asset

import (
	"strconv"
	"strings"

	assetCore "app/app/core/asset"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// ResolveTarget แปลง scope + agent_id ที่อ่านมาจาก body / form เป็นเป้าหมาย (AST-08)
//
//	withScope = true : ส่ง scope = "GLOBAL" / "AGENT" บังคับ · AGENT ต้องมี agent_id > 0 · GLOBAL ต้องไม่ส่ง agent_id (หรือ 0)
//	withScope = false: เส้น branding / bundle — ไม่มี scope (มีแต่ AGENT) · ต้องมี agent_id > 0 · ส่ง scope อื่นนอกจาก AGENT = 422
//
// agentID = ข้อความดิบของ field (ตัวเลข JSON หรือค่า form) · "" = ไม่ส่ง
// ไม่ส่ง / ค่าอื่น / อ่านไม่ออก = 422 msg บอก field — ห้ามตีความค่าว่างหรือ 0 เป็น GLOBAL เงียบๆ
func ResolveTarget(withScope bool, scope utils.JSONString, agentID string) (assetCore.Target, error) {
	agentID = strings.TrimSpace(agentID)

	var id uint
	if agentID != "" {
		n, err := strconv.ParseUint(agentID, 10, 32)
		if err != nil {
			return assetCore.Target{}, apperr.ErrValidation.WithMessage("agent_id ต้องเป็นตัวเลขจำนวนเต็ม", "agent_id must be an integer")
		}
		id = uint(n)
	}

	if !withScope {
		if scope.Present && (!scope.IsString || scope.Value != string(assetCore.ScopeAgent)) {
			return assetCore.Target{}, apperr.ErrValidation.WithMessage("ไม่ต้องส่ง scope (เส้นนี้ใช้ได้เฉพาะของบัญชี)", "scope is not accepted here (this endpoint works on an account only)")
		}
		if id == 0 {
			return assetCore.Target{}, apperr.ErrValidation.WithMessage("ต้องส่ง agent_id มากกว่า 0", "agent_id is required and must be greater than 0")
		}
		return assetCore.Target{Resolved: true, Scope: assetCore.ScopeAgent, AgentID: id}, nil
	}

	if !scope.Present || !scope.IsString {
		return assetCore.Target{}, apperr.ErrValidation.WithMessage(`ต้องส่ง scope เป็น "GLOBAL" หรือ "AGENT"`, `scope must be "GLOBAL" or "AGENT"`)
	}
	switch assetCore.Scope(scope.Value) {
	case assetCore.ScopeGlobal:
		if id != 0 {
			return assetCore.Target{}, apperr.ErrValidation.WithMessage("scope = GLOBAL ห้ามส่ง agent_id", "agent_id must not be sent when scope is GLOBAL")
		}
		return assetCore.Target{Resolved: true, Scope: assetCore.ScopeGlobal}, nil
	case assetCore.ScopeAgent:
		if id == 0 {
			return assetCore.Target{}, apperr.ErrValidation.WithMessage("scope = AGENT ต้องส่ง agent_id มากกว่า 0", "agent_id is required and must be greater than 0 when scope is AGENT")
		}
		return assetCore.Target{Resolved: true, Scope: assetCore.ScopeAgent, AgentID: id}, nil
	}
	return assetCore.Target{}, apperr.ErrValidation.WithMessage(`scope ต้องเป็น "GLOBAL" หรือ "AGENT"`, `scope must be "GLOBAL" or "AGENT"`)
}
