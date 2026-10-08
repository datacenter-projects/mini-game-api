package account

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"app/pkg/apperr"
)

func TestSaveAPICredentialRequestValidate(t *testing.T) { // ACC-06, ACC-07, ACC-32
	tests := []struct {
		name    string
		body    string
		wantErr bool
		msgHas  string
		url     string
		ips     []string
	}{
		{"ครบถูก", `{"callback_url":" https://a.example/cb ","allowed_ips":["1.2.3.4","198.51.100.0/24"]}`, false, "", "https://a.example/cb", []string{"1.2.3.4/32", "198.51.100.0/24"}},
		{"ลิงก์ว่าง ไม่มี IP", `{"callback_url":"","allowed_ips":[]}`, false, "", "", []string{}},
		{"ไม่ส่ง callback_url", `{"allowed_ips":[]}`, true, "callback_url", "", nil},
		{"callback_url null", `{"callback_url":null,"allowed_ips":[]}`, true, "callback_url", "", nil},
		{"callback_url เป็นตัวเลข", `{"callback_url":1,"allowed_ips":[]}`, true, "callback_url", "", nil},
		{"http", `{"callback_url":"http://a.example","allowed_ips":[]}`, true, "https://", "", nil},
		{"ไม่ส่ง allowed_ips", `{"callback_url":""}`, true, "allowed_ips", "", nil},
		{"allowed_ips null", `{"callback_url":"","allowed_ips":null}`, true, "allowed_ips", "", nil},
		{"IPv6", `{"callback_url":"","allowed_ips":["1.2.3.4","::1"]}`, true, "allowed_ips[1]", "", nil},
		{"ซ้ำ", `{"callback_url":"","allowed_ips":["1.2.3.4","1.2.3.4/32"]}`, true, "allowed_ips[1]", "", nil},
		{"host bits", `{"callback_url":"","allowed_ips":["1.2.3.4/24"]}`, true, "allowed_ips[0]", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r SaveAPICredentialRequest
			if err := json.Unmarshal([]byte(tt.body), &r); err != nil {
				t.Fatal(err)
			}
			err := r.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var ae *apperr.Error
				if !errors.As(err, &ae) || ae.Code != 422 || !strings.Contains(ae.MsgEN+ae.MsgTH, tt.msgHas) {
					t.Fatalf("ต้องเป็น 422 ที่บอก %q ได้ %v", tt.msgHas, err)
				}
				return
			}
			if r.NormalizedCallbackURL != tt.url || !reflect.DeepEqual(r.NormalizedIPs, tt.ips) {
				t.Fatalf("got %q %v", r.NormalizedCallbackURL, r.NormalizedIPs)
			}
		})
	}
}
