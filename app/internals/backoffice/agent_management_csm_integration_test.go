//go:build integration

// API test ของ Share Master ที่ใช้ค่าตาม Company Seamless Master — docs/modules/agent_management.md MGMT-19
// (lead review 2026-10-09 H2 / H3 / Q-R1 / E1-1 / E1-2 / B2)
package backoffice_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

// csmChain — superadmin → master01 (CSM ได้รับ 80) → sharem1 / sharem2 (Share Master) → Member ใต้ sharem1 ถือ 50
type csmChain struct {
	chain
	csm, sm1, sm2 created
	csmTok        string
	sm1Tok        string
	sm2Tok        string
	mem           created
}

func buildCSMChain(t *testing.T, app *fiber.App) csmChain {
	t.Helper()
	k := csmChain{chain: buildChain(t, app)}
	k.csm, k.csmTok = mustCreate(t, app, k.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "master01", nil, childPT(80, 0, 0, 0.5)))
	k.sm1, k.sm1Tok = mustCreate(t, app, k.csmTok, agentBody("SHARE_B2C", "sharem1", []string{"THB"}, commissionOnly(0.2)))
	k.sm2, k.sm2Tok = mustCreate(t, app, k.csmTok, agentBody("SHARE_B2C", "sharem2", []string{"USD"}, commissionOnly(0.3)))
	k.mem = createMemberWithPT(t, app, k.sm1Tok, "memsm1", 50)
	return k
}

func settingsOf(t *testing.T, agentID uint) []models.AgentGameSetting {
	t.Helper()
	var rows []models.AgentGameSetting
	database.DBConn.Where("agent_id = ?", agentID).Order("game_code").Find(&rows)
	if len(rows) != 3 {
		t.Fatalf("agent %d rows %d", agentID, len(rows))
	}
	return rows
}

func TestCSMEditsShareMaster(t *testing.T) { // MGMT-19 · lead B2: CSM แก้ Share Master ได้แค่ commission
	app := setup2(t)
	k := buildCSMChain(t, app)

	for name, body := range map[string]map[string]any{
		"ครบ 5 ค่า":          ptBody(k.sm1.ID, 80, 0, 0, 0.4),
		"ส่ง pt_from_parent": {"id": k.sm1.ID, "pt": map[string]any{"minigame": map[string]any{"pt_from_parent": 80, "commission_percent": 0.4}}},
		"ส่ง status":         {"id": k.sm1.ID, "pt": map[string]any{"minigame": map[string]any{"status": false, "commission_percent": 0.4}}},
		"status_game":        {"id": k.sm1.ID, "status_game": map[string]bool{"coin_toss": false}},
	} {
		r := call(t, app, "POST", agentDetailUpdatePath, body, k.csmTok)
		expect(t, r, 200, 422)
		expectMsgHas(t, r.Msg, "Company Seamless Master")
		_ = name
	}
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": k.sm1.ID, "pt": commissionOnly(1.1)}, k.csmTok), 200, 402309)

	// commission อย่างเดียว: แก้ได้ ค่าอื่นคงตาม CSM · info แก้ได้ปกติ (เบอร์ 2 field)
	body := map[string]any{"id": k.sm1.ID, "pt": commissionOnly(0.4),
		"info": map[string]any{"name": "smname", "phone_country_code": "66", "phone": "812345678"}}
	expect(t, call(t, app, "POST", agentDetailUpdatePath, body, k.csmTok), 200, 200)
	for _, s := range settingsOf(t, k.sm1.ID) {
		if s.Commission != 0.4 || s.PTFromParent != 80 || s.Force != 0 || s.Remain != 0 || !s.Status || !s.StatusGame {
			t.Fatalf("share master %+v", s)
		}
	}
	if s := memberSetting(t, k.mem.ID); s.Remain != 30 {
		t.Fatalf("แก้ commission ไม่กระทบ remain ของ Member %+v", s)
	}

	// บัญชีอื่นยังต้องส่งครบ 5 ค่า
	r := call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": k.share.ID, "pt": commissionOnly(0.1)}, k.comTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "pt.minigame.pt_from_parent")
	// Superadmin แก้ CSM ต้องครบ 5 ค่าเหมือนเดิม (CSM ไม่ใช่ Share Master)
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": k.csm.ID, "pt": commissionOnly(0.1)}, k.saTok), 200, 422)
}

