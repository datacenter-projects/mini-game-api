package logger

import (
	"context"

	"go.uber.org/zap"
)

var (
	Logger      *zap.Logger
	SugarLogger *zap.SugaredLogger
)

type ctxKey struct{}

// InitLogger — local ใช้ development format อ่านง่าย, env อื่นใช้ JSON production
func InitLogger(isLocal bool) {
	var err error
	if isLocal {
		Logger, err = zap.NewDevelopment()
	} else {
		Logger, err = zap.NewProduction()
	}
	if err != nil {
		panic(err)
	}
	SugarLogger = Logger.Sugar()
}

// WithRequestID ผูก request_id เข้า ctx — middleware เรียกให้ทุก request แล้ว ไม่ต้องเรียกเอง
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, requestID)
}

// RequestID อ่าน request_id จาก ctx (ว่าง = ไม่ได้มาจาก HTTP request เช่น cron)
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// Ctx คืน logger ที่แนบ request_id ไว้แล้ว — ใช้ใน service/repository:
//
//	logger.Ctx(ctx).Errorw("lock balance failed", "agent_id", id, "error", err)
func Ctx(ctx context.Context) *zap.SugaredLogger {
	if SugarLogger == nil { // ยังไม่ InitLogger (เช่นใน unit test) — ไม่ให้ panic
		return zap.NewNop().Sugar()
	}
	if rid := RequestID(ctx); rid != "" {
		return SugarLogger.With("request_id", rid)
	}
	return SugarLogger
}
