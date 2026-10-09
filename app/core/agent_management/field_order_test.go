package agentmanagement

import (
	"reflect"
	"testing"
)

func TestCheckChildPT(t *testing.T) { // MGMT-18, MGMT-19, MGMT-25 · ไล่ตามลำดับ field ใน body
	tests := []struct {
		name     string
		v        ChildPT
		received int
		master   bool
		want     PTIssue
	}{
		{"ผ่าน", ChildPT{9000, 0, 0, 50}, 10000, false, PTIssue{Violation: PTOK}},
		{"ให้ลูกเกินที่ได้รับ", ChildPT{9000, 0, 0, 0}, 0, false, PTIssue{PTExceedsReceived, "pt_from_parent", 0}},
		{"ให้ลูกไม่ลง step", ChildPT{9025, 0, 0, 0}, 10000, false, PTIssue{PTInvalidStep, "pt_from_parent", FullPTBP}},
		{"force เกินที่ให้ลูก", ChildPT{6000, 7000, 0, 0}, 10000, false, PTIssue{PTForceRemainExceeded, "force", 6000}},
		{"remain เกินที่ให้ลูก", ChildPT{6000, 0, 6500, 0}, 10000, false, PTIssue{PTForceRemainExceeded, "remain_quota", 6000}},
		{"commission เกิน 1%", ChildPT{6000, 0, 0, 110}, 10000, false, PTIssue{PTCommissionExceeded, "commission_percent", MaxCommissionBP}},
		{"commission ไม่ลง step", ChildPT{6000, 0, 0, 15}, 10000, false, PTIssue{PTInvalidStep, "commission_percent", MaxCommissionBP}},
		{"field บนผิดก่อน", ChildPT{9000, 9500, 0, 200}, 8000, false, PTIssue{PTExceedsReceived, "pt_from_parent", 8000}},
		{"master ให้ไม่เท่าที่ได้รับ", ChildPT{7000, 0, 0, 0}, 8000, true, PTIssue{PTSeamlessMasterLock, "pt_from_parent", 8000}},
		{"master ตั้ง force", ChildPT{8000, 500, 0, 0}, 8000, true, PTIssue{PTSeamlessMasterLock, "force", 0}},
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
