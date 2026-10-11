package te

import (
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
)

// DisplayAnsi returns the visible screen as ANSI-styled strings, one per row.
// Each row resets SGR attributes at its start and end, so rows can be rendered
// independently. Wide-character padding is omitted, as in Display.
// Cell protection flags are not visual attributes and are not exported.
func (s *Screen) DisplayAnsi() []string {
	lines := make([]string, s.Lines)
	for row := 0; row < s.Lines; row++ {
		var b strings.Builder
		b.Grow(s.Columns)
		var previous Attr
		first := true
		for col := 0; col < s.Columns; {
			cell := s.Buffer[row][col]
			if cell.Data == "" {
				col++
				continue
			}
			if first || cell.Attr != previous {
				b.WriteString(ansiRendition(cell.Attr))
				previous = cell.Attr
				first = false
			}
			b.WriteString(cell.Data)
			col += maxInt(1, runewidth.StringWidth(cell.Data))
		}
		if first {
			b.WriteString(ControlCSI + "0m")
		}
		b.WriteString(ControlCSI + "0m")
		lines[row] = b.String()
	}
	return lines
}

func ansiRendition(attr Attr) string {
	var b strings.Builder
	b.WriteString(ControlCSI + "0")
	for i, enabled := range [...]bool{
		attr.Bold, attr.Italics, attr.Underline, attr.Blink,
		attr.Reverse, attr.Conceal, attr.Strikethrough,
	} {
		if enabled {
			b.WriteByte(';')
			b.WriteString(strconv.Itoa([...]int{1, 3, 4, 5, 7, 8, 9}[i]))
		}
	}
	writeANSIColor(&b, attr.Fg, 30)
	writeANSIColor(&b, attr.Bg, 40)
	b.WriteByte('m')
	return b.String()
}

func writeANSIColor(b *strings.Builder, color Color, base int) {
	switch color.Mode {
	case ColorANSI16:
		if color.Index < 8 {
			b.WriteString(";" + strconv.Itoa(base+int(color.Index)))
		} else if color.Index < 16 {
			b.WriteString(";" + strconv.Itoa(base+60+int(color.Index)-8))
		}
	case ColorANSI256:
		b.WriteString(";" + strconv.Itoa(base+8) + ";5;" + strconv.Itoa(int(color.Index)))
	case ColorTrueColor:
		if len(color.Name) != 6 {
			return
		}
		rgb, err := strconv.ParseUint(color.Name, 16, 24)
		if err != nil {
			return
		}
		b.WriteString(";" + strconv.Itoa(base+8) + ";2;" +
			strconv.Itoa(int(rgb>>16)) + ";" +
			strconv.Itoa(int((rgb>>8)&255)) + ";" +
			strconv.Itoa(int(rgb&255)))
	}
}