func TestCSMSyncPT(t *testing.T) { // MGMT-19 · lead H3 / Q-R1: Superadmin แก้ CSM → Share Master ทุกคนตาม ใน tx เดียว
	app := setup2(t)
	k := buildCSMChain(t, app)

	// ลด CSM ต่ำกว่า pt ของ Member ใต้ Share Master = 402306 บอกค่าต่ำสุด · ไม่มีอะไรเปลี่ยน
	r := call(t, app, "POST", agentDetailUpdatePath, ptBody(k.csm.ID, 45, 0, 0, 0.5), k.saTok)
	expect(t, r, 200, 402306)
	expectMsgHas(t, r.Msg, "50")
	if s := settingsOf(t, k.sm1.ID)[0]; s.PTFromParent != 80 {
		t.Fatalf("ปฏิเสธแล้วต้องไม่เปลี่ยน %+v", s)
	}

	// ลดเหลือ 60 · ปิด pt.status → Share Master ทุกคน pt_from_parent 60 · force / remain 0 · status ตาม · commission คงเดิม
	body := ptBody(k.csm.ID, 60, 0, 0, 0.5)
	body["pt"].(map[string]any)["minigame"].(map[string]any)["status"] = false
	expect(t, call(t, app, "POST", agentDetailUpdatePath, body, k.saTok), 200, 200)
	for id, comm := range map[uint]float64{k.sm1.ID: 0.2, k.sm2.ID: 0.3} {
		for _, s := range settingsOf(t, id) {
			if s.PTFromParent != 60 || s.Force != 0 || s.Remain != 0 || s.Status || s.Commission != comm || s.UpdatedBy != "superadmin" {
				t.Fatalf("follower %d %+v", id, s)
			}
		}
	}
	// R2: remain_quota ของ Member = ค่าใหม่ของ Share Master − pt
	if s := memberSetting(t, k.mem.ID); s.PT != 50 || s.Remain != 10 {
		t.Fatalf("member %+v", s)
	}
	// log SYNC_FROM_CSM ต่อ Share Master (ผู้ทำ = superadmin) + UPDATE_PT ของ CSM
	var logs []models.AccountChangeLog
	database.DBConn.Where("action = ?", models.ChangeSyncFromCSM).Order("target_id").Find(&logs)
	if len(logs) != 2 || logs[0].TargetID != k.sm1.ID || logs[1].TargetID != k.sm2.ID || logs[0].ActorUsername != "superadmin" {
		t.Fatalf("sync logs %+v", logs)
	}
	var oldV map[string]map[string]any
	var newV struct {
		CSMID    uint           `json:"csm_id"`
		Minigame map[string]any `json:"minigame"`
	}
	_ = json.Unmarshal([]byte(*logs[0].OldValue), &oldV)
	_ = json.Unmarshal([]byte(*logs[0].NewValue), &newV)
	if oldV["minigame"]["pt_from_parent"] != float64(80) || newV.Minigame["pt_from_parent"] != float64(60) || newV.Minigame["status"] != false || newV.CSMID != k.csm.ID {
		t.Fatalf("log value old %v new %s", oldV, *logs[0].NewValue)
	}
	// B5 / V6: log ของ Share Master ใช้ request_id เดียวกับการแก้ CSM
	var csmLogs []models.AccountChangeLog
	database.DBConn.Where("action = ? AND target_id = ?", models.ChangeUpdatePT, k.csm.ID).Find(&csmLogs)
	if len(csmLogs) != 1 || csmLogs[0].RequestID == nil || *csmLogs[0].RequestID == "" {
		t.Fatalf("UPDATE_PT ของ CSM %+v", csmLogs)
	}
	for _, l := range logs {
		if l.RequestID == nil || *l.RequestID != *csmLogs[0].RequestID {
			t.Fatalf("request_id ของ SYNC_FROM_CSM ต้องเท่ากับของ CSM %v", l.RequestID)
		}
	}

	// แก้แค่ commission ของ CSM → Share Master ไม่เปลี่ยน · ไม่มี log sync เพิ่ม
	body = ptBody(k.csm.ID, 60, 0, 0, 0.7)
	body["pt"].(map[string]any)["minigame"].(map[string]any)["status"] = false
	expect(t, call(t, app, "POST", agentDetailUpdatePath, body, k.saTok), 200, 200)
	if n := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeSyncFromCSM); n != 2 {
		t.Fatalf("commission ของ CSM ต้องไม่ sync · logs %d", n)
	}

	// เพิ่มกลับ 80 → Share Master 80 · Member remain 30
	expect(t, call(t, app, "POST", agentDetailUpdatePath, ptBody(k.csm.ID, 80, 0, 0, 0.7), k.saTok), 200, 200)
	if s := settingsOf(t, k.sm2.ID)[0]; s.PTFromParent != 80 || !s.Status {
		t.Fatalf("sm2 %+v", s)
	}
	if s := memberSetting(t, k.mem.ID); s.Remain != 30 {
		t.Fatalf("member %+v", s)
	}
}

