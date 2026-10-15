//go:build integration

package backoffice_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app/app/models"
	assetService "app/app/service/asset"
	"app/pkg/testutil"
	"app/platform/database"
)

func pngFile(t *testing.T, path string, w, h int) {
	t.Helper()
	writeFile(t, path, pngOf(t, w, h))
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSeedGlobal(t *testing.T) { // AST-61
	setup2(t)
	fake := testutil.NewFakeS3(t)
	dir := t.TempDir()
	pngFile(t, filepath.Join(dir, "scratch_card", "icon_button.png"), 32, 32)
	pngFile(t, filepath.Join(dir, "scratch_card", "win_highlight.png"), 400, 400) // ขนาดผิด (ต้อง 425×425)
	pngFile(t, filepath.Join(dir, "scratch_card", "nope.png"), 10, 10)            // ไม่มีช่องนี้ใน catalog
	pngFile(t, filepath.Join(dir, "coin_toss", "background.png"), 123, 45)        // ช่องไม่มีสเปกขนาด
	writeFile(t, filepath.Join(dir, "coin_toss", "winner.png"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	writeFile(t, filepath.Join(dir, "scratch_card", "readme.txt"), []byte("ไม่ใช่รูป — ต้องถูกข้าม"))
	writeFile(t, filepath.Join(dir, "top-level.png"), []byte("ไฟล์นอกโฟลเดอร์เกม — ต้องถูกข้าม"))

	var out bytes.Buffer
	by := assetService.SeedCreatedBy("tester")
	res, err := assetService.SeedGlobalService(context.Background(), dir, false, by, &out)
	if err != nil {
		t.Fatal(err)
	}
	// ไฟล์ที่ผิดทีละไฟล์ไม่หยุดทั้งชุด
	if res.Uploaded != 2 || res.Failed != 3 || res.Skipped != 0 {
		t.Fatalf("รอบแรก: %+v\n%s", res, out.String())
	}
	for _, want := range []string{"อัปแล้ว  scratch_card/icon_button", "อัปแล้ว  coin_toss/background", "405101", "405401", "405102"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("รายงานต้องมี %q:\n%s", want, out.String())
		}
	}
	if n := countRows(t, &models.AssetUsed{}, "agent_id IS NULL"); n != 2 {
		t.Fatalf("GLOBAL used ต้องมี 2 แถว: %d", n)
	}
	var cfgs []models.AssetConfig
	database.DBConn.Order("id").Find(&cfgs)
	for _, c := range cfgs {
		if c.Scope != models.AssetScopeGlobal || c.AgentID != nil || c.CreatedBy != "script:tester" {
			t.Errorf("config ต้องเป็น GLOBAL ที่ created_by = script:tester: %+v", c)
		}
	}
	if fake.Count() != 2 {
		t.Errorf("ต้องมี object ใน bucket 2 ไฟล์ (ไฟล์ที่ผิดไม่ขึ้น): %d", fake.Count())
	}
	// ช่องที่ยังไม่มีรูป = ทุกช่องของ 3 เกม (19 + 11 + 12) ลบที่อัปแล้ว 2
	if len(res.Gaps) != 19+11+12-2 {
		t.Errorf("ช่องที่ GLOBAL ยังไม่มีรูป: %d", len(res.Gaps))
	}
	for _, g := range res.Gaps {
		if (g.GameCode == "scratch_card" && g.AssetType == "icon_button") || (g.GameCode == "coin_toss" && g.AssetType == "background") {
			t.Errorf("ช่องที่อัปแล้วต้องไม่อยู่ในรายการที่ขาด: %+v", g)
		}
	}

	// รันซ้ำโดยไม่ใส่ -force = ข้ามช่องที่มี used แล้ว · ไม่มี config ใหม่
	out.Reset()
	res, err = assetService.SeedGlobalService(context.Background(), dir, false, by, &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 0 || res.Skipped != 2 || res.Failed != 3 {
		t.Fatalf("รอบสอง: %+v\n%s", res, out.String())
	}
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 2 {
		t.Errorf("config ต้องยัง 2 แถว: %d", n)
	}

	// -force = อัปเวอร์ชันใหม่ + ตั้ง used ทับ (แถว used ยังมี 1 แถวต่อช่อง) · ไฟล์เดิมยังอยู่ในคลัง
	out.Reset()
	res, err = assetService.SeedGlobalService(context.Background(), dir, true, by, &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 2 || res.Skipped != 0 {
		t.Fatalf("force: %+v\n%s", res, out.String())
	}
	if n := countRows(t, &models.AssetConfig{}, "is_deleted = false"); n != 4 {
		t.Errorf("config ต้อง 4 แถว (เก็บเวอร์ชันเดิม): %d", n)
	}
	if n := countRows(t, &models.AssetUsed{}, "agent_id IS NULL"); n != 2 {
		t.Errorf("GLOBAL used ต้องยัง 2 แถว: %d", n)
	}
}

func TestSeedGlobalWithoutStorage(t *testing.T) { // storage ไม่พร้อม = 505801 ต่อไฟล์ (ตัวสคริปต์หยุดตั้งแต่ต้นอยู่แล้ว) · ไม่มี config
	setup2(t)
	dir := t.TempDir()
	pngFile(t, filepath.Join(dir, "scratch_card", "icon_button.png"), 32, 32)
	var out bytes.Buffer
	res, err := assetService.SeedGlobalService(context.Background(), dir, false, assetService.SeedCreatedBy("tester"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 0 || res.Failed != 1 || !strings.Contains(out.String(), "505801") {
		t.Fatalf("%+v\n%s", res, out.String())
	}
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 0 {
		t.Errorf("ต้องไม่มี config: %d", n)
	}
}

func TestSeedCreatedBy(t *testing.T) {
	if got := assetService.SeedCreatedBy("tester"); got != "script:tester" {
		t.Errorf("got %q", got)
	}
	if got := assetService.SeedCreatedBy(strings.Repeat("x", 200)); len([]rune(got)) != 71 || !strings.HasPrefix(got, "script:") {
		t.Errorf("ต้องไม่เกิน 71 ตัวอักษร (VARCHAR(71)): %d", len([]rune(got)))
	}
}

// TestSeedGlobalRepoImages — รูปจริงใน seed/global-assets ต้องผ่านการตรวจเดียวกับ API ทุกไฟล์
// ถ้ามีรูปผิดขนาด / ชื่อไม่ตรงช่อง / ชนิดไม่ถูก CI แดงที่นี่ ไม่ใช่ตอนรัน script บน env จริง · AST-61
func TestSeedGlobalRepoImages(t *testing.T) {
	setup2(t)
	fake := testutil.NewFakeS3(t)
	dir := filepath.Join("..", "..", "..", "seed", "global-assets")

	var out bytes.Buffer
	by := assetService.SeedCreatedBy("tester")
	res, err := assetService.SeedGlobalService(context.Background(), dir, false, by, &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 || res.Skipped != 0 || res.Uploaded != 19 {
		t.Fatalf("รูปใน seed/global-assets ต้องอัปได้ครบ 19 ไฟล์: %+v\n%s", res, out.String())
	}
	if fake.Count() != 19 {
		t.Errorf("ต้องมี object ใน bucket 19 ไฟล์: %d", fake.Count())
	}
	for _, g := range res.Gaps {
		if g.GameCode == "scratch_card" {
			t.Errorf("scratch_card ต้องไม่มีช่องว่าง: %+v", g)
		}
	}
	if n := countRows(t, &models.AssetUsed{}, "agent_id IS NULL"); n != 19 {
		t.Errorf("GLOBAL used ต้องมี 19 แถว: %d", n)
	}

	// รันซ้ำ = ข้ามทุกช่อง ไม่มีอะไรเพิ่ม
	out.Reset()
	res, err = assetService.SeedGlobalService(context.Background(), dir, false, by, &out)
	if err != nil {
		t.Fatal(err)
	}
	if res.Uploaded != 0 || res.Skipped != 19 || res.Failed != 0 {
		t.Fatalf("รันซ้ำต้องข้ามครบ: %+v\n%s", res, out.String())
	}
	if n := countRows(t, &models.AssetConfig{}, "1 = 1"); n != 19 {
		t.Errorf("config ต้องยัง 19 แถว: %d", n)
	}
}
