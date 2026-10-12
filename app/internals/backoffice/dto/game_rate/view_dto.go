package gamerate

// response ของเส้น /rate/* — spec: docs/modules/game_rate.md หัวข้อ 5
// ค่าที่คำนวณไม่ได้เก็บใน DB — คำนวณจากค่าที่ตั้งทุกครั้ง (RATE-09) · ไม่มีค่า = 0 / "" (ไม่มี null — ACC-32) ยกเว้น from / to ของ changes

// ListRow — 1 เกมของ RE1
type ListRow struct {
	GameCode            string  `json:"game_code"`
	Version             int     `json:"version"`
	MinHouseEdgePercent float64 `json:"min_house_edge_percent"`
	CreatedAt           string  `json:"created_at"`
	CreatedBy           string  `json:"created_by"`
}

// StepView — 1 ขั้นพร้อมค่าที่คำนวณ (รูปแบบ steps)
type StepView struct {
	Step             int     `json:"step"`
	Multiplier       float64 `json:"multiplier"`
	WinPercent       float64 `json:"win_percent"`
	ReachPercent     float64 `json:"reach_percent"`
	PayoutPercent    float64 `json:"payout_percent"`
	HouseEdgePercent float64 `json:"house_edge_percent"`
	MaxWinPercent    float64 `json:"max_win_percent"`
}

// StepsCalc — ค่าของรูปแบบ steps + สรุปทั้งเกม
type StepsCalc struct {
	Steps               []StepView `json:"steps"`
	MinHouseEdgePercent float64    `json:"min_house_edge_percent"`
	MinHouseEdgeStep    int        `json:"min_house_edge_step"`
	MaxPayoutPercent    float64    `json:"max_payout_percent"`
}

// PrizeView — 1 แถวรางวัลพร้อม % ต่อใบ (รูปแบบ prizes)
type PrizeView struct {
	RowNo            int     `json:"row_no"`
	Multiplier       float64 `json:"multiplier"`
	SharePercent     float64 `json:"share_percent"`
	PerTicketPercent float64 `json:"per_ticket_percent"`
}

// PrizesCalc — ค่าของรูปแบบ prizes
type PrizesCalc struct {
	WinPercent       float64     `json:"win_percent"`
	Prizes           []PrizeView `json:"prizes"`
	AvgMultiplier    float64     `json:"avg_multiplier"`
	PayoutPercent    float64     `json:"payout_percent"`
	HouseEdgePercent float64     `json:"house_edge_percent"`
	MaxWinPercent    float64     `json:"max_win_percent"`
}

// StepsDetail — RE2 / RE6 ของรูปแบบ steps
type StepsDetail struct {
	GameCode string `json:"game_code"`
	Version  int    `json:"version"`
	StepsCalc
	CreatedAt string `json:"created_at"`
	CreatedBy string `json:"created_by"`
}

// PrizesDetail — RE2 / RE6 ของรูปแบบ prizes
type PrizesDetail struct {
	GameCode string `json:"game_code"`
	Version  int    `json:"version"`
	PrizesCalc
	CreatedAt string `json:"created_at"`
	CreatedBy string `json:"created_by"`
}

// Issue — จุดที่ไม่ผ่านกฎใน RE3 (lead Q2)
type Issue struct {
	Field   string `json:"field"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StepsCheck — RE3 ของรูปแบบ steps
type StepsCheck struct {
	GameCode string  `json:"game_code"`
	Valid    bool    `json:"valid"`
	Issues   []Issue `json:"issues"`
	StepsCalc
}

// PrizesCheck — RE3 ของรูปแบบ prizes
type PrizesCheck struct {
	GameCode string  `json:"game_code"`
	Valid    bool    `json:"valid"`
	Issues   []Issue `json:"issues"`
	PrizesCalc
}

// SaveResponse — RE4 · ค่าเท่าปัจจุบัน = created false (RATE-05C)
type SaveResponse struct {
	GameCode string `json:"game_code"`
	Version  int    `json:"version"`
	Created  bool   `json:"created"`
}

// RollbackResponse — RE7
type RollbackResponse struct {
	GameCode     string `json:"game_code"`
	Version      int    `json:"version"`
	RollbackFrom int    `json:"rollback_from"`
	Created      bool   `json:"created"`
}

// ChangeView — 1 รายการของ changes ใน RE5 (lead Q4) · from / to ไม่มีค่า = null
type ChangeView struct {
	Type       string   `json:"type"`
	Step       int      `json:"step,omitempty"`       // type step
	Field      string   `json:"field,omitempty"`      // type step
	Multiplier float64  `json:"multiplier,omitempty"` // รางวัล (> 0 เสมอ)
	From       *float64 `json:"from"`
	To         *float64 `json:"to"`
}

// VersionRow — 1 เวอร์ชันของ RE5
type VersionRow struct {
	Version             int          `json:"version"`
	MinHouseEdgePercent float64      `json:"min_house_edge_percent"`
	WinPercent          *float64     `json:"win_percent,omitempty"` // รูปแบบ prizes เท่านั้น
	RollbackFrom        int          `json:"rollback_from"`
	SameAsCurrent       bool         `json:"same_as_current"`
	Changes             []ChangeView `json:"changes"`
	CreatedAt           string       `json:"created_at"`
	CreatedBy           string       `json:"created_by"`
	IP                  string       `json:"ip"`
}
