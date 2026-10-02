package redis

import "fmt"

// key ทั้งหมดของระบบประกาศที่นี่ที่เดียว

// backoffice session — docs/modules/agent_auth.md หัวข้อ 6
func keyBOSession(sid string) string         { return "bo:sess:" + sid }
func keyBOAgentSession(agentID uint) string  { return fmt.Sprintf("bo:sess:agent:%d", agentID) }
func keyBOLoginFail(username string) string  { return "bo:login:fail:" + username }
func keyBOLoginBlock(username string) string { return "bo:login:block:" + username }
func keyBOLoginIP(ip string) string          { return "bo:login:ip:" + ip }
