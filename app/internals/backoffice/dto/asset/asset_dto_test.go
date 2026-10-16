package asset

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	assetCore "app/app/core/asset"
	"app/pkg/apperr"
	"app/pkg/utils"
)

func str(v string) utils.JSONString { return utils.JSONString{Value: v, Present: true, IsString: true} }

func TestResolveTargetWithScope(t *testing.T) { // AST-08 · เป้าหมาย AGENT เติมจาก token ทีหลัง (lead 2026-10-16)
	tests := []struct {
		name        string
		scope       utils.JSONString
		agentIDSent bool
		want        assetCore.Target
		wantErr     string // "" = ผ่าน · ไม่ว่าง = 422 และ msg ต้องมีข้อความนี้
	}{
		{"GLOBAL", str("GLOBAL"), false, assetCore.Target{Scope: assetCore.ScopeGlobal}, ""},
		{"AGENT (ของตัวเอง)", str("AGENT"), false, assetCore.Target{Scope: assetCore.ScopeAgent}, ""},

		{"ไม่ส่ง scope", utils.JSONString{}, false, assetCore.Target{}, "scope"},
		{"scope ไม่ใช่ string", utils.JSONString{Present: true}, false, assetCore.Target{}, "scope"},
		{"scope ตัวพิมพ์เล็ก", str("global"), false, assetCore.Target{}, "scope"},
		{"scope ว่าง", str(""), false, assetCore.Target{}, "scope"},
		{"scope อื่น", str("ALL"), false, assetCore.Target{}, "scope"},
		{"AGENT ส่ง agent_id มา", str("AGENT"), true, assetCore.Target{}, "agent_id"},
		{"GLOBAL ส่ง agent_id มา", str("GLOBAL"), true, assetCore.Target{}, "agent_id"},
		{"ไม่ส่ง scope แต่ส่ง agent_id (บอกเรื่อง agent_id ก่อน)", utils.JSONString{}, true, assetCore.Target{}, "agent_id"},
	}
	for _, tt := range tests {
		got, err := ResolveTarget(true, tt.scope, tt.agentIDSent)
		checkResolve(t, tt.name, got, err, tt.want, tt.wantErr)
		if err == nil && got.Resolved {
			t.Errorf("%s: ResolveTarget ต้องยังไม่ Resolved (RequireAssetTarget ตั้งให้ — AST-09)", tt.name)
		}
	}
}

func TestResolveTargetWithoutScope(t *testing.T) { // AST-01 · AST-08: เส้น branding / bundle (ของตัวเองเสมอ)
	tests := []struct {
		name        string
		scope       utils.JSONString
		agentIDSent bool
		want        assetCore.Target
		wantErr     string
	}{
		{"ไม่ส่งอะไร", utils.JSONString{}, false, assetCore.Target{Scope: assetCore.ScopeAgent}, ""},
		{"ส่ง scope AGENT ก็ผ่าน", str("AGENT"), false, assetCore.Target{Scope: assetCore.ScopeAgent}, ""},
		{"ส่ง scope GLOBAL ไม่ได้", str("GLOBAL"), false, assetCore.Target{}, "scope"},
		{"ส่ง scope ที่ไม่ใช่ string", utils.JSONString{Present: true}, false, assetCore.Target{}, "scope"},
		{"ส่ง agent_id มา", utils.JSONString{}, true, assetCore.Target{}, "agent_id"},
	}
	for _, tt := range tests {
		got, err := ResolveTarget(false, tt.scope, tt.agentIDSent)
		checkResolve(t, tt.name, got, err, tt.want, tt.wantErr)
	}
}

func checkResolve(t *testing.T, name string, got assetCore.Target, err error, want assetCore.Target, wantErr string) {
	t.Helper()
	if wantErr == "" {
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got (%+v, %v), want %+v", name, got, err, want)
		}
		return
	}
	var e *apperr.Error
	if err == nil || !errors.As(err, &e) || e.Code != 422 || !strings.Contains(e.MsgEN, wantErr) {
		t.Errorf("%s: want 422 ที่บอก field %q, got (%+v, %v)", name, wantErr, got, err)
	}
	if got.Resolved {
		t.Errorf("%s: error แล้วต้องไม่ resolved: %+v", name, got)
	}
}

