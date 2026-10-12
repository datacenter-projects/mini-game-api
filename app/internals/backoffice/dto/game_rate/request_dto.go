// Package gamerate คือ request / response ของเส้น /rate/* — spec: docs/modules/game_rate.md หัวข้อ 5
// controller ใช้ utils.ParseBodyNoNull (ห้าม null) · ตัวเลขเป็น JSON number ทศนิยมไม่เกิน 4 ตำแหน่ง (utils.IsDecimal4)
// ตรวจรูปแบบตามรูปแบบอัตราของเกม (core/game_rate GameSpec) — เกมใหม่ที่ใช้รูปแบบเดิมไม่ต้องแก้ไฟล์นี้
package gamerate

import (
	"fmt"

	gameRateCore "app/app/core/game_rate"
	"app/pkg/apperr"
	"app/pkg/utils"
)

func invalid(field, th, en string) error {
	return apperr.ErrValidation.WithMessage(field+" "+th, field+" "+en)
}

// checkGame — game_code ต้องส่ง · ไม่มีในระบบ = 406401
func checkGame(code string) (gameRateCore.GameSpec, error) {
	if code == "" {
		return gameRateCore.GameSpec{}, invalid("game_code", "ต้องส่ง", "is required")
	}
	spec, ok := gameRateCore.SpecOf(code)
	if !ok {
		return spec, apperr.ErrRateGameNotFound
	}
	return spec, nil
}

// decimal4 — ต้องส่ง · ทศนิยมไม่เกิน 4 ตำแหน่ง (RATE-03 · RATE-04)
func decimal4(field string, v *float64) (float64, error) {
	if v == nil {
		return 0, invalid(field, "ต้องส่ง", "is required")
	}
	if !utils.IsDecimal4(*v) {
		return 0, invalid(field, "ต้องเป็นตัวเลข ทศนิยมไม่เกิน 4 ตำแหน่ง", "must be a number with at most 4 decimals")
	}
	return *v, nil
}

// GameRequest — RE2 detail/get (game_code อย่างเดียว)
type GameRequest struct {
	GameCode string `json:"game_code"`
}

func (r *GameRequest) Validate() error {
	_, err := checkGame(r.GameCode)
	return err
}

// StepIn — 1 ขั้นของรูปแบบ steps · RPS ไม่มีเสมอ: ส่ง draw_percent = 422 (lead Q7)
type StepIn struct {
	Multiplier  *float64 `json:"multiplier"`
	WinPercent  *float64 `json:"win_percent"`
	DrawPercent *float64 `json:"draw_percent"`
}

// PrizeIn — 1 แถวรางวัลของรูปแบบ prizes
type PrizeIn struct {
	Multiplier   *float64 `json:"multiplier"`
	SharePercent *float64 `json:"share_percent"`
}

// RatesInput — ค่าอัตราของ 1 เกม (RE3 · RE4) · ส่ง field ของรูปแบบอื่น = 422 (RATE-01)
type RatesInput struct {
	GameCode   string     `json:"game_code"`
	Steps      *[]StepIn  `json:"steps"`       // รูปแบบ steps
	WinPercent *float64   `json:"win_percent"` // รูปแบบ prizes
	Prizes     *[]PrizeIn `json:"prizes"`      // รูปแบบ prizes

	Spec    gameRateCore.GameSpec `json:"-"` // หลัง Validate
	Values  []gameRateCore.Step   `json:"-"` // รูปแบบ steps
	Scratch gameRateCore.Scratch  `json:"-"` // รูปแบบ prizes
}

// resolveGame — game_code ต้องส่ง · ไม่มี = 406401 · เก็บรูปแบบอัตราไว้ใน r.Spec (หาครั้งเดียว)
func (r *RatesInput) resolveGame() error {
	spec, err := checkGame(r.GameCode)
	r.Spec = spec
	return err
}

// validateFields — หลัง resolveGame: field ของรูปแบบนั้น → ค่าแต่ละตัว → กฎรูปแบบใน core
func (r *RatesInput) validateFields() error {
	switch r.Spec.Kind {
	case gameRateCore.KindSteps:
		return r.validateSteps()
	case gameRateCore.KindPrizes:
		return r.validatePrizes()
	}
	return apperr.ErrInternal
}

