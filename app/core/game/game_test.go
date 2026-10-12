package game

import (
	"reflect"
	"testing"
)

func TestAll(t *testing.T) { // RATE-01A: 3 เกม เรียงตามลำดับที่แสดง
	got := All()
	want := []Game{{"minigame", CoinToss}, {"minigame", RockPaperScissors}, {"minigame", ScratchCard}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}
}

func TestGroupOfAndIsGame(t *testing.T) {
	tests := []struct {
		code  string
		group Group
		ok    bool
	}{
		{CoinToss, GroupMinigame, true},
		{RockPaperScissors, GroupMinigame, true},
		{ScratchCard, GroupMinigame, true},
		{"slot", "", false},
		{"", "", false},
		{"Coin_Toss", "", false}, // ตัวพิมพ์ต้องตรง
	}
	for _, tt := range tests {
		g, ok := GroupOf(tt.code)
		if g != tt.group || ok != tt.ok {
			t.Errorf("GroupOf(%q) = %q, %v · want %q, %v", tt.code, g, ok, tt.group, tt.ok)
		}
		if IsGame(tt.code) != tt.ok {
			t.Errorf("IsGame(%q) = %v, want %v", tt.code, !tt.ok, tt.ok)
		}
	}
}

func TestGamesOf(t *testing.T) {
	if n := len(GamesOf(GroupMinigame)); n != 3 {
		t.Fatalf("GamesOf(minigame) = %d เกม, want 3", n)
	}
	if GamesOf("unknown") != nil {
		t.Fatal("กลุ่มที่ไม่มีต้องได้ nil")
	}
	if !reflect.DeepEqual(Groups(), []Group{GroupMinigame}) {
		t.Fatalf("Groups() = %v", Groups())
	}
}
