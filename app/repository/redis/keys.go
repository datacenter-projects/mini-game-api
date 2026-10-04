package redis

import (
	"fmt"
	"strings"

	"app/app/models"
)

// key ทั้งหมดของระบบประกาศที่นี่ที่เดียว

// backoffice session — docs/modules/agent_auth.md, docs/modules/agent_auth_phase2.md หัวข้อ 6
func keyBOSession(sid string) string { return "bo:sess:" + sid }

// keyBOAccountSession — pointer ไปยัง session ปัจจุบันของบัญชี: bo:sess:agent:{id} / bo:sess:sub:{id}
func keyBOAccountSession(t models.AccountType, id uint) string {
	return fmt.Sprintf("bo:sess:%s:%d", strings.ToLower(string(t)), id)
}

func keyBOLoginFail(username string) string  { return "bo:login:fail:" + username }
func keyBOLoginBlock(username string) string { return "bo:login:block:" + username }
func keyBOLoginIP(ip string) string          { return "bo:login:ip:" + ip }

func keyBOPasscodeFail(t models.AccountType, id uint) string {
	return fmt.Sprintf("bo:passcode:fail:%s:%d", strings.ToLower(string(t)), id)
}

func keyBOPasscodeBlock(t models.AccountType, id uint) string {
	return fmt.Sprintf("bo:passcode:block:%s:%d", strings.ToLower(string(t)), id)
}
