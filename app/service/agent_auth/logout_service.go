package agentauth

import (
	"context"

	redisRepo "app/app/repository/redis"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
)

// LogoutService — POST /api/v1/bo/pr/auth/logout · rule: AUTH-09
//
// ตรวจแค่ลายเซ็น/อายุของ token (ไม่ผ่าน AuthenticateService) เพื่อให้ logout ด้วย session ที่หลุดไปแล้ว
// ยังตอบสำเร็จได้ — token ปลอม/หมดอายุ absolute แล้ว ตอบ ErrInvalidToken
func LogoutService(ctx context.Context, rawToken string) error {
	claims, err := utils.ParseSessionToken(configs.Cfg.Auth.JWTSecret, utils.TokenTypeBackoffice, rawToken)
	if err != nil {
		return apperr.ErrInvalidToken.Wrap(err)
	}
	return redisRepo.DeleteBOSessionRepository(ctx, claims.SessionID, claims.AgentID)
}
