//go:build integration

// helper ของ API test ของ asset — ตามตาราง test case ใน docs/modules/asset.md หัวข้อ 7
package backoffice_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"

	"github.com/gofiber/fiber/v2"
)

const cdnBase = "https://cdn.example.com/"

func assetPath(op string) string { return "/api/v1/bo/pr/asset/" + op }

// assetWorld — บัญชีที่ใช้ทดสอบ: Superadmin → comp01 (Company Transfer) → share01 (Share B2C) → agent01
// และ Superadmin → csm01 (Company Seamless Master) → sm01 / sm02 (Share Master)
// sub: share01@design (asset edit) · share01@viewer (asset view) · share01@none (ไม่มีสิทธิ์) · superadmin@ops (asset edit) · superadmin@look (asset view)
type assetWorld struct {
	chain
	csm, sm1, sm2                         created
	csmTok, sm1Tok, sm2Tok                string
	shareEditTok, shareViewTok, shareNone string
	saEditTok, saViewTok                  string
	saEdit                                created
}

func buildAssetWorld(t *testing.T, app *fiber.App) assetWorld {
	t.Helper()
	w := assetWorld{chain: buildChain(t, app)}
	w.csm, w.csmTok = mustCreate(t, app, w.saTok, agentBody("COMPANY_SEAMLESS_MASTER", "csm01", nil, childPT(80, 0, 0, 0)))
	w.sm1, w.sm1Tok = mustCreate(t, app, w.csmTok, agentBody("SHARE_B2C", "sm01", []string{"THB"}, commissionOnly(0)))
	w.sm2, w.sm2Tok = mustCreate(t, app, w.csmTok, agentBody("SHARE_B2C", "sm02", []string{"THB"}, commissionOnly(0)))

	sub := func(ownerTok, suffix string, perms map[string]string) (created, string) {
		d := createSubAPI(t, app, ownerTok, suffix, perms)
		return d, readyToken(t, app, models.AccountTypeSub, d.ID, d.Username)
	}
	_, w.shareEditTok = sub(w.shareTok, "design", map[string]string{"asset": "edit"})
	_, w.shareViewTok = sub(w.shareTok, "viewer", map[string]string{"asset": "view"})
	_, w.shareNone = sub(w.shareTok, "none", nil)
	w.saEdit, w.saEditTok = sub(w.saTok, "ops", map[string]string{"asset": "edit"})
	_, w.saViewTok = sub(w.saTok, "look", map[string]string{"asset": "view"})
	return w
}

// ---- request ----

