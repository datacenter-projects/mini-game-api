package agentmanagement

import (
	"context"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/repository/postgres"
	"app/platform/database"

	"gorm.io/gorm"
)

// AdminSearchAccountsService — POST /api/v1/bo/pr/admin/accounts/search (MGMT-27B)
// ADMIN ค้นบัญชีทั้งระบบด้วย username ตรงทั้งคำ (ไม่จำกัดสาย — ADMIN ไม่มีสายของตัวเอง)
// ผล 0–2 แถว: ฝั่ง agent หรือ sub (username ไม่ซ้ำกัน) แล้วตามด้วย Member (ตารางแยก username ซ้ำกับฝั่ง agent ได้)
func AdminSearchAccountsService(ctx context.Context, req agentManagementDto.AdminAccountSearchRequest) ([]agentManagementDto.AdminAccountRow, error) {
	db := database.DBConn.WithContext(ctx)
	out := []agentManagementDto.AdminAccountRow{}

	if agentAuthCore.IsSubaccountUsername(req.Username) {
		s, err := postgres.FindSubaccountByUsernameRepository(db, req.Username)
		if err != nil {
			return nil, err
		}
		if s.ID != 0 {
			owner, err := postgres.GetAgentProfileRepository(db, s.AgentID)
			if err != nil {
				return nil, err
			}
			chain, err := chainStatus(db, owner.ID, owner.Status)
			if err != nil {
				return nil, err
			}
			out = append(out, agentManagementDto.AdminAccountRow{Username: s.Username, Role: string(owner.Role),
				UserType: string(agentManagementCore.UserTypeOf(owner.Role, owner.AgentType)), IsSubaccount: true,
				Status: subStatus(s.Status, chain), ParentUsername: owner.Username, CreatedAt: optionalTime(&s.CreatedAt),
				LastLoginAt: optionalTime(s.LastLoginAt), LastLoginIP: stringOrEmpty(s.LastLoginIP)})
		}
		return out, nil // username ของ sub มี @ — ไม่มีในตารางอื่น
	}

	a, err := postgres.FindUserAgentByUsernameRepository(db, req.Username)
	if err != nil {
		return nil, err
	}
	if a.ID != 0 {
		status, err := chainStatus(db, a.ID, a.Status)
		if err != nil {
			return nil, err
		}
		parent, err := parentUsername(db, a.ParentID)
		if err != nil {
			return nil, err
		}
		out = append(out, agentManagementDto.AdminAccountRow{Username: a.Username, Role: string(a.Role),
			UserType: string(agentManagementCore.UserTypeOf(a.Role, a.AgentType)), Status: string(status),
			ParentUsername: parent, CreatedAt: optionalTime(&a.CreatedAt),
			LastLoginAt: optionalTime(a.LastLoginAt), LastLoginIP: stringOrEmpty(a.LastLoginIP)})
	}

	m, err := postgres.FindMemberByUsernameRepository(db, req.Username)
	if err != nil {
		return nil, err
	}
	if m.ID != 0 {
		creator, err := postgres.GetAgentProfileRepository(db, m.AgentID)
		if err != nil {
			return nil, err
		}
		creatorStatus, err := chainStatus(db, creator.ID, creator.Status)
		if err != nil {
			return nil, err
		}
		member := string(agentManagementCore.UserTypeMember)
		out = append(out, agentManagementDto.AdminAccountRow{Username: m.Username, Role: member, UserType: member,
			Status: string(agentAuthCore.WorstStatus(creatorStatus, m.Status)), ParentUsername: creator.Username,
			CreatedAt: optionalTime(&m.CreatedAt), LastLoginAt: optionalTime(m.LastLoginAt), LastLoginIP: stringOrEmpty(m.LastLoginIP)})
	}
	return out, nil
}

// parentUsername — username ของผู้สร้าง · ไม่มีผู้สร้าง (SUPERADMIN / ADMIN) = ""
func parentUsername(db *gorm.DB, parentID *uint) (string, error) {
	if parentID == nil {
		return "", nil
	}
	p, err := postgres.GetAgentProfileRepository(db, *parentID)
	if err != nil {
		return "", err
	}
	return p.Username, nil
}
