//go:build integration

// ยิงพร้อมกันของเส้น Member (lead ตรวจรอบที่ 1 2026-10-11 — งานตามหลัง): ไม่ได้ 500 / deadlock · ซ้ำได้ error ของ field
// และ unique violation ที่หลุดการเช็คล่วงหน้า (แถวที่ยังไม่ commit) ตอบ 402401 / 402403 แทน 409
package backoffice_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"app/app/models"
	"app/pkg/utils"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

type memberJob struct {
	path string
	body map[string]any
	tok  string
}

// runJobs — ยิงทุก job พร้อมกัน คืน code ตามลำดับ job
func runJobs(app *fiber.App, jobs []memberJob) []int {
	codes := make([]int, len(jobs))
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j memberJob) {
			defer wg.Done()
			codes[i] = callCode(app, j.path, j.body, j.tok)
		}(i, j)
	}
	wg.Wait()
	return codes
}

// countCodes — นับ code · code ที่ไม่อยู่ใน allowed = fail
func countCodes(t *testing.T, codes []int, allowed ...int) map[int]int {
	t.Helper()
	ok := map[int]bool{}
	for _, c := range allowed {
		ok[c] = true
	}
	out := map[int]int{}
	for i, c := range codes {
		if !ok[c] {
			t.Fatalf("job %d ได้ code %d (ทั้งหมด %v)", i, c, codes)
		}
		out[c]++
	}
	return out
}

func TestMemberCreateConcurrent(t *testing.T) { // MGMT-05 · MGMT-08 · MGMT-15A
	app := setup2(t)
	c := buildChain(t, app)
	const n = 8

	// เบอร์เดียวกันใต้ผู้สร้างเดียวกัน: สำเร็จ 1 ที่เหลือ 402403
	var jobs []memberJob
	for i := 0; i < n; i++ {
		b := memberBody(fmt.Sprintf("memphrace%d", i), 0)
		b["phone_country_code"], b["phone"] = "66", "811111111"
		jobs = append(jobs, memberJob{createMemberPath, b, c.agentTok})
	}
	if got := countCodes(t, runJobs(app, jobs), 200, 402403); got[200] != 1 {
		t.Fatalf("เบอร์ซ้ำพร้อมกัน %v", got)
	}

	// username เดียวกัน: สำเร็จ 1 ที่เหลือ 402401
	jobs = nil
	for i := 0; i < n; i++ {
		jobs = append(jobs, memberJob{createMemberPath, memberBody("memuserrace", 0), c.agentTok})
	}
	if got := countCodes(t, runJobs(app, jobs), 200, 402401); got[200] != 1 {
		t.Fatalf("username ซ้ำพร้อมกัน %v", got)
	}

	// request_id เดียวกันพร้อมยอดตั้งต้น: ได้ id เดียวกันทุกครั้ง · โอนครั้งเดียว · ledger 2 แถว
	setBalance(t, c.agent.ID, "THB", 100)
	rich := memberBody("memridrace", 0)
	rich["balance"] = map[string]any{"THB": 10}
	jobs = nil
	for i := 0; i < n; i++ {
		jobs = append(jobs, memberJob{createMemberPath, rich, c.agentTok})
	}
	countCodes(t, runJobs(app, jobs), 200)
	if countRows(t, &models.UserMember{}, "username = ?", "memridrace") != 1 || agentBalance(t, c.agent.ID, "THB") != 90 ||
		countRows(t, &models.BalanceLedger{}, "request_id = ?", rich["request_id"]) != 2 {
		t.Fatalf("request_id ซ้ำพร้อมกัน: ยอดผู้สร้าง %v", agentBalance(t, c.agent.ID, "THB"))
	}
}