func TestCSMSyncGames(t *testing.T) { // MGMT-19 · lead H2: status_game ของ Share Master ตาม CSM
	app := setup2(t)
	k := buildCSMChain(t, app)
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": k.csm.ID, "status_game": map[string]bool{"scratch_card": false}}, k.saTok), 200, 200)
	for _, id := range []uint{k.csm.ID, k.sm1.ID, k.sm2.ID} {
		for _, s := range settingsOf(t, id) {
			if s.StatusGame != (s.GameCode != "scratch_card") {
				t.Fatalf("agent %d %+v", id, s)
			}
		}
	}
	var gl models.AccountChangeLog
	database.DBConn.Where("action = ? AND target_id = ?", models.ChangeSyncFromCSM, k.sm1.ID).Take(&gl)
	if gl.NewValue == nil || *gl.NewValue != fmt.Sprintf(`{"csm_id": %d, "status_game": {"scratch_card": false}}`, k.csm.ID) {
		t.Fatalf("log games %v", gl.NewValue)
	}
	if n := countRows(t, &models.AccountChangeLog{}, "action = ?", models.ChangeSyncFromCSM); n != 2 {
		t.Fatalf("sync logs %d", n)
	}
	// Share Master ที่สร้างหลังจากนี้ได้ status_game ตาม CSM
	sm3, _ := mustCreate(t, app, k.csmTok, agentBody("SHARE_B2C", "sharem3", []string{"THB"}, commissionOnly(0)))
	for _, s := range settingsOf(t, sm3.ID) {
		if s.StatusGame != (s.GameCode != "scratch_card") || s.PTFromParent != 80 {
			t.Fatalf("sm3 %+v", s)
		}
	}
	// เปิดกลับ → ตามทั้งหมด
	expect(t, call(t, app, "POST", agentDetailUpdatePath, map[string]any{"id": k.csm.ID, "status_game": map[string]bool{"scratch_card": true}}, k.saTok), 200, 200)
	for _, id := range []uint{k.sm1.ID, sm3.ID} {
		for _, s := range settingsOf(t, id) {
			if !s.StatusGame {
				t.Fatalf("agent %d %+v", id, s)
			}
		}
	}
}