// scopeOwn — scope AGENT = ของบัญชีที่ login (เป้าหมายมาจาก token · ไม่มี agent_id — AST-08 · lead 2026-10-16)
func scopeOwn(kv ...any) map[string]any {
	m := map[string]any{"scope": "AGENT"}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

func scopeGlobal(kv ...any) map[string]any {
	m := map[string]any{"scope": "GLOBAL"}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

// own — body ของเส้น branding / bundle (ของบัญชีที่ login เสมอ · ไม่มี scope / agent_id)
func own(kv ...any) map[string]any {
	m := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

func assetCall(t *testing.T, app *fiber.App, op string, body any, tok string) testutil.Response {
	t.Helper()
	return call(t, app, "POST", assetPath(op), body, tok)
}

// postForm — multipart/form-data พร้อมไฟล์ใน field "file" (file = nil ไม่แนบไฟล์)
func postForm(t *testing.T, app *fiber.App, op string, fields map[string]string, file []byte, tok string) testutil.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if file != nil {
		fw, err := mw.CreateFormFile("file", "upload.bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(file); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", assetPath(op), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := testutil.Response{Status: resp.StatusCode, Header: resp.Header}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("POST %s: invalid json %q", op, raw)
	}
	return out
}

// formOwn — form field ของ multipart: scope AGENT = ของบัญชีที่ login
func formOwn(kv ...string) map[string]string {
	m := map[string]string{"scope": "AGENT"}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func formGlobal(kv ...string) map[string]string {
	m := map[string]string{"scope": "GLOBAL"}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func itoa(n uint) string { b, _ := json.Marshal(n); return string(b) }

// ---- ไฟล์ตัวอย่าง ----

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// padded — เติมท้ายไฟล์ให้ยาว n ไบต์พอดี (header ยังอ่านขนาดได้ปกติ)
func padded(data []byte, n int) []byte {
	if len(data) >= n {
		return data[:n]
	}
	return append(data, make([]byte, n-len(data))...)
}

// webpUnreadable — webp ที่ magic ถูกแต่อ่านขนาดไม่ได้ (ต้องได้ 405102 — AST-17 · lead A2)
func webpUnreadable() []byte {
	d := make([]byte, 40)
	copy(d[0:], "RIFF")
	binary.LittleEndian.PutUint32(d[4:], 32)
	copy(d[8:], "WEBP")
	copy(d[12:], "VP8 ")
	return d
}

// ---- response ----

type uploadData struct {
	ConfigID  uint   `json:"config_id"`
	Scope     string `json:"scope"`
	AgentID   uint   `json:"agent_id"`
	GameCode  string `json:"game_code"`
	AssetType string `json:"asset_type"`
	URL       string `json:"url"`
	IsUsed    bool   `json:"is_used"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type usedItem struct {
	AssetType     string `json:"asset_type"`
	ConfigID      uint   `json:"config_id"`
	URL           string `json:"url"`
	Source        string `json:"source"`
	InheritedFrom string `json:"inherited_from"`
	IsRetired     bool   `json:"is_retired"`
}

type usedListData struct {
	Source string     `json:"source"`
	Items  []usedItem `json:"items"`
}

type themeData struct {
	Source             string            `json:"source"`
	InheritedFrom      string            `json:"inherited_from"`
	FontFamily         string            `json:"font_family"`
	MainColor1         string            `json:"main_color_1"`
	MainColor2         string            `json:"main_color_2"`
	StatsBarColor      string            `json:"stats_bar_color"`
	StatsBarColorHex   string            `json:"stats_bar_color_hex"`
	StatsBarFontFamily string            `json:"stats_bar_font_family"`
	Shades1            map[string]string `json:"main_color_1_shades"`
	Shades2            map[string]string `json:"main_color_2_shades"`
}

type brandingData struct {
	Source        string `json:"source"`
	InheritedFrom string `json:"inherited_from"`
	SiteName      string `json:"site_name"`
	Logo          string `json:"logo"`
}

type bundleData struct {
	Source string            `json:"source"`
	Assets map[string]string `json:"assets"`
	Theme  json.RawMessage   `json:"theme"`
	Site   struct {
		SiteName string `json:"site_name"`
		Logo     string `json:"logo"`
	} `json:"site"`
}

func decode[T any](t *testing.T, r testutil.Response) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.Data, &v); err != nil {
		t.Fatalf("decode %s: %v", r.Data, err)
	}
	return v
}

// ---- ขั้นตอนที่ใช้บ่อย ----

// upload อัป png ขนาด w×h เข้าช่อง (asset type ของ scratch_card) แล้วคืนผล · ต้องสำเร็จ
func upload(t *testing.T, app *fiber.App, tok string, fields map[string]string, game, slot string, w, h int) uploadData {
	t.Helper()
	f := map[string]string{"game_code": game, "asset_type": slot}
	for k, v := range fields {
		f[k] = v
	}
	r := postForm(t, app, "files/upload", f, pngOf(t, w, h), tok)
	expect(t, r, 200, 200)
	return decode[uploadData](t, r)
}

// scratch อัปรูปของ scratch_card ตามขนาดที่ช่องรับ (ช่องที่ใช้บ่อยในชุด test)
var scratchSizes = map[string][2]int{
	"icon_button": {32, 32}, "win_highlight": {425, 425}, "winner": {502, 514}, "background": {1920, 1080},
	"game_cover": {608, 1080}, "you_lose": {544, 287}, "bet_limit": {322, 322},
}

func uploadScratch(t *testing.T, app *fiber.App, tok string, fields map[string]string, slot string) uploadData {
	t.Helper()
	sz, ok := scratchSizes[slot]
	if !ok {
		t.Fatalf("ไม่มีขนาดของช่อง %s ในตาราง scratchSizes", slot)
	}
	return upload(t, app, tok, fields, "scratch_card", slot, sz[0], sz[1])
}

func slotID(t *testing.T, game, slot string) uint {
	t.Helper()
	var at models.AssetType
	if err := database.DBConn.Where("game_code = ? AND asset_type = ?", game, slot).Take(&at).Error; err != nil {
		t.Fatal(err)
	}
	return at.ID
}

func usedCount(t *testing.T, agentID *uint, game, slot string) int64 {
	t.Helper()
	q := database.DBConn.Model(&models.AssetUsed{}).Where("asset_type_id = ?", slotID(t, game, slot))
	if agentID == nil {
		q = q.Where("agent_id IS NULL")
	} else {
		q = q.Where("agent_id = ?", *agentID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func configRow(t *testing.T, id uint) models.AssetConfig {
	t.Helper()
	var c models.AssetConfig
	if err := database.DBConn.Where("id = ?", id).Take(&c).Error; err != nil {
		t.Fatal(err)
	}
	return c
}

func keyOf(url string) string { return strings.TrimPrefix(url, cdnBase) }

func usedItems(t *testing.T, app *fiber.App, tok string, body map[string]any) usedListData {
	t.Helper()
	r := assetCall(t, app, "used/list", body, tok)
	expect(t, r, 200, 200)
	return decode[usedListData](t, r)
}

func findItem(t *testing.T, d usedListData, slot string) usedItem {
	t.Helper()
	for _, it := range d.Items {
		if it.AssetType == slot {
			return it
		}
	}
	t.Fatalf("ไม่พบช่อง %s ใน used/list", slot)
	return usedItem{}
}

// retireSlot — เลิกใช้ช่อง (ผ่าน DB เหมือน migration) แล้วคืนค่าเดิมตอน test จบ — asset_types ไม่ถูกล้างระหว่าง test
func retireSlot(t *testing.T, game, slot string) {
	t.Helper()
	set := func(v bool) {
		if err := database.DBConn.Model(&models.AssetType{}).Where("game_code = ? AND asset_type = ?", game, slot).Update("is_retired", v).Error; err != nil {
			t.Fatal(err)
		}
	}
	set(true)
	t.Cleanup(func() { set(false) })
}
