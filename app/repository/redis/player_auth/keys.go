// Package playerauth คือ Redis ของ Player Auth — key ทั้งหมดประกาศที่นี่ที่เดียว · prefix: pl:sess: pl:kicked: pl:login: pl:op:
// spec: docs/modules/player_auth.md หัวข้อ 6
package playerauth

import "fmt"

func keySession(sid string) string { return "pl:sess:" + sid }

// keyMemberSession — pointer ไปยัง session ปัจจุบันของ Member (1 session ต่อคน — PA-22)
func keyMemberSession(memberID uint) string { return fmt.Sprintf("pl:sess:member:%d", memberID) }

// keyKicked — เครื่องหมายว่า session นี้ถูก login ใหม่เตะ (ให้ตอบ 407204 แทน 407203)
func keyKicked(sid string) string { return "pl:kicked:" + sid }

func keyLoginFail(username string) string  { return "pl:login:fail:" + username }
func keyLoginBlock(username string) string { return "pl:login:block:" + username }
func keyLoginIP(ip string) string          { return "pl:login:ip:" + ip }
func keyOperatorIP(ip string) string       { return "pl:op:ip:" + ip }
