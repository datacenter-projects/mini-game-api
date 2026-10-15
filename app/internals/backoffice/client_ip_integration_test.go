//go:build integration

// IP ของ client หลัง proxy (lead S1 รอบ 6): ไล่ X-Forwarded-For จากขวาข้าม proxy ที่เชื่อถือ — ใช้กับ whitelist ของ operator และ rate limit
// (operator transfer/* · player login · BO login) · connection ของ app.Test มาจาก 0.0.0.0 จึงใช้ 0.0.0.0/32 เป็น "ingress ที่เชื่อถือ"
package backoffice_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"app/app/models"
	"app/pkg/configs"
	"app/pkg/testutil"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const (
	boLoginPath = "/api/v1/bo/pb/auth/login"
	clientIP    = "203.0.113.9" // IP ของ operator / ผู้เล่นตัวจริงที่ ingress เห็น
)

// callXFF ยิง POST พร้อม X-Forwarded-For (หลายบรรทัดได้) — testutil.Call ใส่ header เองไม่ได้
func callXFF(t *testing.T, app *fiber.App, path string, body any, xff ...string) testutil.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for _, v := range xff {
		req.Header.Add("X-Forwarded-For", v)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := testutil.Response{Status: resp.StatusCode, Header: resp.Header}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("POST %s: invalid json %q", path, raw)
	}
	return out
}

// ipEnv — operator agent01 มี whitelist เฉพาะ clientIP · ingress ที่เชื่อถือ = 0.0.0.0/32 (connection ของ test)
func ipEnv(t *testing.T) trEnv {
	t.Helper()
	e := plSetup(t)
	opSeed(t, e.c.agent.ID, clientIP+"/32")
	opSeed(t, e.c.share.ID, clientIP+"/32")
	configs.Cfg.TrustedProxies = []string{"0.0.0.0/32"}
	return trEnv{e}
}

func TestClientIPWhitelistBehindProxies(t *testing.T) { // S1: XFF หลายชั้นต้องผ่าน whitelist (เดิมได้ 407305 เพราะ c.IP() เป็นก้อน)
	e := ipEnv(t)
	body := e.body("p001", nil)

	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP), 200, 200)                               // ingress ชั้นเดียว
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP+", 0.0.0.0"), 200, 200)                   // หลายชั้น ก้อนเดียว
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP, "0.0.0.0"), 200, 200)                    // หลายบรรทัด
	expect(t, callXFF(t, e.app, trLoginPath, body, "66.66.66.66, "+clientIP+", 0.0.0.0"), 200, 200)   // client ปลอมทางซ้าย ไม่มีผล
	expect(t, callXFF(t, e.app, trRegisterPath, e.body("p002", nil), clientIP+", 0.0.0.0"), 200, 200) // register ด้วย
	if m, ok := memberByUsername(t, "agent01@p001"); !ok {
		t.Fatal("ต้องสร้างผู้เล่น")
	} else if m.LastLoginIP == nil || *m.LastLoginIP != clientIP { // last_login_ip = IP จริง ไม่ใช่ก้อน XFF
		t.Errorf("last_login_ip = %v want %s", m.LastLoginIP, clientIP)
	}
}

func TestClientIPSpoofCannotBypassWhitelist(t *testing.T) { // S1: ปลอม IP ที่อยู่ใน whitelist ไว้ทางซ้ายต้องไม่ผ่าน
	e := ipEnv(t)
	body := e.body("p001", nil)
	// ผู้โจมตี (6.6.6.6) ใส่ XFF = IP ที่ whitelist ไว้ · ingress ต่อท้ายด้วย IP จริงของผู้โจมตี
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP+", 6.6.6.6"), 403, 407305)
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP, "6.6.6.6"), 403, 407305)
	expect(t, callXFF(t, e.app, trLoginPath, body, "garbage, 6.6.6.6"), 403, 407305)

	// ค่าที่ client ใส่เองทางซ้ายไม่มีผล แม้เป็นขยะ: ingress ต่อท้ายด้วย IP จริงแล้วเราอ่านจากขวา
	expect(t, callXFF(t, e.app, trLoginPath, body, "garbage, "+clientIP), 200, 200)
	// ค่าอ่านไม่ได้ทางขวา (มีอะไรผิดปกติในสาย proxy) = fail-closed ใช้ IP ของ connection → ไม่อยู่ใน whitelist
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP+", garbage"), 403, 407305)
}