func TestRequestValidate(t *testing.T) {
	ok := []utils.Validatable{
		&GameRequest{GameCode: "scratch_card"},
		&FilesListRequest{GameCode: "scratch_card", AssetType: "icon_button"},
		&FilesDeleteRequest{ConfigID: 1},
		&UsedUpdateRequest{GameCode: "scratch_card", ConfigID: 1},
		&UsedResetRequest{GameCode: "scratch_card", AssetType: "icon_button"},
		&ThemeUpdateRequest{GameCode: "scratch_card"}, // 5 ค่าตรวจที่ core.NormalizeTheme
		&BrandingUpdateRequest{SiteName: "x"},
		&BrandingUpdateRequest{}, // site_name ตรวจที่ service (ตัดช่องว่างก่อน)
		&UploadRequest{GameCode: "scratch_card", AssetType: "icon_button"},
	}
	for i, r := range ok {
		if err := r.Validate(); err != nil {
			t.Errorf("ok[%d] %T: %v", i, r, err)
		}
	}
	bad := []struct {
		name  string
		r     utils.Validatable
		field string
	}{
		{"GameRequest ว่าง", &GameRequest{}, "game_code"},
		{"GameRequest ช่องว่าง", &GameRequest{GameCode: "  "}, "game_code"},
		{"FilesList ไม่มี asset_type", &FilesListRequest{GameCode: "scratch_card"}, "asset_type"},
		{"FilesList ไม่มี game_code", &FilesListRequest{AssetType: "x"}, "game_code"},
		{"FilesDelete config_id 0", &FilesDeleteRequest{}, "config_id"},
		{"UsedUpdate config_id 0", &UsedUpdateRequest{GameCode: "scratch_card"}, "config_id"},
		{"UsedUpdate ไม่มี game_code", &UsedUpdateRequest{ConfigID: 1}, "game_code"},
		{"UsedReset ไม่มี asset_type", &UsedResetRequest{GameCode: "scratch_card"}, "asset_type"},
		{"ThemeUpdate ไม่มี game_code", &ThemeUpdateRequest{}, "game_code"},
		{"Branding ส่ง site_theme", &BrandingUpdateRequest{SiteName: "x", SiteTheme: str("dark")}, "site_theme"},
		{"Branding site_theme ไม่ใช่ string", &BrandingUpdateRequest{SiteTheme: utils.JSONString{Present: true}}, "site_theme"},
		{"Upload ไม่มี game_code", &UploadRequest{AssetType: "x"}, "game_code"},
		{"Upload ไม่มี asset_type", &UploadRequest{GameCode: "scratch_card"}, "asset_type"},
		{"Upload use_now ผิด", &UploadRequest{GameCode: "scratch_card", AssetType: "x", UseNow: "maybe"}, "use_now"},
		{"Upload use_now = 1", &UploadRequest{GameCode: "scratch_card", AssetType: "x", UseNow: "1"}, "use_now"},
	}
	for _, tt := range bad {
		err := tt.r.Validate()
		var e *apperr.Error
		if err == nil || !errors.As(err, &e) || e.Code != 422 || !strings.Contains(e.MsgEN, tt.field) {
			t.Errorf("%s: want 422 ที่บอก field %q, got %v", tt.name, tt.field, err)
		}
	}
}

func TestUploadUseNow(t *testing.T) { // AST-21: ค่าเริ่มต้น true
	for _, tt := range []struct {
		in   string
		want bool
	}{{"", true}, {"true", true}, {" true ", true}, {"false", false}} {
		r := UploadRequest{GameCode: "scratch_card", AssetType: "x", UseNow: tt.in}
		if err := r.Validate(); err != nil || r.UseNowValue != tt.want {
			t.Errorf("use_now=%q: got (%v, %v) want %v", tt.in, r.UseNowValue, err, tt.want)
		}
	}
}
