package agentauth

import (
	"context"
	"errors"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
)

// loadResetTarget หาเป้าหมายและตรวจว่า admin รีเซ็ตได้ (AUTH-45)
func loadResetTarget(ctx context.Context, actor Actor, rawUsername string) (account, error) {
	username := agentAuthCore.NormalizeUsername(rawUsername)
	if agentAuthCore.IsSubaccountUsername(username) {
		if _, _, ok := agentAuthCore.SplitSubaccountUsername(username); !ok {
			return account{}, apperr.ErrResetTargetNotFound
		}
	}

	db := database.DBConn.WithContext(ctx)
	target, err := loadAccountForLogin(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		return account{}, apperr.ErrResetTargetNotFound
	}
	if err != nil {
		return account{}, err
	}

	isSelf := target.Type == actor.AccountType && target.ID == actor.AccountID()
	if isSelf || (!target.IsSub() && !agentAuthCore.IsResettableRole(target.Role)) {
		return account{}, apperr.ErrResetNotAllowed
	}

	if target.Status == models.AgentStatusLocked {
		return account{}, apperr.ErrResetTargetLocked
	}
	if target, err = withUplineStatus(db, target); err != nil {
		return account{}, err
	}
	if target.UplineLocked() {
		return account{}, apperr.ErrResetTargetLocked
	}
	return target, nil
}

// newTempPassword สุ่มรหัสชั่วคราวที่ผ่าน AUTH-36 และไม่ซ้ำกับรหัสปัจจุบัน/ก่อนหน้า (AUTH-46, AUTH-48)
func newTempPassword(acc account) (string, error) {
	for {
		p, err := utils.RandomString(agentAuthCore.TempPasswordAlphabet, agentAuthCore.TempPasswordLength)
		if err != nil {
			return "", err
		}
		if agentAuthCore.CheckPasswordPolicy(p) == agentAuthCore.PasswordOK && !isRecentPassword(acc, p) {
			return p, nil
		}
	}
}
