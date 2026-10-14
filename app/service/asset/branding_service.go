package asset

import (
	"context"
	"time"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"

	"gorm.io/gorm"
)

// resolveBranding — branding ที่ใช้งานจริงตามสาย: บัญชีใกล้สุดที่ "มี branding" ใช้ทั้งแถว (AST-47 · query เดียว)
func resolveBranding(db *gorm.DB, chain []uint) (assetCore.ResolvedBranding, error) {
	rows, err := assetPostgres.ListBrandingCandidatesRepository(db, chain)
	if err != nil {
		return assetCore.ResolvedBranding{}, err
	}
	cands := make([]assetCore.BrandingCandidate, len(rows))
	for i, r := range rows {
		cands[i] = assetCore.BrandingCandidate{OwnerID: r.AgentID, Branding: assetCore.Branding{SiteName: r.SiteName, LogoKey: r.LogoKey}}
	}
	return assetCore.ResolveBranding(chain, cands), nil
}

// GetBrandingService — POST asset/branding/detail/get (GN12 · AST-47, AST-53): ค่าที่ไล่สายแล้ว + source + inherited_from
func GetBrandingService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target) (assetDto.BrandingResponse, error) {
	var res assetDto.BrandingResponse
	db := database.DBConn.WithContext(ctx)
	chain, err := targetChain(db, target)
	if err != nil {
		return res, err
	}
	r, err := resolveBranding(db, chain)
	if err != nil {
		return res, err
	}
	res = assetDto.BrandingResponse{Source: string(r.Source), SiteName: r.Branding.SiteName, Logo: urlOf(r.Branding.LogoKey)}
	if r.Source == assetCore.SourceUpline {
		names, err := usernames(db, []uint{r.InheritedFrom})
		if err != nil {
			return res, err
		}
		res.InheritedFrom = names[r.InheritedFrom]
	}
	return res, nil
}

// UpdateBrandingService — POST asset/branding/detail/update (GN13 · AST-42): แทนชุด site_name · สร้างแถวให้ถ้ายังไม่มี
func UpdateBrandingService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.BrandingUpdateRequest) error {
	if !target.Resolved {
		return errTargetNotResolved
	}
	name, v := assetCore.NormalizeSiteName(req.SiteName)
	switch v {
	case assetCore.NameEmpty:
		return apperr.ErrValidation.WithMessage("ต้องส่ง site_name (1–50 ตัวอักษร)", "site_name is required (1–50 characters)")
	case assetCore.NameTooLong:
		return apperr.ErrValidation.WithMessage("site_name ยาวเกิน 50 ตัวอักษร", "site_name must not exceed 50 characters")
	case assetCore.NameControlChar:
		return apperr.ErrValidation.WithMessage("site_name ห้ามมีอักขระควบคุม (เช่น ขึ้นบรรทัดใหม่ / tab)", "site_name must not contain control characters")
	}
	return assetPostgres.UpsertSiteBrandingNameRepository(database.DBConn.WithContext(ctx), target.AgentID, name, actor.Username)
}

// UploadLogoService — POST asset/branding/logo/upload (GN14 · AST-43): ชนิด / ขนาดไฟล์ตาม AST-16 · ไม่ตรวจขนาดภาพ
// เขียนทับ key เดิมของบัญชี · ไม่ลบไฟล์เดิมใน storage (lead GQ5)
func UploadLogoService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, file []byte) (assetDto.LogoResponse, error) {
	var res assetDto.LogoResponse
	if !target.Resolved {
		return res, errTargetNotResolved
	}
	info, err := checkImage(file)
	if err != nil {
		return res, err
	}
	suffix, err := newObjectSuffix()
	if err != nil {
		return res, err
	}
	key := assetCore.LogoObjectKey(target.AgentID, time.Now().UnixNano(), suffix, info.Ext())
	if err := putObject(ctx, key, file, info); err != nil {
		return res, err
	}
	if err := assetPostgres.UpsertSiteBrandingLogoRepository(database.DBConn.WithContext(ctx), target.AgentID, key, actor.Username); err != nil {
		removeObject(ctx, key)
		return res, err
	}
	return assetDto.LogoResponse{URL: urlOf(key)}, nil
}

// ResetBrandingService — POST asset/branding/reset (GN16 · AST-56): ลบแถว branding ของบัญชีตัวเอง กลับไปใช้ของชั้นบน
// ไม่มีแถวอยู่แล้ว = สำเร็จ · ไม่ลบไฟล์โลโก้ใน storage (AST-29)
func ResetBrandingService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target) error {
	if !target.Resolved {
		return errTargetNotResolved
	}
	return assetPostgres.DeleteSiteBrandingRepository(database.DBConn.WithContext(ctx), target.AgentID)
}
