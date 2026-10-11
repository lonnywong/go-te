package te

import (
	"reflect"
	"strings"
	"testing"
)

func TestRecycledScrollRows(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		s := NewScreen(4, 4)
		for row := range s.Buffer {
			for col := range s.Buffer[row] {
				s.Buffer[row][col].Data = string(rune('A' + row))
			}
		}
		s.SetMargins(2, 3)
		s.SetMode([]int{ModeDECSCNM >> 5}, true)
		cell := s.defaultCell()
		outgoing := 1
		s.Cursor.Row = 2
		if reverse {
			outgoing = 2
			s.Cursor.Row = 1
		}
		original := &s.Buffer[outgoing][0]
		if reverse {
			s.ReverseIndex()
		} else {
			s.Index()
		}
		incoming := 2
		if reverse {
			incoming = 1
		}
		if &s.Buffer[incoming][0] != original {
			t.Fatal("outgoing row was not recycled")
		}
		for _, got := range s.Buffer[incoming] {
			if got != cell {
				t.Fatalf("recycled cell: %#v, want %#v", got, cell)
			}
		}
		if s.Buffer[0][0].Data != "A" || s.Buffer[3][0].Data != "D" {
			t.Fatal("scroll changed rows outside margins")
		}
		if &s.Buffer[1][0] == &s.Buffer[2][0] {
			t.Fatal("rows alias after scrolling")
		}
	}
}

func TestResetReusesAndClearsCells(t *testing.T) {
	s := NewScreen(8, 3)
	first := &s.Buffer[0][0]
	s.SetMode([]int{ModeDECSCNM >> 5}, true)
	s.Draw("hello界")
	s.Reset()
	if &s.Buffer[0][0] != first {
		t.Fatal("reset replaced cell storage")
	}
	want := NewScreen(8, 3)
	if !reflect.DeepEqual(s.Buffer, want.Buffer) {
		t.Fatal("reset retained cell data or attributes")
	}
	for _, size := range [][2]int{{0, 0}, {-1, -2}, {1, 1}} {
		got := NewScreen(size[0], size[1])
		if got.Columns != 1 || got.Lines != 1 {
			t.Fatal("dimensions not clamped")
		}
	}
}

func TestResizeReusesCapacityWithoutStaleCells(t *testing.T) {
	s := NewScreen(8, 3)
	s.Draw("abcdefgh")
	first := &s.Buffer[0][0]
	s.Resize(3, 4)
	s.Resize(3, 8)
	if &s.Buffer[0][0] != first {
		t.Fatal("resize did not reuse capacity")
	}
	for _, got := range s.Buffer[0][4:] {
		if got != s.defaultCell() {
			t.Fatal("resize exposed stale cells")
		}
	}
}

func TestTranslationFastPathParity(t *testing.T) {
	s := NewScreen(40, 8)
	for _, charset := range [][]rune{charsetLat1, charsetVT100} {
		s.G0 = charset
		for _, input := range []string{"ASCII text", "界 e\u0301", "\xff", "\xef\xbf\xbd", "lqkx"} {
			var want strings.Builder
			for _, r := range input {
				if r < 256 {
					r = charset[r]
				}
				want.WriteRune(r)
			}
			if got := s.translate(input); got != want.String() {
				t.Fatalf("translation %q: %q != %q", input, got, want.String())
			}
		}
	}
}
