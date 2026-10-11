package te

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestDisplayAnsiColors(t *testing.T) {
	tests := []struct {
		name  string
		color Color
		fg    string
		bg    string
	}{
		{"default", Color{Mode: ColorDefault, Name: "default"}, "", ""},
		{"zero value", Color{}, "", ""},
		{"indexed zero", Color{Mode: ColorANSI256, Index: 0}, ";38;5;0", ";48;5;0"},
		{"indexed max", Color{Mode: ColorANSI256, Index: 255}, ";38;5;255", ";48;5;255"},
		{"true color", Color{Mode: ColorTrueColor, Name: "01aBff"}, ";38;2;1;171;255", ";48;2;1;171;255"},
		{"black", Color{Mode: ColorTrueColor, Name: "000000"}, ";38;2;0;0;0", ";48;2;0;0;0"},
		{"invalid hex", Color{Mode: ColorTrueColor, Name: "zzzzzz"}, "", ""},
		{"invalid length", Color{Mode: ColorTrueColor, Name: "fff"}, "", ""},
		{"invalid ANSI16", Color{Mode: ColorANSI16, Index: 16}, "", ""},
		{"unknown mode", Color{Mode: ColorMode(255)}, "", ""},
	}
	for index := 0; index < 16; index++ {
		fg, bg := 30+index, 40+index
		if index >= 8 {
			fg, bg = 90+index-8, 100+index-8
		}
		tests = append(tests, struct {
			name  string
			color Color
			fg    string
			bg    string
		}{
			"ANSI16 " + strconv.Itoa(index),
			Color{Mode: ColorANSI16, Index: uint8(index)},
			";" + strconv.Itoa(fg), ";" + strconv.Itoa(bg),
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewScreen(1, 1)
			screen.Buffer[0][0] = Cell{Data: "X", Attr: Attr{Fg: tt.color, Bg: tt.color}}
			want := []string{ControlCSI + "0" + tt.fg + tt.bg + "mX" + ControlCSI + "0m"}
			if got := screen.DisplayAnsi(); !reflect.DeepEqual(got, want) {
				t.Fatalf("DisplayAnsi() = %q, want %q", got, want)
			}
		})
	}
}

func TestDisplayAnsiStyles(t *testing.T) {
	tests := []struct {
		name string
		attr Attr
		sgr  string
	}{
		{"bold", Attr{Bold: true}, ";1"},
		{"italics", Attr{Italics: true}, ";3"},
		{"underline", Attr{Underline: true}, ";4"},
		{"blink", Attr{Blink: true}, ";5"},
		{"reverse", Attr{Reverse: true}, ";7"},
		{"conceal", Attr{Conceal: true}, ";8"},
		{"strikethrough", Attr{Strikethrough: true}, ";9"},
		{"protection is not visual", Attr{Protected: true, ISOProtected: true}, ""},
		{"combined", Attr{
			Bold: true, Italics: true, Underline: true, Blink: true,
			Reverse: true, Conceal: true, Strikethrough: true,
		}, ";1;3;4;5;7;8;9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := NewScreen(2, 1)
			screen.Buffer[0][0] = Cell{Data: "X", Attr: tt.attr}
			screen.Buffer[0][1] = Cell{Data: "Y", Attr: tt.attr}
			want := []string{ControlCSI + "0" + tt.sgr + "mXY" + ControlCSI + "0m"}
			if got := screen.DisplayAnsi(); !reflect.DeepEqual(got, want) {
				t.Fatalf("DisplayAnsi() = %q, want %q", got, want)
			}
		})
	}
}

func TestDisplayAnsiTransitions(t *testing.T) {
	screen := NewScreen(6, 2)
	stream := NewStream(screen, false)
	input := "\x1b[1;31;44mAB\x1b[22;39mC\x1b[0mD\x1b[32m \x1b[2;1HE"
	if err := stream.Feed(input); err != nil {
		t.Fatal(err)
	}
	before := screen.LinesCells()
	cursor := screen.Cursor
	dirty := make(map[int]struct{}, len(screen.Dirty))
	for row := range screen.Dirty {
		dirty[row] = struct{}{}
	}
	want := []string{
		"\x1b[0;1;31;44mAB\x1b[0;44mC\x1b[0mD\x1b[0;32m \x1b[0m \x1b[0m",
		"\x1b[0;32mE\x1b[0m     \x1b[0m",
	}
	got := screen.DisplayAnsi()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DisplayAnsi() = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(screen.LinesCells(), before) || screen.Cursor != cursor ||
		!reflect.DeepEqual(screen.Dirty, dirty) {
		t.Fatal("DisplayAnsi modified screen state")
	}
	if plain := screen.Display(); !reflect.DeepEqual(plain, []string{"ABCD  ", "E     "}) {
		t.Fatalf("Display() = %q", plain)
	}
	// Replay each row with unrelated initial styling to verify row independence.
	for row, line := range got {
		target := NewScreen(screen.Columns, 1)
		targetStream := NewStream(target, false)
		if err := targetStream.Feed("\x1b[1;3;4;5;7;9;35;45m" + line); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(target.LinesCells()[0], before[row]) {
			t.Fatalf("row %d did not round-trip: %#v", row, target.LinesCells()[0])
		}
		if target.Cursor.Attr != target.defaultAttr() {
			t.Fatalf("row %d left styling enabled", row)
		}
	}
}

func TestDisplayAnsiUnicodeAndBlankRows(t *testing.T) {
	screen := NewScreen(10, 2)
	screen.SelectGraphicRendition([]int{38, 2, 12, 34, 56}, false)
	screen.Draw("界e\u0301👩‍💻")
	// Padding attributes must not produce extra style transitions.
	screen.Buffer[0][1].Attr = Attr{Bold: true}
	before := screen.Display()
	for row, line := range screen.DisplayAnsi() {
		if !strings.HasPrefix(line, ControlCSI+"0") || !strings.HasSuffix(line, ControlCSI+"0m") {
			t.Fatalf("row %d lacks resets: %q", row, line)
		}
		target := NewScreen(screen.Columns, 1)
		if err := NewStream(target, false).Feed(line); err != nil {
			t.Fatal(err)
		}
		if got := target.Display()[0]; got != before[row] {
			t.Fatalf("row %d text = %q, want %q", row, got, before[row])
		}
		if row == 0 && strings.Contains(line, ";1m") {
			t.Fatalf("rendered padding attributes: %q", line)
		}
	}
	// Empty data, including zero-width data, must not cause an infinite loop.
	screen.Buffer[0][0] = Cell{Data: "\u0301"}
	screen.Buffer[0][1] = Cell{}
	screen.Buffer[0][2] = Cell{}
	if got := screen.DisplayAnsi()[0]; !strings.Contains(got, "\u0301") {
		t.Fatalf("lost zero-width data: %q", got)
	}
	screen.Buffer[0] = make([]Cell, screen.Columns)
	if got := screen.DisplayAnsi()[0]; got != "\x1b[0m\x1b[0m" {
		t.Fatalf("empty row = %q", got)
	}
}
