package account

import (
	"net"
	"net/url"
	"strings"
)

// กฎของข้อมูลรับรอง API — docs/modules/account.md ACC-06, ACC-07 (ช่วงกว้าง / IP ภายใน — lead A7 2026-10-09)

const (
	CallbackURLMaxLen = 500
	AllowedIPsMax     = 50
)

// CallbackViolation — ผลตรวจลิงก์ตอบกลับ (ACC-06)
type CallbackViolation int

const (
	CallbackOK CallbackViolation = iota
	CallbackTooLong
	CallbackInvalidURL
	CallbackNotHTTPS
)

// NormalizeCallbackURL ตัดช่องว่างหัวท้ายแล้วตรวจ (ACC-06) · ว่าง = ยังไม่ตั้ง (คืน "" กับ CallbackOK)
func NormalizeCallbackURL(raw string) (string, CallbackViolation) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", CallbackOK
	}
	if len(s) > CallbackURLMaxLen {
		return "", CallbackTooLong
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return "", CallbackInvalidURL
	}
	if u.Scheme != "https" {
		return "", CallbackNotHTTPS
	}
	return s, CallbackOK
}

// IPViolation — ผลตรวจรายการ IP ที่อนุญาต (ACC-07)
type IPViolation int

const (
	IPOK IPViolation = iota
	IPTooMany
	IPInvalid     // ไม่ใช่ IPv4 หรือ CIDR ของ IPv4
	IPHostBitsSet // CIDR ที่มีบิตของ host เช่น 1.2.3.4/24 (ต้องเป็น 1.2.3.0/24)
	IPDuplicate   // ซ้ำหลังแปลงรูปแบบแล้ว เช่น 1.2.3.4 กับ 1.2.3.4/32
	IPTooWide     // ช่วงกว้างกว่า /16 เช่น 0.0.0.0/0 · 10.0.0.0/8 (lead A7 2026-10-09)
	IPReserved    // อยู่ในช่วง IP ภายใน / พิเศษ (ReservedRanges — lead A7 2026-10-09)
)

// MinAllowedPrefix — ช่วงที่กว้างที่สุดที่ตั้งได้ (ACC-07 · lead A7)
const MinAllowedPrefix = 16

// ReservedRanges — IP ภายใน / พิเศษที่ห้ามใส่ (ACC-07 · lead A7) · ทุกช่วงกว้างไม่น้อยกว่า /16
// จึงเช็คแค่ว่าที่อยู่เครือข่ายของรายการอยู่ในช่วงไหนก็พอ (รายการกว้างสุดคือ /16)
var ReservedRanges = []string{"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16"}

var reservedNets = func() []*net.IPNet {
	out := make([]*net.IPNet, len(ReservedRanges))
	for i, r := range ReservedRanges {
		_, n, _ := net.ParseCIDR(r)
		out[i] = n
	}
	return out
}()

func isReserved(ip net.IP) bool {
	for _, n := range reservedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// NormalizeAllowedIPs แปลงทุกรายการเป็น CIDR ของ IPv4 (IP เดี่ยว → /32) และตรวจตาม ACC-07
// คืน index ของรายการที่ผิด (-1 = ไม่มี) เพื่อให้ msg บอกได้ว่าผิดที่รายการไหน
func NormalizeAllowedIPs(list []string) ([]string, IPViolation, int) {
	if len(list) > AllowedIPsMax {
		return nil, IPTooMany, -1
	}
	out := make([]string, 0, len(list))
	seen := make(map[string]bool, len(list))
	for i, raw := range list {
		cidr, v := normalizeIPv4(strings.TrimSpace(raw))
		if v != IPOK {
			return nil, v, i
		}
		if seen[cidr] {
			return nil, IPDuplicate, i
		}
		seen[cidr] = true
		out = append(out, cidr)
	}
	return out, IPOK, -1
}

func normalizeIPv4(s string) (string, IPViolation) {
	if strings.Contains(s, ":") { // IPv6 รวมรูปแบบ ::ffff:1.2.3.4
		return "", IPInvalid
	}
	if strings.Contains(s, "/") {
		ip, network, err := net.ParseCIDR(s)
		if err != nil || ip.To4() == nil {
			return "", IPInvalid
		}
		if !ip.Equal(network.IP) {
			return "", IPHostBitsSet
		}
		if ones, _ := network.Mask.Size(); ones < MinAllowedPrefix {
			return "", IPTooWide
		}
		if isReserved(network.IP) {
			return "", IPReserved
		}
		return network.String(), IPOK
	}
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		return "", IPInvalid
	}
	if isReserved(ip) {
		return "", IPReserved
	}
	return ip.To4().String() + "/32", IPOK
}
