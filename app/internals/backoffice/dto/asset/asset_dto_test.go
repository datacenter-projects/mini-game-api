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

func TestResolveTargetWithScope(t *testing.T) { // AST-08
	tests := []struct {
		name    string
		scope   utils.JSONString
		agentID string
		want    assetCore.Target
		wantErr string // "" = ผ่าน · ไม่ว่าง = 422 และ msg ต้องมีข้อความนี้
	}{
		{"GLOBAL ไม่ส่ง agent_id", str("GLOBAL"), "", assetCore.Target{Resolved: true, Scope: assetCore.ScopeGlobal}, ""},
		{"GLOBAL ส่ง agent_id = 0", str("GLOBAL"), "0", assetCore.Target{Resolved: true, Scope: assetCore.ScopeGlobal}, ""},
		{"AGENT มี agent_id", str("AGENT"), "12", assetCore.Target{Resolved: true, Scope: assetCore.ScopeAgent, AgentID: 12}, ""},
		{"AGENT ช่องว่างรอบเลข (form)", str("AGENT"), " 12 ", assetCore.Target{Resolved: true, Scope: assetCore.ScopeAgent, AgentID: 12}, ""},

		{"ไม่ส่ง scope", utils.JSONString{}, "12", assetCore.Target{}, "scope"},
		{"scope ไม่ใช่ string", utils.JSONString{Present: true}, "12", assetCore.Target{}, "scope"},
		{"scope ตัวพิมพ์เล็ก", str("global"), "", assetCore.Target{}, "scope"},
		{"scope ว่าง", str(""), "", assetCore.Target{}, "scope"},
		{"scope อื่น", str("ALL"), "", assetCore.Target{}, "scope"},
		{"AGENT ไม่ส่ง agent_id", str("AGENT"), "", assetCore.Target{}, "agent_id"},
		{"AGENT agent_id = 0", str("AGENT"), "0", assetCore.Target{}, "agent_id"},
		{"AGENT agent_id เป็นข้อความ", str("AGENT"), "abc", assetCore.Target{}, "agent_id"},
		{"AGENT agent_id เป็น JSON string", str("AGENT"), `"12"`, assetCore.Target{}, "agent_id"},
		{"AGENT agent_id ติดลบ", str("AGENT"), "-1", assetCore.Target{}, "agent_id"},
		{"AGENT agent_id ทศนิยม", str("AGENT"), "1.5", assetCore.Target{}, "agent_id"},
		{"AGENT agent_id ใหญ่เกิน", str("AGENT"), "99999999999", assetCore.Target{}, "agent_id"},
		{"GLOBAL แต่ส่ง agent_id", str("GLOBAL"), "12", assetCore.Target{}, "agent_id"},
		{"GLOBAL agent_id ไม่ใช่ตัวเลข (ห้ามเงียบ)", str("GLOBAL"), "abc", assetCore.Target{}, "agent_id"},
	}
	for _, tt := range tests {
		got, err := ResolveTarget(true, tt.scope, tt.agentID)
		checkResolve(t, tt.name, got, err, tt.want, tt.wantErr)
	}
}

func TestResolveTargetWithoutScope(t *testing.T) { // AST-01 · AST-08: เส้น branding / bundle
	tests := []struct {
		name    string
		scope   utils.JSONString
		agentID string
		want    assetCore.Target
		wantErr string
	}{
		{"agent_id อย่างเดียว", utils.JSONString{}, "12", assetCore.Target{Resolved: true, Scope: assetCore.ScopeAgent, AgentID: 12}, ""},
		{"ส่ง scope AGENT ก็ผ่าน", str("AGENT"), "12", assetCore.Target{Resolved: true, Scope: assetCore.ScopeAgent, AgentID: 12}, ""},
		{"ส่ง scope GLOBAL ไม่ได้", str("GLOBAL"), "12", assetCore.Target{}, "scope"},
		{"ส่ง scope ที่ไม่ใช่ string", utils.JSONString{Present: true}, "12", assetCore.Target{}, "scope"},
		{"ไม่ส่ง agent_id", utils.JSONString{}, "", assetCore.Target{}, "agent_id"},
		{"agent_id = 0", utils.JSONString{}, "0", assetCore.Target{}, "agent_id"},
		{"agent_id ไม่ใช่ตัวเลข", utils.JSONString{}, "x", assetCore.Target{}, "agent_id"},
	}
	for _, tt := range tests {
		got, err := ResolveTarget(false, tt.scope, tt.agentID)
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