func (r *RatesInput) validateSteps() error {
	if r.WinPercent != nil {
		return invalid("win_percent", "ไม่ใช่ค่าของเกมนี้ (ตั้งโอกาสชนะทีละขั้นใน steps)", "is not part of this game (set win_percent per step)")
	}
	if r.Prizes != nil {
		return invalid("prizes", "ไม่ใช่ค่าของเกมนี้", "is not part of this game")
	}
	if r.Steps == nil {
		return invalid("steps", "ต้องส่ง", "is required")
	}
	r.Values = make([]gameRateCore.Step, len(*r.Steps))
	for i, s := range *r.Steps {
		p := fmt.Sprintf("steps[%d].", i)
		if s.DrawPercent != nil {
			return invalid(p+"draw_percent", "ไม่มีในเกมนี้ (ไม่มีผลเสมอ)", "is not part of this game (no draw)")
		}
		var err error
		if r.Values[i].Multiplier, err = decimal4(p+"multiplier", s.Multiplier); err != nil {
			return err
		}
		if r.Values[i].WinPercent, err = decimal4(p+"win_percent", s.WinPercent); err != nil {
			return err
		}
	}
	return FormatError(gameRateCore.CheckStepsFormat(r.Values, r.Spec.StepCount), "steps", r.Spec.StepCount)
}

func (r *RatesInput) validatePrizes() error {
	if r.Steps != nil {
		return invalid("steps", "ไม่ใช่ค่าของเกมนี้", "is not part of this game")
	}
	var err error
	if r.Scratch.WinPercent, err = decimal4("win_percent", r.WinPercent); err != nil {
		return err
	}
	if r.Prizes == nil {
		return invalid("prizes", "ต้องส่ง", "is required")
	}
	r.Scratch.Prizes = make([]gameRateCore.Prize, len(*r.Prizes))
	for i, p := range *r.Prizes {
		f := fmt.Sprintf("prizes[%d].", i)
		if r.Scratch.Prizes[i].Multiplier, err = decimal4(f+"multiplier", p.Multiplier); err != nil {
			return err
		}
		if r.Scratch.Prizes[i].SharePercent, err = decimal4(f+"share_percent", p.SharePercent); err != nil {
			return err
		}
	}
	return FormatError(gameRateCore.CheckScratchFormat(r.Scratch), "prizes", 0)
}

// FormatError — FormatIssue จาก core → 422 พร้อม field และช่วงที่ตั้งได้ · service ใช้ตรวจค่าตอน rollback ด้วย (RATE-05A)
func FormatError(is *gameRateCore.FormatIssue, array string, stepCount int) error {
	if is == nil {
		return nil
	}
	field := is.Field
	if is.Index >= 0 {
		field = fmt.Sprintf("%s[%d].%s", array, is.Index, is.Field)
	}
	switch is.Reason {
	case gameRateCore.ReasonStepCount:
		return invalid(field, fmt.Sprintf("ต้องส่งครบ %d ขั้น", stepCount), fmt.Sprintf("must contain exactly %d steps", stepCount))
	case gameRateCore.ReasonMultiplierRange:
		if array == "steps" {
			return invalid(field, "ต้องอยู่ระหว่าง 1 ถึง 1,000", "must be between 1 and 1,000")
		}
		return invalid(field, "ต้องมากกว่า 0 และไม่เกิน 1,000", "must be greater than 0 and at most 1,000")
	case gameRateCore.ReasonMultiplierOrder:
		return invalid(field, "ต้องมากกว่าตัวคูณของขั้นก่อนหน้า", "must be greater than the previous step's multiplier")
	case gameRateCore.ReasonWinRange:
		m := utils.FormatNum(gameRateCore.MaxWinPercent)
		return invalid(field, "ต้องมากกว่า 0 และไม่เกิน "+m, "must be greater than 0 and at most "+m)
	case gameRateCore.ReasonPrizeCount:
		return invalid(field, fmt.Sprintf("ต้องมี %d–%d แถว", gameRateCore.MinPrizes, gameRateCore.MaxPrizes),
			fmt.Sprintf("must have %d–%d rows", gameRateCore.MinPrizes, gameRateCore.MaxPrizes))
	case gameRateCore.ReasonPrizeDuplicate:
		return invalid(field, "ตัวคูณซ้ำกับแถวอื่น", "duplicates another row's multiplier")
	case gameRateCore.ReasonShareRange:
		return invalid(field, "ต้องมากกว่า 0 และไม่เกิน 100", "must be greater than 0 and at most 100")
	}
	return apperr.ErrValidation
}

