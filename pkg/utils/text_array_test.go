package utils

import (
	"reflect"
	"testing"
)

func TestTextArrayRoundTrip(t *testing.T) {
	tests := []TextArray{
		{},
		{"1.2.3.4/32"},
		{"1.2.3.4/32", "198.51.100.0/24"},
		{`a"b`, `c\d`, "e,f", " sp "},
	}
	for _, in := range tests {
		v, err := in.Value()
		if err != nil {
			t.Fatal(err)
		}
		var out TextArray
		if err := out.Scan(v); err != nil {
			t.Fatalf("%v: %v", in, err)
		}
		if !reflect.DeepEqual(out, in) {
			t.Errorf("round trip %v → %v → %v", in, v, out)
		}
	}
}

func TestTextArrayScan(t *testing.T) {
	tests := []struct {
		src  any
		want TextArray
	}{
		{"{}", TextArray{}},
		{"{1.2.3.4/32,10.0.0.0/8}", TextArray{"1.2.3.4/32", "10.0.0.0/8"}}, // Postgres ส่งแบบไม่มี " ได้
		{[]byte(`{"a b",c}`), TextArray{"a b", "c"}},
		{nil, TextArray{}},
	}
	for _, tt := range tests {
		var out TextArray
		if err := out.Scan(tt.src); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(out, tt.want) {
			t.Errorf("Scan(%v) = %v, want %v", tt.src, out, tt.want)
		}
	}
	var out TextArray
	if err := out.Scan("not-array"); err == nil {
		t.Error("ต้อง error")
	}
}