// callCode — เรียก API จาก goroutine (ห้าม t.Fatal นอก goroutine หลัก) · คืน code หรือ -1 ถ้าเรียกไม่สำเร็จ
func callCode(app *fiber.App, path string, body any, tok string) int {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := app.Test(req, -1)
	if err != nil {
		return -1
	}
	defer func() { _ = resp.Body.Close() }()
	var out testutil.Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return -1
	}
	return out.Code
}

func TestCSMSyncConcurrent(t *testing.T) { // MGMT-19 · lead V5: ยิงพร้อมกัน — ไม่ deadlock · ค่าทุกชั้นตรงกันหลังจบ
	app := setup2(t)
	k := buildCSMChain(t, app)
	type job struct {
		path string
		body map[string]any
		tok  string
	}
	var jobs []job
	for i := 0; i < 6; i++ {
		give := []float64{70, 80, 75}[i%3]
		jobs = append(jobs,
			job{agentDetailUpdatePath, ptBody(k.csm.ID, give, 0, 0, 0.5), k.saTok},                                                     // Superadmin แก้ CSM → sync
			job{agentDetailUpdatePath, map[string]any{"id": k.csm.ID, "status_game": map[string]bool{"coin_toss": i%2 == 0}}, k.saTok}, // sync เกม
			job{createAgentPath, agentBody("SHARE_B2C", fmt.Sprintf("smrace%d", i), []string{"THB"}, commissionOnly(0.1)), k.csmTok},   // CSM สร้าง Share Master
			job{agentDetailUpdatePath, map[string]any{"id": k.sm2.ID, "pt": commissionOnly(float64(i) / 10)}, k.csmTok},                // CSM แก้ commission
		)
		mb := memberBody(fmt.Sprintf("memrace%d", i), 0) // Share Master สร้าง Member (pt 60 ≤ ค่าต่ำสุดที่ CSM จะได้รับ)
		mb["pt"].(map[string]any)["minigame"].(map[string]any)["pt"] = 60
		jobs = append(jobs, job{createMemberPath, mb, k.sm1Tok})
	}
	codes := make([]int, len(jobs))
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			codes[i] = callCode(app, j.path, j.body, j.tok)
		}(i, j)
	}
	wg.Wait()
	for i, code := range codes {
		if code != 200 {
			t.Fatalf("job %d %s ได้ code %d", i, jobs[i].path, code)
		}
	}

	// หลังจบ: Share Master ทุกคน = ค่าของ CSM · Member ทุกคนใต้ Share Master remain = ค่าที่ได้รับ − pt
	csm := settingsOf(t, k.csm.ID)
	var smIDs []uint
	database.DBConn.Model(&models.UserAgent{}).Where("parent_id = ?", k.csm.ID).Pluck("id", &smIDs)
	if len(smIDs) != 8 {
		t.Fatalf("share masters %d", len(smIDs))
	}
	for _, id := range smIDs {
		for i, s := range settingsOf(t, id) {
			if s.PTFromParent != csm[i].PTFromParent || s.Force != 0 || s.Remain != 0 || s.Status != csm[i].Status || s.StatusGame != csm[i].StatusGame {
				t.Fatalf("share master %d %+v · csm %+v", id, s, csm[i])
			}
		}
	}
	var bad int64
	database.DBConn.Raw(`SELECT count(*) FROM user_member_game_settings ums
		JOIN user_members m ON m.id = ums.user_member_id
		JOIN agent_game_settings sm ON sm.agent_id = m.agent_id AND sm.game_code = ums.game_code
		WHERE m.agent_id IN ? AND ums.remain <> sm.pt_from_parent - ums.pt`, smIDs).Scan(&bad)
	if bad != 0 {
		t.Fatalf("remain_quota ของ Member ไม่ตรง %d แถว", bad)
	}
	if n := countRows(t, &models.UserMember{}, "agent_id = ?", k.sm1.ID); n != 7 {
		t.Fatalf("members ใต้ sm1 %d", n)
	}
}
