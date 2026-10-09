//go:build integration

// test ของ migration ข้อมูลใน agent_management (lead V5 / V2 / V7 2026-10-10)
// ถอย DB test กลับไปก่อน migration → ใส่ข้อมูลแบบเก่าด้วย SQL → รัน migration → ตรวจผล · จบแล้ว up กลับเสมอ
package backoffice_test

import (
	"context"
	"testing"

	"app/app/models"
	"app/database/migrations"
	"app/platform/database"

	"github.com/pressly/goose/v3"
)

const (
	migPermissionsDropPT = 20261009150000 // ก่อนเบอร์ 2 field
	migPhone             = 20261009160000 // MGMT-08A
	migFollowCSM         = 20261009170000 // MGMT-19
)

// migrateDownTo — ถอย schema ไปที่ version · จบ test แล้ว up ทั้งหมดคืน (test อื่นใช้ schema ล่าสุด · cleanup นี้ทำก่อนล้างตารางของ Setup)
func migrateDownTo(t *testing.T, version int64) {
	t.Helper()
	sqlDB, err := database.DBConn.DB()
	if err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := goose.UpContext(context.Background(), sqlDB, "."); err != nil {
			t.Fatalf("migrate up คืน: %v", err)
		}
	})
	if err := goose.DownToContext(context.Background(), sqlDB, ".", version); err != nil {
		t.Fatalf("down to %d: %v", version, err)
	}
}

func gooseUpTo(t *testing.T, version int64) {
	t.Helper()
	sqlDB, err := database.DBConn.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := goose.UpToContext(context.Background(), sqlDB, ".", version); err != nil {
		t.Fatalf("up to %d: %v", version, err)
	}
}

