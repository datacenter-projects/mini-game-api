package account

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeCallbackURL(t *testing.T) { // ACC-06
	long := "https://a.example/" + strings.Repeat("x", CallbackURLMaxLen-len("https://a.example/")+1)
	tests := []struct {
		in   string
		want string
		v    CallbackViolation
	}{
		{"", "", CallbackOK},
		{"   ", "", CallbackOK},
		{" https://api.customer.example/minigame ", "https://api.customer.example/minigame", CallbackOK},
		{"https://api.customer.example:8443/cb?x=1", "https://api.customer.example:8443/cb?x=1", CallbackOK},
		{"http://api.customer.example", "", CallbackNotHTTPS},
		{"ftp://api.customer.example", "", CallbackNotHTTPS},
		{"HTTPS://api.customer.example", "HTTPS://api.customer.example", CallbackOK}, // scheme ไม่สนตัวพิมพ์ตามมาตรฐาน URL
		{"https://", "", CallbackInvalidURL},
		{"https:///path", "", CallbackInvalidURL},
		{"not a url", "", CallbackInvalidURL},
		{long, "", CallbackTooLong},
	}
	for _, tt := range tests {
		got, v := NormalizeCallbackURL(tt.in)
		if got != tt.want || v != tt.v {
			t.Errorf("NormalizeCallbackURL(%q) = %q, %d · want %q, %d", tt.in, got, v, tt.want, tt.v)
		}
	}
}

func TestNormalizeAllowedIPs(t *testing.T) { // ACC-07
	tests := []struct {
		name string
		in   []string
		want []string
		v    IPViolation
		idx  int
	}{
		{"ว่าง", []string{}, []string{}, IPOK, -1},
		{"IP เดี่ยวเป็น /32", []string{" 1.2.3.4 "}, []string{"1.2.3.4/32"}, IPOK, -1},
		{"CIDR", []string{"198.51.100.0/24", "10.0.0.0/8"}, []string{"198.51.100.0/24", "10.0.0.0/8"}, IPOK, -1},
		{"/32 ตรงๆ", []string{"1.2.3.4/32"}, []string{"1.2.3.4/32"}, IPOK, -1},
		{"IPv6", []string{"1.2.3.4", "2001:db8::1"}, nil, IPInvalid, 1},
		{"IPv4-mapped IPv6", []string{"::ffff:1.2.3.4"}, nil, IPInvalid, 0},
		{"เลขเกิน", []string{"300.1.1.1"}, nil, IPInvalid, 0},
		{"mask เกิน", []string{"1.2.3.4/33"}, nil, IPInvalid, 0},
		{"มีบิตของ host", []string{"1.2.3.4/24"}, nil, IPHostBitsSet, 0},
		{"ซ้ำหลังแปลง", []string{"1.2.3.4", "1.2.3.4/32"}, nil, IPDuplicate, 1},
		{"ข้อความ", []string{"abc"}, nil, IPInvalid, 0},
	}
	for _, tt := range tests {
		got, v, idx := NormalizeAllowedIPs(tt.in)
		if v != tt.v || idx != tt.idx || (tt.want != nil && !reflect.DeepEqual(got, tt.want)) {
			t.Errorf("%s: got %v, %d, %d · want %v, %d, %d", tt.name, got, v, idx, tt.want, tt.v, tt.idx)
		}
	}

	max := make([]string, AllowedIPsMax)
	for i := range max {
		max[i] = fmt.Sprintf("10.0.0.%d", i)
	}
	if _, v, _ := NormalizeAllowedIPs(max); v != IPOK {
		t.Errorf("%d รายการต้องผ่าน ได้ %d", AllowedIPsMax, v)
	}
	if _, v, _ := NormalizeAllowedIPs(append(max, "10.0.1.1")); v != IPTooMany {
		t.Errorf("%d รายการต้อง IPTooMany ได้ %d", AllowedIPsMax+1, v)
	}
}
