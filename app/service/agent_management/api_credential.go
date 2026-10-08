package agentmanagement

import (
	"errors"

	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// CreateAPICredentialIfAbsent — สร้าง Key ของ account 1.3 ให้เจ้าของถ้ายังไม่มี (account ACC-03 – ACC-05)
// เรียกพร้อมกันได้ Key เดียว (agent_id เป็น PK) · ใช้ทั้งตอนสร้างบัญชีเจ้าของ (MGMT-04) และตอนเปิดหน้า 1.3 ครั้งแรก
// อยู่ใน module นี้เพราะ account import module นี้อยู่แล้ว (กัน import วน)
func CreateAPICredentialIfAbsent(db *gorm.DB, agentID uint) error {
	encKey := configs.Cfg.Account.APIKeyEncryptionKey
	if len(encKey) == 0 { // local ที่ไม่ได้ตั้ง API_KEY_ENCRYPTION_KEY
		return apperr.ErrUnavailable.Wrap(errors.New("API_KEY_ENCRYPTION_KEY is not set"))
	}
	key, err := utils.GenerateAPIKey()
	if err != nil {
		return err
	}
	ciphertext, err := utils.EncryptAPIKey(encKey, key)
	if err != nil {
		return err
	}
	return postgres.CreateAPICredentialIfAbsentRepository(db, &models.APICredential{
		AgentID: agentID, KeyCiphertext: ciphertext, KeyHash: utils.HashAPIKey(key)})
}