func execSQL(t *testing.T, sql string, args ...any) {
	t.Helper()
	if err := database.DBConn.Exec(sql, args...).Error; err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

type phoneRow struct {
	Code  *string `gorm:"column:phone_country_code"`
	Phone *string `gorm:"column:phone"`
}

func TestMigrationPhone(t *testing.T) { // MGMT-08A · lead B1 / B6 / V7
	app := setup2(t)
	c := buildChain(t, app)
	var ids []uint
	for _, n := range []string{"ph1", "ph2", "ph3", "ph4", "ph5", "ph6"} {
		ids = append(ids, createAccount(t, n, models.AgentRoleAgent, &c.agent.ID).ID)
	}
	var subIDs []uint
	for _, n := range []string{"sub1", "sub2", "sub3"} {
		s := models.Subaccount{AgentID: c.agent.ID, Username: "agent01@" + n, PasswordHash: "x", Status: models.AgentStatusActive}
		if err := database.DBConn.Create(&s).Error; err != nil {
			t.Fatal(err)
		}
		subIDs = append(subIDs, s.ID)
	}

	migrateDownTo(t, migPermissionsDropPT)
	// ข้อมูลแบบเก่า (เบอร์เดียว 8–15 หลัก) — id ของ ph2 มากกว่า ph1 · ph2 แปลงแล้วซ้ำกับ ph1
	old := []string{"0812345678", "66812345678", "021234567", "12345678", "0012345678", "6681234567"}
	for i, p := range old {
		execSQL(t, "UPDATE user_agents SET phone = ? WHERE id = ?", p, ids[i])
	}
	for i, p := range []string{"0899999999", "0899999999", "123456789"} {
		execSQL(t, "UPDATE subaccounts SET phone = ? WHERE id = ?", p, subIDs[i])
	}
	gooseUpTo(t, migPhone)

	str := func(s string) *string { return &s }
	want := []phoneRow{
		{str("66"), str("812345678")}, // 0 → 66 + ตัด 0
		{nil, nil},                    // 66 + 11 หลัก → ซ้ำกับ ph1 (id น้อยกว่า) → ล้าง
		{str("66"), str("21234567")},  // เบอร์บ้าน 0 นำหน้า
		{nil, nil},                    // ไม่ขึ้นต้น 0 / 66 → ล้าง
		{nil, nil},                    // 00… → ล้าง
		{nil, nil},                    // 66 แต่ไม่ครบ 11 หลัก → ล้าง
	}
	eq := func(a, b *string) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
	for i, id := range ids {
		var got phoneRow
		database.DBConn.Table("user_agents").Select("phone_country_code, phone").Where("id = ?", id).Scan(&got)
		if !eq(got.Code, want[i].Code) || !eq(got.Phone, want[i].Phone) {
			t.Fatalf("%s → %v %v", old[i], got.Code, got.Phone)
		}
	}
	for i, wantPhone := range []*string{str("899999999"), str("899999999"), nil} { // sub ซ้ำได้ (MGMT-41 · V4)
		var got phoneRow
		database.DBConn.Table("subaccounts").Select("phone_country_code, phone").Where("id = ?", subIDs[i]).Scan(&got)
		if !eq(got.Phone, wantPhone) || (wantPhone != nil && (got.Code == nil || *got.Code != "66")) {
			t.Fatalf("sub %d → %v %v", i, got.Code, got.Phone)
		}
	}

	type cleared struct {
		TableName   string `gorm:"column:table_name"`
		RowID       uint   `gorm:"column:row_id"`
		OldPhone    string `gorm:"column:old_phone"`
		Reason      string `gorm:"column:reason"`
		KeptAgentID *uint  `gorm:"column:kept_agent_id"`
	}
	var logs []cleared
	database.DBConn.Table("migration_phone_cleared").Order("table_name, row_id").Find(&logs)
	wantLogs := []cleared{
		{"subaccounts", subIDs[2], "123456789", "FORMAT", nil},
		{"user_agents", ids[1], "66812345678", "DUPLICATE", &ids[0]},
		{"user_agents", ids[3], "12345678", "FORMAT", nil},
		{"user_agents", ids[4], "0012345678", "FORMAT", nil},
		{"user_agents", ids[5], "6681234567", "FORMAT", nil},
	}
	if len(logs) != len(wantLogs) {
		t.Fatalf("migration_phone_cleared %+v", logs)
	}
	for i, w := range wantLogs {
		g := logs[i]
		if g.TableName != w.TableName || g.RowID != w.RowID || g.OldPhone != w.OldPhone || g.Reason != w.Reason ||
			(w.KeptAgentID == nil) != (g.KeptAgentID == nil) || (w.KeptAgentID != nil && *g.KeptAgentID != *w.KeptAgentID) {
			t.Fatalf("log %d = %+v want %+v", i, g, w)
		}
	}
}

func TestMigrationShareMasterFollowCSM(t *testing.T) { // MGMT-19 · lead V2
	app := setup2(t)
	k := buildCSMChain(t, app)                                 // CSM 80 · sm1 (Member pt 50) · sm2
	mem2 := createMemberWithPT(t, app, k.sm2Tok, "memsm2", 20) // Member ใต้ sm2 pt 20

	migrateDownTo(t, migPhone)
	// ข้อมูลก่อนกฎใหม่: Share Master ค่าไม่ตรง CSM · CSM ได้รับ 45 (ต่ำกว่า pt 50 ของ Member ใต้ sm1)
	execSQL(t, "UPDATE agent_game_settings SET pt_from_parent = 45 WHERE agent_id = ?", k.csm.ID)
	execSQL(t, "UPDATE agent_game_settings SET status_game = false WHERE agent_id = ? AND game_code = 'scratch_card'", k.csm.ID)
	execSQL(t, "UPDATE agent_game_settings SET pt_from_parent = 70, force = 5, remain = 3, status = false WHERE agent_id = ?", k.sm1.ID)
	execSQL(t, "UPDATE user_member_game_settings SET remain = 1 WHERE user_member_id = ?", mem2.ID)
	gooseUpTo(t, migFollowCSM)

	for id, comm := range map[uint]float64{k.sm1.ID: 0.2, k.sm2.ID: 0.3} {
		for _, s := range settingsOf(t, id) {
			if s.PTFromParent != 45 || s.Force != 0 || s.Remain != 0 || !s.Status || s.Commission != comm || s.StatusGame != (s.GameCode != "scratch_card") {
				t.Fatalf("share master %d %+v", id, s)
			}
		}
	}
	// Member ใต้ sm2: remain = 45 − 20 · Member ใต้ sm1 (pt 50 > 45): ไม่แก้ + บันทึก
	var rows []models.UserMemberGameSetting
	database.DBConn.Where("user_member_id = ?", mem2.ID).Find(&rows)
	for _, s := range rows {
		if s.Remain != 25 {
			t.Fatalf("member sm2 %+v", s)
		}
	}
	if s := memberSetting(t, k.mem.ID); s.PT != 50 || s.Remain != 30 {
		t.Fatalf("member sm1 ต้องไม่ถูกแก้ %+v", s)
	}
	type skipped struct {
		UserMemberID    uint    `gorm:"column:user_member_id"`
		GameCode        string  `gorm:"column:game_code"`
		ShareMasterID   uint    `gorm:"column:share_master_id"`
		MemberPT        float64 `gorm:"column:member_pt"`
		NewPTFromParent float64 `gorm:"column:new_pt_from_parent"`
		Remain          float64 `gorm:"column:remain"`
	}
	var sk []skipped
	database.DBConn.Table("migration_member_remain_skipped").Order("game_code").Find(&sk)
	if len(sk) != 3 {
		t.Fatalf("skipped %+v", sk)
	}
	for _, s := range sk {
		if s.UserMemberID != k.mem.ID || s.ShareMasterID != k.sm1.ID || s.MemberPT != 50 || s.NewPTFromParent != 45 || s.Remain != 30 {
			t.Fatalf("skipped %+v", s)
		}
	}
}
