package middleware

import (
	"net/netip"
	"strings"

	"app/pkg/configs"

	"github.com/gofiber/fiber/v2"
)

// ClientIP คือ IP ของ client จริง — ใช้กับ rate limit · whitelist · last_login_ip ทุกที่ (แทน c.IP())
//
// ทำไมไม่ใช้ c.IP(): เมื่อตั้ง ProxyHeader = X-Forwarded-For c.IP() คืนค่า header ทั้งก้อน ("client, proxy1")
// ทำให้ whitelist อ่านไม่ออก และ client ปลอม "X-Forwarded-For: <ค่าสุ่ม>" เพื่อเปลี่ยน key ของ rate limit ได้ทุก request
//
// กติกา (ไล่ XFF จากขวาข้าม proxy ที่เชื่อถือ — ไม่ใช้ IP ซ้ายสุดที่ client ปลอมได้):
//  0. TRUSTED_PROXIES ถูก validate + parse ครั้งเดียวตอน configs.Load (พิมพ์ผิด = boot ไม่ขึ้น) — ที่นี่อ่านจาก Cfg.TrustedProxyPrefixes
//  1. ไม่ตั้ง TRUSTED_PROXIES → IP ของ connection ตรงๆ (ไม่เชื่อ header)
//  2. connection มาจากที่ไม่ใช่ proxy ที่เชื่อถือ → IP ของ connection (ไม่เชื่อ header)
//  3. มาจาก proxy ที่เชื่อถือ → ไล่ X-Forwarded-For จากขวาไปซ้าย ข้ามทุก IP ที่อยู่ใน TRUSTED_PROXIES แล้วคืน IP แรกที่ไม่ใช่ proxy
//     (= ผู้ที่ proxy ชั้นนอกสุดของเราเห็นจริง — ค่าที่ client ใส่เองจะอยู่ทางซ้ายของค่านี้ จึงไม่มีผล)
func ClientIP(c *fiber.Ctx) string {
	var xff []string
	for _, v := range c.Request().Header.PeekAll(fiber.HeaderXForwardedFor) { // header หลายบรรทัดต่อกัน
		xff = append(xff, string(v))
	}
	return clientIPFrom(c.Context().RemoteIP().String(), xff, configs.Cfg.TrustedProxyPrefixes)
}

func inPrefixes(a netip.Addr, ps []netip.Prefix) bool {
	for _, p := range ps {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// parseHop อ่านค่า 1 รายการใน X-Forwarded-For — รับ "1.2.3.4" · "1.2.3.4:5678" · "[::1]:80" · "::1"
func parseHop(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap(), true
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	return netip.Addr{}, false
}

// clientIPFrom คือตรรกะล้วนของ ClientIP (ทดสอบได้โดยไม่ต้องมี connection จริง)
// peer = IP ของ connection ที่ต่อมาถึงเรา · xff = ค่า X-Forwarded-For ทุกบรรทัด · trusted = TRUSTED_PROXIES ที่แปลงแล้ว
func clientIPFrom(peer string, xff []string, trusted []netip.Prefix) string {
	peerAddr, err := netip.ParseAddr(peer)
	if err != nil {
		return peer // อ่านไม่ได้ก็ไม่แต่งค่าใหม่
	}
	peerAddr = peerAddr.Unmap()
	if len(trusted) == 0 || !inPrefixes(peerAddr, trusted) {
		return peerAddr.String()
	}

	var hops []string
	for _, line := range xff {
		hops = append(hops, strings.Split(line, ",")...)
	}
	var leftmost netip.Addr
	for i := len(hops) - 1; i >= 0; i-- {
		if strings.TrimSpace(hops[i]) == "" {
			continue // ช่องว่าง / comma ซ้อน
		}
		a, ok := parseHop(hops[i])
		if !ok {
			return peerAddr.String() // มีค่าที่อ่านไม่ได้แทรกในสาย — ไม่เดา ใช้ IP ของ proxy ที่ต่อมา (fail-closed)
		}
		if !inPrefixes(a, trusted) {
			return a.String() // IP แรกจากขวาที่ไม่ใช่ proxy ของเรา
		}
		leftmost = a
	}
	if leftmost.IsValid() { // ทุกค่าใน XFF เป็น proxy ของเราเอง (เรียกจากภายในเครือข่าย) → ตัวซ้ายสุดคือต้นทาง
		return leftmost.String()
	}
	return peerAddr.String() // ไม่มี XFF
}
