package agentmanagement

import (
	"strings"

	"app/app/models"
)

// ChainNode — ชั้นบนหนึ่งชั้นใน user_agents.cnf (MGMT-61)
type ChainNode struct {
	ID       uint   `json:"id"`
	Position string `json:"position"` // role ตัวเล็ก: superadmin / company / shareholder / agent
}

// Chain — ค่าใน user_agents.cnf · Parent เรียงจาก Superadmin ลงมาถึงผู้สร้างตรง ไม่รวมตัวเอง (MGMT-61)
type Chain struct {
	Parent []ChainNode `json:"parent"`
}

// ChildChain — สายชั้นบนของบัญชีใหม่ = สายของผู้สร้าง + ผู้สร้าง (MGMT-61) · ไม่แก้ slice ของผู้สร้าง
func ChildChain(creator Chain, creatorID uint, creatorRole models.AgentRole) Chain {
	parent := make([]ChainNode, 0, len(creator.Parent)+1)
	parent = append(parent, creator.Parent...)
	parent = append(parent, ChainNode{ID: creatorID, Position: strings.ToLower(string(creatorRole))})
	return Chain{Parent: parent}
}
