// Package asset คือ request / response ของ module asset — docs/modules/asset.md หัวข้อ 5
package asset

import (
	assetCore "app/app/core/asset"
	"app/pkg/apperr"
	"app/pkg/utils"
)

// ResolveTarget แปลง scope ที่อ่านมาจาก body / form เป็นเป้าหมาย (AST-08) · บัญชี AGENT เติมทีหลังจาก token (RequireAssetTarget)
//
//	withScope = true : ส่ง scope = "GLOBAL" / "AGENT" บังคับ (asset / theme)
//	withScope = false: เส้น branding / bundle — ไม่มี scope (ของตัวเองเสมอ) · ส่ง scope อื่นนอกจาก AGENT = 422
//
// agentIDSent = มี field agent_id มา (ค่าใดก็ตาม) → 422 · BO ระบุบัญชีอื่นไม่ได้ (lead 2026-10-16)
// ไม่เมินเงียบๆ: หน้าบ้านเดิมที่ยังส่ง id ของบัญชีอื่นต้องรู้ตัว ไม่ใช่ไปแก้ของตัวเองโดยไม่ตั้งใจ
// ผลยังไม่ Resolved (RequireAssetTarget ตั้งให้หลังเติมบัญชี) · ไม่ส่ง / ค่าอื่น = 422 — ห้ามตีความค่าว่างเป็น GLOBAL เงียบๆ
func ResolveTarget(withScope bool, scope utils.JSONString, agentIDSent bool) (assetCore.Target, error) {
	if agentIDSent {
		return assetCore.Target{}, apperr.ErrValidation.WithMessage(
			"ไม่ต้องส่ง agent_id (ใช้บัญชีที่เข้าสู่ระบบ)", "agent_id is not accepted (the logged-in account is used)")
	}

	if !withScope {
		if scope.Present && (!scope.IsString || scope.Value != string(assetCore.ScopeAgent)) {
			return assetCore.Target{}, apperr.ErrValidation.WithMessage("ไม่ต้องส่ง scope (เส้นนี้ใช้ได้เฉพาะของบัญชี)", "scope is not accepted here (this endpoint works on an account only)")
		}
		return assetCore.Target{Scope: assetCore.ScopeAgent}, nil
	}

	if !scope.Present || !scope.IsString {
		return assetCore.Target{}, apperr.ErrValidation.WithMessage(`ต้องส่ง scope เป็น "GLOBAL" หรือ "AGENT"`, `scope must be "GLOBAL" or "AGENT"`)
	}
	switch assetCore.Scope(scope.Value) {
	case assetCore.ScopeGlobal, assetCore.ScopeAgent:
		return assetCore.Target{Scope: assetCore.Scope(scope.Value)}, nil
	}
	return assetCore.Target{}, apperr.ErrValidation.WithMessage(`scope ต้องเป็น "GLOBAL" หรือ "AGENT"`, `scope must be "GLOBAL" or "AGENT"`)
}
