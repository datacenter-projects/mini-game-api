// seed_mock_users สร้างบัญชี mock ทุกประเภทใต้ superadmin ที่มีอยู่แล้ว (ใช้ทดสอบหน้าบ้าน / Postman)
//
//	go run ./scripts/seed_mock_users -root superadmin1
//	go run ./scripts/seed_mock_users -root superadmin1 -prefix mock2
//
// สร้างผ่าน service ตัวเดียวกับ API (กฎ PT / สกุล / ยอดเงิน / log ครบ) · รหัสผ่านพิมพ์ทาง stdin ครั้งเดียว ใช้ร่วมทุกบัญชี
// รันซ้ำได้: request_id คิดจาก username → บัญชีที่สร้างแล้วถูกข้าม (รหัสผ่านเดิมไม่เปลี่ยน)
// จบแล้วพิมพ์ตาราง username / รหัสผ่าน · ทุกบัญชีต้องตั้ง passcode ตอน login ครั้งแรก
//
// สาย (ค่าได้รับลดชั้นละ 10 · ใต้ Seamless Master ได้เท่าที่ Master ได้ · force / remain / commission = 0):
//
//	root
//	├─ COMPANY_TRANSFER ─┬─ SHARE_B2B → AGENT → AGENT
//	│                    └─ SHARE_B2C ─┬─ AGENT → MEMBER
//	│                                  └─ MEMBER
//	├─ COMPANY_SEAMLESS_1TO1 → MEMBER
//	├─ COMPANY_SEAMLESS_RESELLER → SHARE_RESELLER → AGENT → MEMBER
//	└─ COMPANY_SEAMLESS_MASTER → SHARE_MASTER → AGENT → MEMBER
//
// ฝั่ง Transfer ทุกบัญชีเหลือยอด THB 10,000 (ผู้สร้างโอนให้เท่ากับ 10,000 × จำนวนบัญชีในสายนั้น) · Seamless ไม่มียอดเงิน
// บัญชีฝั่ง agent ได้ sub `{username}@staff` บัญชีละ 1 ตัว สิทธิ์ view ทุกเมนูของเจ้าของ
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	memberManagementService "app/app/service/member_management"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/term"
)

const (
	balancePerAccount = 10000 // บาท ต่อบัญชี ฝั่ง Transfer
	subSuffix         = "staff"
)

type node struct {
	suffix string
	typ    agentManagementCore.UserType
	kids   []node
}

func tree() []node {
	const (
		ct  = agentManagementCore.UserTypeCompanyTransfer
		c11 = agentManagementCore.UserTypeCompanySeamless1to1
		cr  = agentManagementCore.UserTypeCompanySeamlessReseller
		cm  = agentManagementCore.UserTypeCompanySeamlessMaster
		sbb = agentManagementCore.UserTypeShareB2B
		sbc = agentManagementCore.UserTypeShareB2C
		ag  = agentManagementCore.UserTypeAgent
		mem = agentManagementCore.UserTypeMember
	)
	return []node{
		{"comtransfer", ct, []node{
			{"shareb2b", sbb, []node{{"agentb2b", ag, []node{{"agentb2b2", ag, nil}}}}},
			{"shareb2c", sbc, []node{
				{"agentb2c", ag, []node{{"memagentb2c", mem, nil}}},
				{"memshareb2c", mem, nil},
			}},
		}},
		{"com1to1", c11, []node{{"mem1to1", mem, nil}}},
		{"comreseller", cr, []node{{"sharereseller", sbc, []node{{"agentreseller", ag, []node{{"memreseller", mem, nil}}}}}}},
		{"commaster", cm, []node{{"sharemaster", sbc, []node{{"agentmaster", ag, []node{{"memmaster", mem, nil}}}}}}},
	}
}

func size(n node) int64 {
	s := int64(1)
	for _, k := range n.kids {
		s += size(k)
	}
	return s
}

type row struct{ userType, username, parent string }

type seeder struct {
	ctx      context.Context
	prefix   string
	password string
	meta     agentAuthService.RequestMeta
	rows     []row
}

