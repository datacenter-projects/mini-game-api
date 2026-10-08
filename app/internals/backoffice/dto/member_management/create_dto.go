// Package membermanagement คือ request / response ของเส้น /manage/members/* — spec: docs/modules/agent_management.md หัวข้อ 5
// ไม่มี null ใน API (account ACC-32): controller ใช้ utils.ParseBodyNoNull · ข้อความว่าง = "" · ตัวเลขว่าง = 0
// helper ตรวจค่าใช้ร่วมกับ agent จาก dto/agent_management
package membermanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/pkg/utils"
)

// MemberPTRequest — Member มีแค่ commission_percent ต่อกลุ่ม (MGMT-21)
type MemberPTRequest struct {
	CommissionPercent utils.Decimal `json:"commission_percent"`

	CommissionBP int `json:"-"` // bp หลัง Validate
}

// CreateMemberRequest — POST /api/v1/bo/pr/manage/members/create
type CreateMemberRequest struct {
	RequestID string                     `json:"request_id"`
	Username  string                     `json:"username"`
	Password  string                     `json:"password"`
	Name      string                     `json:"name"`
	Phone     string                     `json:"phone"`
	Balance   map[string]utils.Decimal   `json:"balance"`
	PT        map[string]MemberPTRequest `json:"pt"`

	BalanceMinor map[string]int64 `json:"-"` // หน่วยย่อย 1/100 หลัง Validate
}

type CreateMemberResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func (r *CreateMemberRequest) Validate() error {
	if err := agentManagementDto.ValidateAccountFields(r.RequestID, &r.Username, r.Password, r.Name, &r.Phone); err != nil {
		return err
	}
	var err error
	if r.BalanceMinor, err = agentManagementDto.ParseBalance(r.Balance); err != nil {
		return err
	}
	if err := agentManagementDto.CheckGroups(r.PT); err != nil {
		return err
	}
	for g, v := range r.PT {
		if v.CommissionBP, err = agentManagementDto.PercentBP("pt."+g+".commission_percent", v.CommissionPercent); err != nil {
			return err
		}
		r.PT[g] = v
	}
	return nil
}
