package configs

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

// ParseTrustedProxies แปลง TRUSTED_PROXIES (IP เดี่ยวหรือ CIDR) เป็น prefix — รายการที่พิมพ์ผิดคืน error (S1-a)
// เดิมข้ามเงียบ ทำให้ทุกคนได้ IP ของ proxy → rate limit ใช้ร่วมกันทั้งระบบ + whitelist ของ operator ไม่ผ่าน · ตอนนี้ boot ไม่ขึ้นแทน
func ParseTrustedProxies(list []string) ([]netip.Prefix, error) {
	out := make([]netip.Prefix, 0, len(list))
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if p, err := netip.ParsePrefix(s); err == nil {
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(s)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: %q is not an IP address or CIDR", s)
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// validateLobbyURL — PLAYER_LOBBY_URL ถ้าตั้งต้องเป็น URL แบบ absolute (http / https + host) (N5)
// ค่าผิดรูปแบบเดิมไปพังตอน transfer/login (ผู้เล่นถูกเตะ session แต่ไม่ได้ link) · ไม่ตั้ง = ปล่อย (transfer/login ตอบ 503)
func validateLobbyURL(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("PLAYER_LOBBY_URL must be an absolute http(s) URL with a host, got %q", s)
	}
	return nil
}
