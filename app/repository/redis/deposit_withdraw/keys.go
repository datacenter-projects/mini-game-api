package depositwithdraw

import (
	"fmt"

	"app/app/models"
)

// key ทั้งหมดของ module deposit_withdraw ประกาศที่นี่ที่เดียว · prefix: dw:

// keyDWLock — lock รายการฝาก / ถอนที่บัญชีปลายทาง: dw:lock:AGENT:{id} / dw:lock:MEMBER:{id} (docs/modules/deposit_withdraw.md DW-20)
func keyDWLock(owner models.BalanceOwnerType, id uint) string {
	return fmt.Sprintf("dw:lock:%s:%d", owner, id)
}