func TestMemberUpdateConcurrent(t *testing.T) { // MGMT-08 · MGMT-21 · MGMT-24 R2 · MGMT-30: ไม่ deadlock · remain_quota ตรงหลังจบ
	app := setup2(t)
	c := buildChain(t, app) // agent01 ได้รับ 60
	var ids []uint
	for i := 0; i < 6; i++ {
		ids = append(ids, createMemberWithPT(t, app, c.agentTok, fmt.Sprintf("memupdrace%d", i), 10).ID)
	}

	// เปลี่ยนเป็นเบอร์เดียวกันพร้อมกัน: สำเร็จ 1 ที่เหลือ 402403
	var jobs []memberJob
	for _, id := range ids {
		jobs = append(jobs, memberJob{memberDetailUpdatePath, map[string]any{"id": id,
			"info": map[string]any{"phone_country_code": "66", "phone": "822222222"}}, c.agentTok})
	}
	if got := countCodes(t, runJobs(app, jobs), 200, 402403); got[200] != 1 {
		t.Fatalf("แก้เบอร์ซ้ำพร้อมกัน %v", got)
	}

	// แก้ pt ของ Member · ชั้นบนแก้ค่าที่ให้ agent01 (sync remain_quota) · สร้าง Member · แก้สถานะ — พร้อมกัน
	jobs = nil
	for i := 0; i < 6; i++ {
		jobs = append(jobs,
			memberJob{memberDetailUpdatePath, map[string]any{"id": ids[i], "info": map[string]any{"name": fmt.Sprintf("ชื่อ%d", i)},
				"pt": map[string]any{"minigame": map[string]any{"pt": float64(i * 5), "commission_percent": 0.1}}}, c.agentTok},
			memberJob{agentDetailUpdatePath, ptBody(c.agent.ID, []float64{60, 55, 50}[i%3], 0, 0, 0.5), c.shareTok},
			memberJob{createMemberPath, func() map[string]any {
				b := memberBody(fmt.Sprintf("memnewrace%d", i), 0)
				b["pt"].(map[string]any)["minigame"].(map[string]any)["pt"] = 30
				return b
			}(), c.agentTok},
			memberJob{memberStatusPath, map[string]any{"id": ids[(i+1)%6], "status": []string{"LOCKED", "ACTIVE"}[i%2]}, c.agentTok},
		)
	}
	countCodes(t, runJobs(app, jobs), 200)

	var bad int64
	database.DBConn.Raw(`SELECT count(*) FROM user_member_game_settings ums
		JOIN user_members m ON m.id = ums.user_member_id
		JOIN agent_game_settings a ON a.agent_id = m.agent_id AND a.game_code = ums.game_code
		WHERE m.agent_id = ? AND ums.remain <> a.pt_from_parent - ums.pt`, c.agent.ID).Scan(&bad)
	if bad != 0 {
		t.Fatalf("remain_quota ของ Member ไม่ตรงกับค่าที่ผู้สร้างได้รับ %d แถว", bad)
	}
}

// TestMemberUniqueViolationMapped — แถวที่ยังไม่ commit (ไม่ผ่าน advisory lock) ทำให้เช็คล่วงหน้าไม่เห็น → INSERT / UPDATE ชน unique
// ต้องตอบ error ของ field (402401 / 402403) ไม่ใช่ 409 / 500
func TestMemberUniqueViolationMapped(t *testing.T) {
	app := setup2(t)
	c := buildChain(t, app)
	target := createMember(t, app, c.agentTok, "memtarget", nil)
	hash, _ := utils.HashPassword(mgPassword)
	str := func(s string) *string { return &s }

	cases := []struct {
		name  string
		other models.UserMember // แถวที่อีก tx ใส่ไว้แต่ยังไม่ commit
		job   memberJob
		want  int
	}{
		{"สร้าง เบอร์ชน", models.UserMember{Username: "memext1", PhoneCountryCode: str("66"), Phone: str("833333333")},
			memberJob{createMemberPath, func() map[string]any {
				b := memberBody("memapi1", 0)
				b["phone_country_code"], b["phone"] = "66", "833333333"
				return b
			}(), c.agentTok}, 402403},
		{"สร้าง username ชน", models.UserMember{Username: "memapi2"},
			memberJob{createMemberPath, memberBody("memapi2", 0), c.agentTok}, 402401},
		{"แก้เบอร์ ชน", models.UserMember{Username: "memext3", PhoneCountryCode: str("66"), Phone: str("844444444")},
			memberJob{memberDetailUpdatePath, map[string]any{"id": target.ID,
				"info": map[string]any{"phone_country_code": "66", "phone": "844444444"}}, c.agentTok}, 402403},
	}
	for _, tc := range cases {
		other := tc.other
		other.AgentID, other.PasswordHash, other.Name, other.Currency, other.Status = c.agent.ID, hash, "ext", "THB", models.AgentStatusActive
		tx := database.DBConn.Begin()
		if err := tx.Create(&other).Error; err != nil {
			tx.Rollback()
			t.Fatalf("%s: %v", tc.name, err)
		}
		done := make(chan int, 1)
		go func(j memberJob) { done <- callCode(app, j.path, j.body, j.tok) }(tc.job)
		time.Sleep(500 * time.Millisecond) // ให้ request ไปรอ lock ของ unique index
		if err := tx.Commit().Error; err != nil {
			t.Fatalf("%s: commit %v", tc.name, err)
		}
		if got := <-done; got != tc.want {
			t.Fatalf("%s: ได้ code %d, want %d", tc.name, got, tc.want)
		}
	}
}