func TestClientIPUntrustedPeerIgnoresXFF(t *testing.T) { // S1: ไม่ตั้ง TRUSTED_PROXIES / connection ไม่ใช่ proxy ที่เชื่อถือ = ไม่เชื่อ XFF
	e := ipEnv(t)
	body := e.body("p001", nil)

	configs.Cfg.TrustedProxies = nil // ไม่ตั้ง: IP ของ connection ตรงๆ (0.0.0.0) ไม่อยู่ใน whitelist
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP), 403, 407305)

	configs.Cfg.TrustedProxies = []string{"10.0.0.0/8"} // ตั้ง แต่ connection (0.0.0.0) ไม่ใช่ proxy ที่เชื่อถือ
	expect(t, callXFF(t, e.app, trLoginPath, body, clientIP), 403, 407305)

	// ตรงข้าม: whitelist = IP ของ connection จริง · XFF ปลอมเป็นอย่างอื่นก็ไม่กระทบ
	if err := database.DBConn.Where("agent_id = ?", e.c.agent.ID).Delete(&models.APIAllowedIP{}).Error; err != nil {
		t.Fatal(err)
	}
	opSeedIPs(t, e.c.agent.ID, trTestIP)
	configs.Cfg.TrustedProxies = nil
	expect(t, callXFF(t, e.app, trLoginPath, body, "1.2.3.4"), 200, 200)
}

// opSeedIPs เพิ่ม IP เข้า whitelist ของ agent ที่มีแถว Key อยู่แล้ว
func opSeedIPs(t *testing.T, agentID uint, cidrs ...string) {
	t.Helper()
	for _, c := range cidrs {
		if err := database.DBConn.Create(&models.APIAllowedIP{AgentID: agentID, CIDR: c}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestClientIPRateLimitNotBypassedByForgedXFF(t *testing.T) { // S1: client สลับค่า XFF ทางซ้ายทุก request ต้องยังถูกนับเป็นคนเดียว
	t.Run("operator transfer/*", func(t *testing.T) {
		e := ipEnv(t)
		configs.Cfg.Auth.OperatorIPLimit = 3
		for i := 1; i <= 3; i++ {
			expect(t, callXFF(t, e.app, trLoginPath, e.body("p001", nil), fmt.Sprintf("10.9.9.%d, %s", i, clientIP)), 200, 200)
		}
		expect(t, callXFF(t, e.app, trLoginPath, e.body("p001", nil), "10.9.9.99, "+clientIP), 429, 429)
		// IP จริงอื่นยังไม่โดน (นับต่อ IP จริง)
		if err := database.DBConn.Create(&models.APIAllowedIP{AgentID: e.c.agent.ID, CIDR: "198.51.100.20/32"}).Error; err != nil {
			t.Fatal(err)
		}
		expect(t, callXFF(t, e.app, trLoginPath, e.body("p001", nil), "198.51.100.20"), 200, 200)
	})

	t.Run("player login", func(t *testing.T) {
		e := ipEnv(t)
		configs.Cfg.Auth.LoginIPLimit = 3
		for i := 1; i <= 3; i++ {
			r := callXFF(t, e.app, plLoginPath, map[string]any{"username": fmt.Sprintf("ghost%d", i), "password": "x"}, fmt.Sprintf("10.9.9.%d, %s", i, clientIP))
			expect(t, r, 401, 407201)
		}
		expect(t, callXFF(t, e.app, plLoginPath, map[string]any{"username": "ghost9", "password": "x"}, "10.9.9.99, "+clientIP), 429, 429)
	})

	t.Run("BO login (กระทบของเดิมด้วย)", func(t *testing.T) {
		e := ipEnv(t)
		configs.Cfg.Auth.LoginIPLimit = 3
		for i := 1; i <= 3; i++ {
			r := callXFF(t, e.app, boLoginPath, map[string]any{"username": fmt.Sprintf("ghost%d", i), "password": "x"}, fmt.Sprintf("10.9.9.%d, %s", i, clientIP))
			expect(t, r, 401, 401201)
		}
		expect(t, callXFF(t, e.app, boLoginPath, map[string]any{"username": "ghost9", "password": "x"}, "10.9.9.99, "+clientIP), 429, 429)
	})
}
