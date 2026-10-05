package utils

import "github.com/goccy/go-json"

// JSONString คือ field ที่ต้องส่งมาเป็น JSON string เท่านั้น
//
// ใช้แทน string ธรรมดาเมื่อค่าที่เป็นตัวเลขต้องไม่ถูกรับเป็น number (เช่น passcode — เลข 0 นำหน้าหาย)
// ส่งเป็นชนิดอื่น parse ผ่านแต่ IsString = false ให้ Validate() ตอบ 422 พร้อมบอก field ได้
// (ถ้าใช้ string ตรงๆ BodyParser จะล้มทั้ง body แล้วตอบ 400)
type JSONString struct {
	Value    string
	Present  bool // มี key นี้ใน body และไม่ใช่ null
	IsString bool
}

func (s *JSONString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = JSONString{}
		return nil
	}
	s.Present = true
	if err := json.Unmarshal(b, &s.Value); err != nil {
		s.Value, s.IsString = "", false
		return nil
	}
	s.IsString = true
	return nil
}
