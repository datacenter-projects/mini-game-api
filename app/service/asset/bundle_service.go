package asset

import (
	"context"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// GetBundleService — POST asset/bundle/detail/get (GN22 · AST-45 – AST-48): asset + theme + branding ของบัญชีที่ login
// preview ว่าผู้เล่นของบัญชีนี้จะเห็นอะไร · ผู้เล่นใช้ตัวสร้างเดียวกัน (BundleOfAgentService — GN23) ห้ามเขียน logic รวมซ้ำ
func GetBundleService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.GameRequest) (assetDto.BundleResponse, error) {
	if err := requireGame(req.GameCode); err != nil {
		return assetDto.BundleResponse{}, err
	}
	db := database.DBConn.WithContext(ctx)
	chain, err := targetChain(db, target)
	if err != nil {
		return assetDto.BundleResponse{}, err
	}
	return buildBundle(ctx, db, chain, req.GameCode, target.AgentID)
}

// BundleOfAgentService — bundle ของบัญชี agentID (ไล่สายจากบัญชีนี้) · ใช้โดยเส้นผู้เล่น POST /api/v1/player/pr/asset/bundle (GN23)
// agentID = ผู้สร้างตรงของ Member จาก session เท่านั้น (AST-52) · ผลเท่ากับ preview ของบัญชีนั้นทุกส่วน (AST-48)
// ไม่เช็คเกมปิด (lead 2026-10-16 Q5 — Bet เช็คเอง) · theme ไม่มี inherited_from (Q6) · ไม่ cache (Q7)
func BundleOfAgentService(ctx context.Context, agentID uint, gameCode string) (assetDto.BundleResponse, error) {
	if err := requireGame(gameCode); err != nil {
		return assetDto.BundleResponse{}, err
	}
	db := database.DBConn.WithContext(ctx)
	chain, err := chainOf(db, agentID)
	if err != nil {
		return assetDto.BundleResponse{}, err
	}
	return buildBundle(ctx, db, chain, gameCode, agentID)
}

// buildBundle — ตัวสร้าง bundle ตัวเดียวของ BO preview และผู้เล่น (AST-45 – AST-48)
// asset คือส่วนที่จำเป็น — อ่าน branding ไม่สำเร็จต้องไม่ทำให้ทั้ง bundle ล้ม (site เป็นค่าว่าง + log warn — AST-46)
func buildBundle(ctx context.Context, db *gorm.DB, chain []uint, gameCode string, logAgentID uint) (assetDto.BundleResponse, error) {
	var res assetDto.BundleResponse
	types, err := assetPostgres.ListAssetTypesRepository(db, gameCode, true)
	if err != nil {
		return res, err
	}
	resolved, err := resolveUsed(db, gameCode, chain)
	if err != nil {
		return res, err
	}
	assets := make(map[string]string, len(resolved))
	var sources []assetCore.Source
	for _, t := range types {
		if r, ok := resolved[t.ID]; ok {
			assets[t.AssetType] = urlOf(r.StorageKey)
			sources = append(sources, r.Source)
		}
	}

	theme, err := resolveTheme(ctx, db, gameCode, chain)
	if err != nil {
		return res, err
	}
	themeView, err := themeBase(theme)
	if err != nil {
		return res, err
	}

	site := assetDto.BundleSite{}
	if b, err := resolveBranding(db, chain); err != nil {
		logger.Ctx(ctx).Warnw("asset bundle: read branding failed", "agent_id", logAgentID, "error", err)
	} else {
		site = assetDto.BundleSite{SiteName: b.Branding.SiteName, Logo: urlOf(b.Branding.LogoKey)}
	}

	return assetDto.BundleResponse{Source: string(assetCore.OverallSource(sources)), Assets: assets, Theme: themeView, Site: site}, nil
}
