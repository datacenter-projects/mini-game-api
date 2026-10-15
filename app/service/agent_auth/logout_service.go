package agentauth

import (
	"context"

	agentAuthRedis "app/app/repository/redis/agent_auth"
)

// LogoutService — POST /api/v1/bo/pr/auth/logout · rule: AUTH-09, AUTH-23
//
// ตรวจแค่ลายเซ็น/อายุของ token (ไม่ผ่าน AuthenticateService) เพื่อให้ logout ด้วย session ที่หลุดไปแล้ว
// ยังตอบสำเร็จได้ — token ปลอม/หมดอายุ absolute แล้ว ตอบ ErrInvalidToken
// pointer ของ session อ่านจาก account_type + account_id ใน token (sub ต้องไม่ไปลบ pointer ของผู้สร้าง)
func LogoutService(ctx context.Context, rawToken string) error {
	claims, accountType, err := parseToken(rawToken)
	if err != nil {
		return err
	}
	return agentAuthRedis.DeleteBOSessionRepository(ctx, claims.SessionID, accountType, claims.AccountID)
}
