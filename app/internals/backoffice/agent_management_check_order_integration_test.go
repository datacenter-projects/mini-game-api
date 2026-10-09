//go:build integration

// ลำดับการเช็คและข้อความ error ของ agent_management (หัวข้อ 7.1 — แก้ 2026-10-09)
// รูปแบบทุก field บนลงล่างก่อน (DTO) แล้วกฎที่ต้องดู DB บนลงล่าง · msg บอก field และค่าที่ตั้งได้
package backoffice_test

import (
	"strings"
	"testing"
)

func expectMsgHas(t *testing.T, msg string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(msg, p) {
			t.Fatalf("msg %q ต้องมี %q", msg, p)
		}
	}
}

func TestCreateCheckOrderAndMessages(t *testing.T) {
	app := setup2(t)
	c := buildChain(t, app) // superadmin → comp01 (ได้รับ 90) → share01 (70) → agent01 (60)

	// 402301 บอก user_type ที่สร้างได้
	r := call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "newshare", []string{"THB"}, childPT(50, 0, 0, 0)), c.saTok)
	expect(t, r, 200, 402301)
	expectMsgHas(t, r.Msg, "user_type", "ส่ง SHARE_B2C ไม่ได้", "SHARE_B2C สร้างได้โดย COMPANY_TRANSFER", "แต่คุณเป็น SUPERADMIN", "COMPANY_SEAMLESS_1TO1")
	r = call(t, app, "POST", createAgentPath, agentBody("SHARE_B2", "newshare", []string{"THB"}, childPT(50, 0, 0, 0)), c.saTok)
	expect(t, r, 200, 402301)
	expectMsgHas(t, r.Msg, "ไม่มีประเภท SHARE_B2")

	// currencies ของ Company Transfer ห้ามส่ง — บอกเหตุผล
	r = call(t, app, "POST", createAgentPath, agentBody("COMPANY_TRANSFER", "newcom", []string{"THB"}, childPT(90, 0, 0, 0)), c.saTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "currencies", "ห้ามส่ง")

	// username ซ้ำ (field บน) มาก่อน pt เกิน (field ล่าง)
	r = call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "share01", []string{"THB"}, childPT(95, 0, 0, 0)), c.comTok)
	expect(t, r, 200, 402401)

	// pt เกินที่ได้รับ — บอกค่าสูงสุด
	r = call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "share02", []string{"THB"}, childPT(95, 0, 0, 0)), c.comTok)
	expect(t, r, 200, 402305)
	expectMsgHas(t, r.Msg, "pt.minigame.pt_from_parent", "90")

	// force เกินที่ให้ลูก — บอกค่าสูงสุด · ผิดหลาย field ใน pt ได้ field แรก
	r = call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "share02", []string{"THB"}, childPT(50, 60, 70, 2)), c.comTok)
	expect(t, r, 200, 402308)
	expectMsgHas(t, r.Msg, "pt.minigame.force", "50")

	// Commission เกิน — บอกช่วง
	r = call(t, app, "POST", createAgentPath, agentBody("SHARE_B2C", "share02", []string{"THB"}, childPT(50, 0, 0, 2)), c.comTok)
	expect(t, r, 200, 402309)
	expectMsgHas(t, r.Msg, "pt.minigame.commission_percent", "0 – 1")

	// ยอดไม่พอ (balance อยู่เหนือ pt) มาก่อน pt เกิน — บอกยอดที่มี
	b := agentBody("AGENT", "agentx", nil, childPT(95, 0, 0, 0))
	b["balance"] = map[string]any{"THB": 100}
	r = call(t, app, "POST", createAgentPath, b, c.agentTok)
	expect(t, r, 200, 402312)
	expectMsgHas(t, r.Msg, "balance.THB", "0.00")

	// update-hold บอกค่าที่ได้รับ
	r = call(t, app, "POST", updateHoldPath, holdBody(65), c.agentTok)
	expect(t, r, 200, 402305)
	expectMsgHas(t, r.Msg, "pt.minigame.pt", "60")

	// update-pt: เพดานของ pt_from_parent มาก่อนเงื่อนไขอื่น
	r = call(t, app, "POST", updatePTPath, ptBody(c.share.ID, 95, 0, 0, 5), c.comTok)
	expect(t, r, 200, 402305)
	expectMsgHas(t, r.Msg, "90")

	// สิทธิ์ sub ที่ไม่มีเมนู — บอกเมนูที่ให้ได้
	r = call(t, app, "POST", subCreatePath, subBody("staffx", map[string]string{"rate": "view"}), c.comTok)
	expect(t, r, 200, 422)
	expectMsgHas(t, r.Msg, "permissions.rate", "member", "announcement")
}
