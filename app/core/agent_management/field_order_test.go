package agentmanagement

import (
	"reflect"
	"testing"

	"app/app/models"
)

func TestCheckChildPT(t *testing.T) { // MGMT-18, MGMT-19, MGMT-25 · ไล่ตามลำดับ field ใน body
	tests := []struct {
		name     string
		v        ChildPT
		received float64
		master   bool
		want     PTIssue
	}{
		{"ผ่าน", ChildPT{90, 0, 0, 0.5}, 100, false, PTIssue{Violation: PTOK}},
		{"ให้ลูกเกินที่ได้รับ", ChildPT{90, 0, 0, 0}, 0, false, PTIssue{PTExceedsReceived, "pt_from_parent", 0}},
		{"ให้ลูกไม่ลง step", ChildPT{90.25, 0, 0, 0}, 100, false, PTIssue{PTInvalidStep, "pt_from_parent", FullPT}},
		{"force เกินที่ให้ลูก", ChildPT{60, 70, 0, 0}, 100, false, PTIssue{PTForceRemainExceeded, "force", 60}},
		{"remain เกินที่ให้ลูก", ChildPT{60, 0, 65, 0}, 100, false, PTIssue{PTForceRemainExceeded, "remain_quota", 60}},
		{"commission เกิน 1%", ChildPT{60, 0, 0, 1.1}, 100, false, PTIssue{PTCommissionExceeded, "commission_percent", MaxCommission}},
		{"commission ไม่ลง step", ChildPT{60, 0, 0, 0.15}, 100, false, PTIssue{PTInvalidStep, "commission_percent", MaxCommission}},
		{"field บนผิดก่อน", ChildPT{90, 95, 0, 2}, 80, false, PTIssue{PTExceedsReceived, "pt_from_parent", 80}},
		{"master ให้ไม่เท่าที่ได้รับ", ChildPT{70, 0, 0, 0}, 80, true, PTIssue{PTSeamlessMasterLock, "pt_from_parent", 80}},
		{"master ตั้ง force", ChildPT{80, 5, 0, 0}, 80, true, PTIssue{PTSeamlessMasterLock, "force", 0}},
	}
	for _, tt := range tests {
		if got := CheckChildPT(tt.v, tt.received, tt.master); got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestCreatableTypes(t *testing.T) { // MGMT-02
	tests := []struct {
		creator UserType
		want    []UserType
	}{
		{UserTypeSuperadmin, []UserType{UserTypeCompanyTransfer, UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster, UserTypeCompanySeamless1to1}},
		{UserTypeCompanyTransfer, []UserType{UserTypeShareB2B, UserTypeShareB2C}},
		{UserTypeCompanySeamlessMaster, []UserType{UserTypeShareB2C}},
		{UserTypeShareB2B, []UserType{UserTypeAgent}},
		{UserTypeAgent, []UserType{UserTypeAgent}},
		{UserTypeCompanySeamless1to1, nil},
		{UserTypeAdmin, nil},
	}
	for _, tt := range tests {
		if got := CreatableTypes(tt.creator); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.creator, got, tt.want)
		}
		for _, nt := range CreatableTypes(tt.creator) { // ทุกค่าที่บอกต้องสร้างได้จริง
			if _, ok := ResolveNewAgent(tt.creator, nt); !ok {
				t.Errorf("%s บอกว่าสร้าง %s ได้ แต่ ResolveNewAgent ปฏิเสธ", tt.creator, nt)
			}
		}
	}
}

func TestCurrencyRequirementOf(t *testing.T) { // MGMT-10 – MGMT-13
	tests := []struct {
		creator, newType UserType
		want             CurrencyRequirement
	}{
		{UserTypeSuperadmin, UserTypeCompanyTransfer, CurrencyAllNoSend},
		{UserTypeSuperadmin, UserTypeCompanySeamless1to1, CurrencyPickOne},
		{UserTypeCompanyTransfer, UserTypeShareB2B, CurrencyPickMany},
		{UserTypeShareB2B, UserTypeAgent, CurrencyPickOne},
		{UserTypeShareB2C, UserTypeAgent, CurrencyFromCreatorNoSend},
	}
	for _, tt := range tests {
		if got := CurrencyRequirementOf(tt.creator, tt.newType); got != tt.want {
			t.Errorf("%s → %s: got %v, want %v", tt.creator, tt.newType, got, tt.want)
		}
	}
}

func TestCreatorsOf(t *testing.T) { // MGMT-02
	tests := []struct {
		newType UserType
		want    []UserType
	}{
		{UserTypeCompanyTransfer, []UserType{UserTypeSuperadmin}},
		{UserTypeShareB2B, []UserType{UserTypeCompanyTransfer}},
		{UserTypeShareB2C, []UserType{UserTypeCompanyTransfer, UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster}},
		{UserTypeAgent, []UserType{UserTypeShareB2B, UserTypeShareB2C, UserTypeShareReseller, UserTypeShareMaster, UserTypeAgent}},
		{UserTypeShareReseller, nil},
		{UserTypeMember, nil},
		{UserTypeSuperadmin, nil},
		{"SHARE_B2", nil},
	}
	for _, tt := range tests {
		got := CreatorsOf(tt.newType)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.newType, got, tt.want)
		}
		for _, c := range got { // ทุกผู้สร้างที่บอกต้องสร้างได้จริง
			if _, ok := ResolveNewAgent(c, tt.newType); !ok {
				t.Errorf("บอกว่า %s สร้าง %s ได้ แต่ ResolveNewAgent ปฏิเสธ", c, tt.newType)
			}
		}
	}
}

func TestChildChain(t *testing.T) { // MGMT-61
	root := Chain{Parent: []ChainNode{}}
	com := ChildChain(root, 1, models.AgentRoleSuperAdmin)
	share := ChildChain(com, 4754, models.AgentRoleCompany)
	agent := ChildChain(share, 5322, models.AgentRoleShareholder)
	agent2 := ChildChain(agent, 5330, models.AgentRoleAgent)
	tests := []struct {
		name string
		got  Chain
		want []ChainNode
	}{
		{"ลูกของ superadmin", com, []ChainNode{{1, "superadmin"}}},
		{"ลูกของ company", share, []ChainNode{{1, "superadmin"}, {4754, "company"}}},
		{"ลูกของ shareholder", agent, []ChainNode{{1, "superadmin"}, {4754, "company"}, {5322, "shareholder"}}},
		{"agent ซ้อน agent", agent2, []ChainNode{{1, "superadmin"}, {4754, "company"}, {5322, "shareholder"}, {5330, "agent"}}},
	}
	for _, tt := range tests {
		if !reflect.DeepEqual(tt.got.Parent, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got.Parent, tt.want)
		}
	}
	// สร้างลูกหลายคนจากผู้สร้างเดียวกัน ต้องไม่ทับกัน
	a := ChildChain(share, 10, models.AgentRoleShareholder)
	b := ChildChain(share, 11, models.AgentRoleShareholder)
	if a.Parent[2].ID != 10 || b.Parent[2].ID != 11 || len(share.Parent) != 2 {
		t.Errorf("ChildChain แก้ slice ของผู้สร้าง: a=%v b=%v share=%v", a.Parent, b.Parent, share.Parent)
	}
}
