package account

import (
	"errors"

	"app/app/models"
	"app/app/repository/postgres"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// createAPICredentialIfAbsent — สร้าง Key ของ 1.3 ให้เจ้าของถ้ายังไม่มี (ACC-03 – ACC-05)
// เรียกพร้อมกันได้ Key เดียว (agent_id เป็น PK) · สร้างพร้อมบัญชีเจ้าของ (module ② MGMT-04) ทำหลัง module ② และ account เข้า dev
func createAPICredentialIfAbsent(db *gorm.DB, agentID uint) error {
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
