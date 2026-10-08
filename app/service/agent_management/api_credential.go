package agentmanagement

import (
	"app/app/models"
	accountPostgres "app/app/repository/postgres/account"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// CreateAPICredentialIfAbsent — สร้าง Key ของ account 1.3 ให้เจ้าของถ้ายังไม่มี (account ACC-03 – ACC-05)
// เก็บ Key ตรงๆ (ACC-04 แก้ 2026-10-08) · เรียกพร้อมกันได้ Key เดียว (agent_id เป็น PK)
// ใช้ทั้งตอนสร้างบัญชีเจ้าของ (MGMT-04) และตอนเปิดหน้า 1.3 ครั้งแรก
// อยู่ใน module นี้เพราะ account import module นี้อยู่แล้ว (กัน import วน)
func CreateAPICredentialIfAbsent(db *gorm.DB, agentID uint) error {
	key, err := utils.GenerateAPIKey()
	if err != nil {
		return err
	}
	return accountPostgres.CreateAPICredentialIfAbsentRepository(db, &models.APICredential{AgentID: agentID, APIKey: key})
}
