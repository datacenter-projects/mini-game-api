package asset

import (
	"context"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/models"
	assetPostgres "app/app/repository/postgres/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// resolveTheme — theme ที่ใช้งานจริงของเกมตามสาย (AST-37 · query เดียว + pure function)
// ค่ากลางถูก seed ทุกเกม — ถ้าหายไป (ข้อมูลผิดปกติ) ใช้ค่าตั้งต้นแทนการล้มทั้งเส้น พร้อม log
func resolveTheme(ctx context.Context, db *gorm.DB, game string, chain []uint) (assetCore.ResolvedTheme, error) {
	rows, err := assetPostgres.ListThemeCandidatesRepository(db, game, chain)
	if err != nil {
		return assetCore.ResolvedTheme{}, err
	}
	cands := make([]assetCore.ThemeCandidate, len(rows))
	for i, r := range rows {
		cands[i] = assetCore.ThemeCandidate{OwnerID: r.AgentID, Theme: assetCore.Theme{FontFamily: r.FontFamily,
			MainColor1: r.MainColor1, MainColor2: r.MainColor2, StatsBarColor: r.StatsBarColor, StatsBarFontFamily: r.StatsBarFontFamily}}
	}
	resolved, ok := assetCore.ResolveTheme(chain, cands)
	if !ok {
		logger.Ctx(ctx).Warnw("asset: global theme row missing, using default", "game", game)
		return assetCore.ResolvedTheme{Theme: assetCore.DefaultTheme(), Source: assetCore.SourceGlobal}, nil
	}
	return resolved, nil
}

// themeBase — ธีม + shades คำนวณสดจากสีที่ resolve แล้ว (AST-40)
func themeBase(r assetCore.ResolvedTheme) (assetDto.ThemeBase, error) {
	s1, err := assetCore.Shades(r.Theme.MainColor1)
	if err != nil {
		return assetDto.ThemeBase{}, apperr.ErrInternal.Wrap(err)
	}
	s2, err := assetCore.Shades(r.Theme.MainColor2)
	if err != nil {
		return assetDto.ThemeBase{}, apperr.ErrInternal.Wrap(err)
	}
	return assetDto.ThemeBase{Source: string(r.Source), FontFamily: r.Theme.FontFamily, MainColor1: r.Theme.MainColor1,
		MainColor2: r.Theme.MainColor2, StatsBarColor: r.Theme.StatsBarColor, StatsBarColorHex: assetCore.StatsBarColorHex(r.Theme),
		StatsBarFontFamily: r.Theme.StatsBarFontFamily, MainColor1Shades: s1, MainColor2Shades: s2}, nil
}

// GetThemeService — POST asset/theme/detail/get (GN9 · AST-37, AST-53): ธีมที่ไล่สายแล้ว + source + inherited_from
func GetThemeService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.GameRequest) (assetDto.ThemeDetailResponse, error) {
	var res assetDto.ThemeDetailResponse
	if err := requireGame(req.GameCode); err != nil {
		return res, err
	}
	db := database.DBConn.WithContext(ctx)
	chain, err := targetChain(db, target)
	if err != nil {
		return res, err
	}
	resolved, err := resolveTheme(ctx, db, req.GameCode, chain)
	if err != nil {
		return res, err
	}
	base, err := themeBase(resolved)
	if err != nil {
		return res, err
	}
	res.ThemeBase = base
	if resolved.Source == assetCore.SourceUpline {
		names, err := usernames(db, []uint{resolved.InheritedFrom})
		if err != nil {
			return res, err
		}
		res.InheritedFrom = names[resolved.InheritedFrom]
	}
	return res, nil
}

// themeViolation — แปลงผลตรวจธีมเป็น 422 ที่บอกว่าผิดที่ field ไหน (AST-33 – AST-36)
func themeViolation(v *assetCore.ThemeViolation) error {
	switch v.Reason {
	case assetCore.ThemeRequired:
		return apperr.ErrValidation.WithMessage("ต้องส่ง "+v.Field+" (ธีมต้องส่งครบทั้ง 5 ค่า)", v.Field+" is required (send all 5 theme values)")
	case assetCore.ThemeBadFont:
		return apperr.ErrValidation.WithMessage(v.Field+" ต้องเป็นหนึ่งใน noto, prompt, kanit, mitr, sarabun", v.Field+" must be one of noto, prompt, kanit, mitr, sarabun")
	case assetCore.ThemeBadColor:
		return apperr.ErrValidation.WithMessage(v.Field+" ต้องเป็นสีรูปแบบ #RRGGBB", v.Field+" must be a color in #RRGGBB format")
	case assetCore.ThemeBadStatsBarColor:
		return apperr.ErrValidation.WithMessage(v.Field+" ต้องเป็น main_color_1 หรือ main_color_2", v.Field+" must be main_color_1 or main_color_2")
	}
	return apperr.ErrValidation
}

// UpdateThemeService — POST asset/theme/detail/update (GN10 · AST-33 – AST-38)
// แทนทั้งชุด · กฎเดียวกันทั้ง GLOBAL และ AGENT · ส่งค่าเท่ากับที่ใช้อยู่จริง (ไล่สายแล้ว) = 200 ไม่เขียนอะไร (lead GE10-1)
func UpdateThemeService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.ThemeUpdateRequest) error {
	if err := requireGame(req.GameCode); err != nil {
		return err
	}
	next, v := assetCore.NormalizeTheme(assetCore.Theme{FontFamily: req.FontFamily, MainColor1: req.MainColor1, MainColor2: req.MainColor2,
		StatsBarColor: req.StatsBarColor, StatsBarFontFamily: req.StatsBarFontFamily})
	if v != nil {
		return themeViolation(v)
	}
	db := database.DBConn.WithContext(ctx)
	chain, err := targetChain(db, target)
	if err != nil {
		return err
	}
	current, err := resolveTheme(ctx, db, req.GameCode, chain)
	if err != nil {
		return err
	}
	if current.Theme == next {
		return nil
	}
	return assetPostgres.UpsertGameThemeRepository(db, targetAgentID(target), req.GameCode, models.GameTheme{
		FontFamily: next.FontFamily, MainColor1: next.MainColor1, MainColor2: next.MainColor2,
		StatsBarColor: next.StatsBarColor, StatsBarFontFamily: next.StatsBarFontFamily}, actor.Username)
}

// ResetThemeService — POST asset/theme/reset (GN11 · AST-55): ลบแถว override ของ (บัญชีตัวเอง, เกม) กลับไปใช้ของชั้นบน
// scope GLOBAL = 422 (ค่ากลางห้ามหาย) · ไม่มีแถวอยู่แล้ว = สำเร็จ
func ResetThemeService(ctx context.Context, actor agentAuthService.Actor, target assetCore.Target, req assetDto.GameRequest) error {
	if !target.Resolved {
		return errTargetNotResolved
	}
	if target.Scope != assetCore.ScopeAgent {
		return resetGlobalNotAllowed()
	}
	if err := requireGame(req.GameCode); err != nil {
		return err
	}
	return assetPostgres.DeleteGameThemeRepository(database.DBConn.WithContext(ctx), target.AgentID, req.GameCode)
}