func main() {
	root := flag.String("root", "superadmin1", "username ของ superadmin ที่จะสร้างสายไว้ข้างใต้")
	prefix := flag.String("prefix", "mock", "ขึ้นต้น username (a-z 0-9)")
	flag.Parse()

	if err := run(*root, *prefix); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(rootName, prefix string) error {
	prefix = agentManagementCore.NormalizeUsername(prefix)
	if !agentManagementCore.IsValidUsername(prefix + "x") {
		return fmt.Errorf("-prefix ต้องเป็น a-z 0-9")
	}
	password, err := readPassword()
	if err != nil {
		return err
	}
	if err := agentAuthDto.PasswordPolicyError("password", agentAuthCore.CheckPasswordPolicy(password)); err != nil {
		return fmt.Errorf("%s", apperr.From(err).MsgTH)
	}

	if err := configs.Load(); err != nil {
		return err
	}
	if configs.Cfg.AppEnv == "prod" {
		return fmt.Errorf("APP_ENV=prod: seed_mock_users ห้ามรันบน production")
	}
	utils.SetPasswordCost(configs.Cfg.Auth.PasswordCost)
	logger.InitLogger(true)
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}

	r, err := agentAuthPostgres.GetUserAgentForLoginRepository(database.DBConn, agentAuthCore.NormalizeUsername(rootName))
	if err != nil {
		return fmt.Errorf("หา %q ไม่เจอ: %w", rootName, err)
	}
	if r.Role != models.AgentRoleSuperAdmin {
		return fmt.Errorf("%q ไม่ใช่ SUPERADMIN", rootName)
	}

	s := &seeder{ctx: context.Background(), prefix: prefix, password: password, meta: agentAuthService.RequestMeta{IP: "127.0.0.1", UserAgent: "seed_mock_users"}}
	rootActor := actorOf(r.ID, r.ParentID, r.Username, r.Role)
	for _, n := range tree() {
		if err := s.create(rootActor, agentManagementCore.UserTypeSuperadmin, agentManagementCore.FullPTBP, false, n, 1); err != nil {
			return err
		}
	}
	s.print(configs.Cfg.AppEnv)
	return nil
}

func actorOf(id uint, parent *uint, username string, role models.AgentRole) agentAuthService.Actor {
	return agentAuthService.Actor{AccountType: models.AccountTypeAgent, AgentID: id, ParentID: parent, Username: username, Role: role,
		Status: models.AgentStatusActive, EffectiveStatus: models.AgentStatusActive, PasscodeSet: true}
}

// requestID — UUID ที่คิดจาก username · รันซ้ำได้ request เดิม → service ตอบบัญชีเดิม
func requestID(username string) string {
	h := sha256.Sum256([]byte("seed_mock_users:" + username))
	h[6] = (h[6] & 0x0f) | 0x40
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// give — ค่า pt_from_parent ที่ให้ลูก: ชั้นละ 10 ลดลง (90, 80, 70 …) · ใต้ Seamless Master ต้องเท่าที่ Master ได้รับ (MGMT-19)
func give(creator agentManagementCore.UserType, creatorReceived, depth int) int {
	if creator == agentManagementCore.UserTypeCompanySeamlessMaster {
		return creatorReceived
	}
	return min(agentManagementCore.FullPTBP-depth*1000, creatorReceived)
}

func pct(bp int) json.Number { return json.Number(fmt.Sprintf("%d.%02d", bp/100, bp%100)) }

func money(baht int64) json.Number { return json.Number(fmt.Sprintf("%d.00", baht)) }

func decode(body map[string]any, out interface{ Validate() error }) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return out.Validate()
}

