package utils

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// TextArray — คอลัมน์ TEXT[] ของ Postgres (เช่น api_credential_logs.old_ips)
// เขียนเป็น array literal {"a","b"} · อ่านกลับเป็น []string
type TextArray []string

func (a TextArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, s := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String(), nil
}

func (a *TextArray) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case nil:
		*a = TextArray{}
		return nil
	default:
		return fmt.Errorf("TextArray: unsupported type %T", src)
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return fmt.Errorf("TextArray: invalid array literal %q", s)
	}
	body := s[1 : len(s)-1]
	out := TextArray{}
	for i := 0; i < len(body); {
		var item strings.Builder
		if body[i] == '"' {
			i++
			for i < len(body) && body[i] != '"' {
				if body[i] == '\\' && i+1 < len(body) {
					i++
				}
				item.WriteByte(body[i])
				i++
			}
			i++ // ปิด "
		} else {
			for i < len(body) && body[i] != ',' {
				item.WriteByte(body[i])
				i++
			}
		}
		out = append(out, item.String())
		i++ // ข้าม ,
	}
	*a = out
	return nil
}
