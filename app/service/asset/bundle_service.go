package asset

import (
	"context"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/platform/database"
	"app/platform/logger"
)

// GetBundleService — POST asset/bundle/detail/get (GN22 · AST-45 – AST-48): asset + theme + branding ของ (บัญชี, เกม)
// preview ว่าผู้เล่นของบัญชีนี้จะเห็นอะไร · ผู้เล่นใช้ service ตัวเดียวกัน (GN23) ห้ามเขียน logic รวมซ้ำ
// asset คือส่วนที่จำเป็น — อ่าน branding ไม่สำเร็จต้องไม่ทำให้ทั้ง bundle ล้ม (site เป็นค่าว่าง + log warn — AST-46)
func GetBundleService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.GameRequest) (assetDto.BundleResponse, error) {
	var res assetDto.BundleResponse
	if err := requireGame(req.GameCode); err != nil {
		return res, err
	}
	db := database.DBConn.WithContext(ctx)
	chain, err := targetChain(db, target)
	if err != nil {
		return res, err
	}

	types, err := assetPostgres.ListAssetTypesRepository(db, req.GameCode, true)
	if err != nil {
		return res, err
	}
	resolved, err := resolveUsed(db, req.GameCode, chain)
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

	theme, err := resolveTheme(ctx, db, req.GameCode, chain)
	if err != nil {
		return res, err
	}
	themeView, err := themeBase(theme)
	if err != nil {
		return res, err
	}

	site := assetDto.BundleSite{}
	if b, err := resolveBranding(db, chain); err != nil {
		logger.Ctx(ctx).Warnw("asset bundle: read branding failed", "agent_id", target.AgentID, "error", err)
	} else {
		site = assetDto.BundleSite{SiteName: b.Branding.SiteName, Logo: urlOf(b.Branding.LogoKey)}
	}

	return assetDto.BundleResponse{Source: string(assetCore.OverallSource(sources)), Assets: assets, Theme: themeView, Site: site}, nil
}
