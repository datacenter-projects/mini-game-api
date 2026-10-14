package gamerate

import (
	"context"
	"errors"
	"sync"

	"app/app/models"
	gameRatePostgres "app/app/repository/postgres/game_rate"
	"app/platform/database"

	"gorm.io/gorm"
)

// ฟังก์ชันสำหรับ engine / module เดิมพัน (spec หัวข้อ 5.8 · lead C6) — ไม่ใช่ endpoint

// ErrRateNotConfigured — เกมยังไม่เคยตั้งอัตรา (version 0 — RATE-09A) · engine ต้องปฏิเสธ Bet (RH1)
// error ภายใน ไม่ส่งถึง client ตรงๆ — module เดิมพันแปลงเป็น error ของตัวเอง
var ErrRateNotConfigured = errors.New("game_rate: game has no rate version")

// ErrRateVersionMissing — ไม่พบ rate_version_id ที่เดิมพันอ้าง (ข้อมูลเสีย)
var ErrRateVersionMissing = errors.New("game_rate: rate version not found")

// CurrentVersionService — ค่าของเวอร์ชันปัจจุบันของเกม ใช้ตอนเริ่มเดิมพันใหม่ (RATE-06) · ยังไม่เคยตั้ง = ErrRateNotConfigured
// รับ tx ของ module เดิมพันเพื่ออ่านใน tx เดียวกับการสร้าง Bet (เก็บ VersionID เป็น rate_version_id)
func CurrentVersionService(ctx context.Context, tx *gorm.DB, gameCode string) (Rates, error) {
	db := tx.WithContext(ctx)
	v, err := gameRatePostgres.GetLatestVersionRepository(db, gameCode)
	if err != nil {
		return Rates{}, err
	}
	if v.ID == 0 {
		return Rates{}, ErrRateNotConfigured
	}
	rates, err := loadRates(db, []models.GameRateVersion{v})
	if err != nil {
		return Rates{}, err
	}
	return rates[v.ID], nil
}

// rateCache — ค่าตาม version_id · เวอร์ชันแก้ / ลบไม่ได้ (RATE-05) จึงไม่ต้องลบรายตัว · ไม่ใช่ source of truth ของเงิน (กฎข้อ 14)
// จำกัดจำนวน: เกิน rateCacheMax ล้างทั้งก้อนแล้วเริ่มใหม่ (เวอร์ชันที่ใช้จริงมีไม่กี่ตัว — กัน memory โตไม่จำกัด)
const rateCacheMax = 256

var rateCache = struct {
	sync.RWMutex
	byID map[uint]Rates
}{byID: map[uint]Rates{}}

func cachedRates(id uint) (Rates, bool) {
	rateCache.RLock()
	defer rateCache.RUnlock()
	r, ok := rateCache.byID[id]
	return r, ok
}

func storeRates(r Rates) {
	rateCache.Lock()
	defer rateCache.Unlock()
	if len(rateCache.byID) >= rateCacheMax {
		rateCache.byID = map[uint]Rates{}
	}
	rateCache.byID[r.VersionID] = r.clone()
}

// ByVersionIDService — ค่าของเวอร์ชันที่เดิมพันอ้าง (Streak ทุกขั้นอ่านจาก id เดิม — RATE-06 / RH1) · cache ใน memory
func ByVersionIDService(ctx context.Context, versionID uint) (Rates, error) {
	if r, ok := cachedRates(versionID); ok {
		return r.clone(), nil // สำเนา — ผู้เรียกแก้ slice แล้วไม่กระทบ cache
	}
	db := database.DBConn.WithContext(ctx)
	v, err := gameRatePostgres.GetVersionByIDRepository(db, versionID)
	if err != nil {
		return Rates{}, err
	}
	if v.ID == 0 {
		return Rates{}, ErrRateVersionMissing
	}
	rates, err := loadRates(db, []models.GameRateVersion{v})
	if err != nil {
		return Rates{}, err
	}
	storeRates(rates[v.ID])
	return rates[v.ID], nil
}