// CheckRequest — RE3 detail/check (ไม่บันทึก · ไม่มี base_version / passcode)
type CheckRequest struct {
	RatesInput
}

// Validate — game_code → ค่าอัตรา
func (r *CheckRequest) Validate() error {
	if err := r.resolveGame(); err != nil {
		return err
	}
	return r.validateFields()
}

// checkBaseVersion — base_version ต้องส่ง จำนวนเต็ม ≥ 0 (RATE-05B · ยังไม่เคยตั้ง = 0)
func checkBaseVersion(v *int) error {
	if v == nil {
		return invalid("base_version", "ต้องส่ง (เวอร์ชันที่หน้าจอโหลดมา · ยังไม่เคยตั้ง = 0)", "is required (the version you loaded · 0 if never set)")
	}
	if *v < 0 {
		return invalid("base_version", "ต้องไม่ติดลบ", "must not be negative")
	}
	return nil
}

// UpdateRequest — RE4 detail/update · passcode ตรวจที่ middleware RequirePasscode
type UpdateRequest struct {
	RatesInput
	BaseVersion *int   `json:"base_version"`
	Passcode    string `json:"passcode"`
}

// Validate — game_code → base_version → ค่าอัตรา
func (r *UpdateRequest) Validate() error {
	if err := r.resolveGame(); err != nil {
		return err
	}
	if err := checkBaseVersion(r.BaseVersion); err != nil {
		return err
	}
	return r.validateFields()
}

// VersionsListRequest — RE5 versions/list · page / limit ใช้ utils.NewPage
type VersionsListRequest struct {
	GameCode string `json:"game_code"`
	Page     int    `json:"page"`
	Limit    int    `json:"limit"`
}

func (r *VersionsListRequest) Validate() error {
	_, err := checkGame(r.GameCode)
	return err
}

// VersionRequest — RE6 versions/detail/get
type VersionRequest struct {
	GameCode string `json:"game_code"`
	Version  int    `json:"version"`
}

func (r *VersionRequest) Validate() error {
	if _, err := checkGame(r.GameCode); err != nil {
		return err
	}
	if r.Version < 1 {
		return invalid("version", "ต้องเป็นจำนวนเต็มตั้งแต่ 1", "must be an integer of at least 1")
	}
	return nil
}

// StatusRequest — RE8 status/update (สวิตช์ปิดเกมทั้งระบบ — RATE-01B) · passcode ตรวจที่ middleware RequirePasscode
type StatusRequest struct {
	GameCode   string `json:"game_code"`
	StatusGame *bool  `json:"status_game"`
	Passcode   string `json:"passcode"`
}

// Validate — game_code → status_game
func (r *StatusRequest) Validate() error {
	if _, err := checkGame(r.GameCode); err != nil {
		return err
	}
	if r.StatusGame == nil {
		return invalid("status_game", "ต้องส่ง (true เปิด · false ปิด)", "is required (true = open · false = closed)")
	}
	return nil
}

// RollbackRequest — RE7 versions/rollback · passcode ตรวจที่ middleware RequirePasscode
type RollbackRequest struct {
	GameCode    string `json:"game_code"`
	Version     int    `json:"version"`
	BaseVersion *int   `json:"base_version"`
	Passcode    string `json:"passcode"`
}

// Validate — game_code → version → base_version
func (r *RollbackRequest) Validate() error {
	if _, err := checkGame(r.GameCode); err != nil {
		return err
	}
	if r.Version < 1 {
		return invalid("version", "ต้องเป็นจำนวนเต็มตั้งแต่ 1", "must be an integer of at least 1")
	}
	return checkBaseVersion(r.BaseVersion)
}