func (s *seeder) create(creator agentAuthService.Actor, creatorType agentManagementCore.UserType, creatorReceived int,
	seamless bool, n node, depth int) error {
	username := s.prefix + n.suffix
	var balance map[string]any
	if !seamless && !agentManagementCore.IsSeamless(n.typ) {
		balance = map[string]any{"THB": money(balancePerAccount * size(n))}
	}

	if n.typ == agentManagementCore.UserTypeMember {
		body := map[string]any{"request_id": requestID(username), "username": username, "password": s.password, "name": username,
			"phone": "", "pt": map[string]any{"minigame": map[string]any{"commission_percent": 0}}}
		if balance != nil {
			body["balance"] = balance
		}
		var req memberManagementDto.CreateMemberRequest
		if err := decode(body, &req); err != nil {
			return s.fail(username, err)
		}
		if _, err := memberManagementService.CreateMemberService(s.ctx, creator, req, s.meta); err != nil {
			return s.fail(username, err)
		}
		s.rows = append(s.rows, row{string(n.typ), username, creator.Username})
		return nil
	}

	g := give(creatorType, creatorReceived, depth)
	body := map[string]any{"request_id": requestID(username), "user_type": string(n.typ), "username": username, "password": s.password,
		"name": username, "phone": "", "pt": map[string]any{"minigame": map[string]any{"pt_from_parent": pct(g), "force": 0,
			"remain_quota": 0, "commission_percent": 0, "status": true}}}
	switch agentManagementCore.CurrencyRequirementOf(creatorType, n.typ) {
	case agentManagementCore.CurrencyPickOne, agentManagementCore.CurrencyPickMany:
		body["currencies"] = []string{"THB"}
	}
	if balance != nil {
		body["balance"] = balance
	}
	var req agentManagementDto.CreateAgentRequest
	if err := decode(body, &req); err != nil {
		return s.fail(username, err)
	}
	res, err := agentManagementService.CreateAgentService(s.ctx, creator, req, s.meta)
	if err != nil {
		return s.fail(username, err)
	}
	newAcc, _ := agentManagementCore.ResolveNewAgent(creatorType, n.typ)
	s.rows = append(s.rows, row{res.UserType, res.Username, creator.Username})

	self := actorOf(res.ID, &creator.AgentID, res.Username, newAcc.Role)
	if err := s.createSub(self); err != nil {
		return err
	}
	childSeamless := seamless || agentManagementCore.IsSeamless(n.typ)
	for _, k := range n.kids {
		if err := s.create(self, newAcc.UserType, g, childSeamless, k, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// createSub — sub บัญชีละ 1 ตัว สิทธิ์ view ทุกเมนูของเจ้าของ · มีแล้วข้าม
func (s *seeder) createSub(owner agentAuthService.Actor) error {
	perms := map[string]any{}
	for _, m := range agentManagementCore.MenusForRole(owner.Role) {
		perms[string(m)] = string(agentManagementCore.LevelView)
	}
	body := map[string]any{"name_suffix": subSuffix, "password": s.password, "name": "staff" + owner.Username, "phone": "", "permissions": perms}
	username := owner.Username + "@" + subSuffix
	var req agentManagementDto.SubCreateRequest
	if err := decode(body, &req); err != nil {
		return s.fail(username, err)
	}
	if _, err := agentManagementService.CreateSubaccountService(s.ctx, owner, req, s.meta); err != nil && !errors.Is(err, apperr.ErrUsernameTaken) {
		return s.fail(username, err)
	}
	s.rows = append(s.rows, row{"SUB", username, owner.Username})
	return nil
}

func (s *seeder) fail(username string, err error) error {
	if ae := apperr.From(err); ae != nil && ae.Code != 0 && ae.Code != 500 {
		return fmt.Errorf("%s: %d %s", username, ae.Code, ae.MsgTH)
	}
	return fmt.Errorf("%s: %w", username, err)
}

func (s *seeder) print(env string) {
	fmt.Printf("\nสร้างครบ %d บัญชี (APP_ENV=%s) · รหัสผ่านทุกบัญชี: %s · login ครั้งแรกต้องตั้ง passcode\n\n", len(s.rows), env, s.password)
	fmt.Printf("%-26s %-28s %s\n", "user_type", "username", "ผู้สร้าง")
	fmt.Println(strings.Repeat("-", 80))
	for _, r := range s.rows {
		fmt.Printf("%-26s %-28s %s\n", r.userType, r.username, r.parent)
	}
}

func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	fmt.Fprint(os.Stderr, "Password (ใช้ทุกบัญชี): ")
	p1, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "Confirm password: ")
	p2, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(p1) != string(p2) {
		return "", fmt.Errorf("รหัสผ่านไม่ตรงกัน")
	}
	return string(p1), nil
}
