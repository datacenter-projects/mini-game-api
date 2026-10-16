package depositwithdraw

import (
	"context"

	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
)

// RecordFailureForTest — เรียก recordFailure ตรงจาก test (C-S1 จำลอง COMMIT ผลไม่แน่ชัดผ่าน API ไม่ได้) · คอมไพล์เฉพาะตอน test
func RecordFailureForTest(ctx context.Context, parent models.UserAgent, owner models.BalanceOwnerType, targetID uint, role, username string,
	actor agentAuthService.Actor, req depositWithdrawDto.CreateRequest, meta agentAuthService.RequestMeta, reason string) {
	recordFailure(ctx, attempt{found: true, parent: parent, target: target{owner: owner, id: targetID, role: role, username: username}},
		actor, req, meta, reason)
}
