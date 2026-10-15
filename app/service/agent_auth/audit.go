package agentauth

import (
	"context"
	"strings"
	"time"

	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// RequestMeta — ข้อมูลของ HTTP request ที่ service ต้องใช้ (controller/middleware สร้างจาก fiber.Ctx)
type RequestMeta struct {
	IP        string
	UserAgent string
}

const maxUserAgentLen = 255

// auditEntry — เหตุการณ์ 1 รายการของ auth_audit_logs (AUTH-49)
// ห้ามใส่รหัสผ่าน, passcode, token หรือค่าชั่วคราวลงใน field ใด
type auditEntry struct {
	Action        models.AuthAuditAction
	ActorType     models.AuthAuditActorType
	ActorID       uint // 0 = ไม่ใช่บัญชี (SCRIPT / SYSTEM)
	ActorUsername string
	Target        *account // nil = ไม่มีบัญชีนี้ในระบบ
	TargetName    string   // ใช้เมื่อ Target = nil
}

// actorEntry — ผู้ทำคือบัญชีที่ login อยู่
func actorEntry(action models.AuthAuditAction, actor Actor, target account) auditEntry {
	actorType := models.AuthAuditActorAgent
	if actor.AccountType == models.AccountTypeSub {
		actorType = models.AuthAuditActorSub
	}
	return auditEntry{Action: action, ActorType: actorType, ActorID: actor.AccountID(), ActorUsername: actor.Username, Target: &target}
}

// writeAudit บันทึกเหตุการณ์ด้วย db ที่ส่งมา — ส่ง tx เพื่อให้อยู่ใน transaction เดียวกับการเปลี่ยนข้อมูล
func writeAudit(ctx context.Context, db *gorm.DB, e auditEntry, meta RequestMeta) error {
	l := models.AuthAuditLog{
		Action:    e.Action,
		ActorType: e.ActorType,
		IP:        optional(meta.IP),
		UserAgent: optional(truncate(meta.UserAgent, maxUserAgentLen)),
		RequestID: optional(logger.RequestID(ctx)),
		CreatedAt: time.Now(),
	}
	if e.ActorID != 0 {
		l.ActorID = &e.ActorID
	}
	l.ActorUsername = optional(e.ActorUsername)
	if e.Target != nil {
		t, id := e.Target.Type, e.Target.ID
		l.TargetType, l.TargetID, l.TargetUsername = &t, &id, e.Target.Username
	} else {
		l.TargetUsername = e.TargetName
	}
	return agentAuthPostgres.CreateAuthAuditLogRepository(db, &l)
}

// writeAuditOutsideTx — เหตุการณ์ที่ไม่ได้อยู่ใน transaction (บล็อก) · บันทึกไม่ได้ไม่ทำให้ request ล้ม
func writeAuditOutsideTx(ctx context.Context, e auditEntry, meta RequestMeta) {
	if err := writeAudit(ctx, database.DBConn.WithContext(ctx), e, meta); err != nil {
		logger.Ctx(ctx).Warnw("write auth audit log failed", "action", e.Action, "error", err)
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// truncate ตัดตามจำนวนตัวอักษร (VARCHAR นับตัวอักษร) และทิ้ง byte ที่ไม่ใช่ UTF-8 (Postgres ไม่รับ)
func truncate(s string, n int) string {
	r := []rune(strings.ToValidUTF8(s, ""))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}
