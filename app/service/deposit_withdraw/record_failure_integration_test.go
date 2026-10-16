//go:build integration

// C-S1 (lead review รอบ 3): แถว FAILED ต้องไม่ถูกเขียนเมื่อรายการ commit สำเร็จไปแล้ว — DW-36
// จำลอง COMMIT ผลไม่แน่ชัดผ่าน API ไม่ได้ → เรียก recordFailure ตรง (export_test.go) กับ create_requests ที่มี / ไม่มี / กำลัง commit
package depositwithdraw_test

import (
	"context"
	"testing"
	"time"

	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
	depositWithdrawService "app/app/service/deposit_withdraw"
	"app/pkg/apperr"
	"app/pkg/testutil"
	"app/platform/database"

	"gorm.io/gorm"
)

func TestRecordFailureSkipsCommitted(t *testing.T) {
	testutil.Setup(t)
	db := database.DBConn
	parent := models.UserAgent{Username: "agentcs1", PasswordHash: "x", Role: models.AgentRoleAgent, Status: models.AgentStatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	mem := models.UserMember{AgentID: parent.ID, Username: "memcs1", PasswordHash: "x", Name: "Namecs1", Currency: "THB",
		Credit: 50, Status: models.AgentStatusActive, PaymentType: models.PaymentTypeCash}
	if err := db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}
	actor := agentAuthService.Actor{AccountType: models.AccountTypeAgent, AgentID: parent.ID, Role: models.AgentRoleAgent, Username: parent.Username}
	record := func(requestID string) {
		req := depositWithdrawDto.CreateRequest{RequestID: requestID, Role: "MEMBER", UserID: mem.ID, Currency: "THB", Amount: 10,
			TransactionType: "DEPOSIT", Owner: models.BalanceOwnerMember, Type: models.DWDeposit}
		depositWithdrawService.RecordFailureForTest(context.Background(), parent, models.BalanceOwnerMember, mem.ID, "MEMBER", mem.Username,
			actor, req, agentAuthService.RequestMeta{IP: "203.0.113.10"}, apperr.ErrInternal.MsgTH)
	}
	count := func(requestID string) int64 {
		var n int64
		db.Model(&models.DepositWithdrawLog{}).Where("request_id = ? AND status = ?", requestID, models.DWStatusFailed).Count(&n)
		return n
	}
	register := func(tx *gorm.DB, requestID string) {
		if err := tx.Create(&models.CreateRequest{RequestID: requestID, CreatorType: models.AccountTypeAgent, CreatorID: parent.ID,
			TargetType: models.CreateRequestTargetDW, TargetID: 1}).Error; err != nil {
			t.Error(err)
		}
	}

	// 1) commit ไปแล้ว (create_requests มี) = ไม่เขียน FAILED
	const committed = "00000000-0000-4000-8000-0000000c5001"
	register(db, committed)
	record(committed)
	if n := count(committed); n != 0 {
		t.Fatalf("committed: FAILED %d แถว", n)
	}

	// 2) ไม่มี create_requests = เขียน FAILED · ยอด ณ ตอนบันทึก · note ข้อความกลางของระบบล้ม
	const failed = "00000000-0000-4000-8000-0000000c5002"
	record(failed)
	var l models.DepositWithdrawLog
	if err := db.Where("request_id = ?", failed).Take(&l).Error; err != nil {
		t.Fatal(err)
	}
	if l.Status != models.DWStatusFailed || l.TargetBalanceBefore != 50 || l.TargetBalanceAfter != 50 || l.Note != "ล้มเหลว: "+apperr.ErrInternal.MsgTH || l.IP != "203.0.113.10" {
		t.Fatalf("FAILED row = %+v", l)
	}

	// 3) tx หลักยังถือ advisory lock และ commit ทีหลัง — recordFailure ต้องรอแล้วเห็นว่า commit แล้ว
	const inFlight = "00000000-0000-4000-8000-0000000c5003"
	locked := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "create_request:"+inFlight).Error; err != nil {
				t.Error(err)
				return err
			}
			register(tx, inFlight)
			close(locked)
			time.Sleep(500 * time.Millisecond) // commit ช้า (เทียบกับ COMMIT ที่ client timeout ไปก่อน)
			return nil
		})
	}()
	<-locked
	record(inFlight)
	<-done
	if n := count(inFlight); n != 0 {
		t.Fatalf("in-flight commit: FAILED %d แถว", n)
	}
}
