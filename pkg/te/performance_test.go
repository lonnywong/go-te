package te

import (
	"fmt"
	"runtime"
	"testing"
)

// These benchmarks exercise representative workloads without weakening any
// pyte/ESCTest assertions. Profiling remains an explicit separate workflow.
func BenchmarkNewScreen(b *testing.B) {
	for _, size := range [][2]int{{40, 8}, {80, 24}, {100, 30}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				runtime.KeepAlive(NewScreen(size[0], size[1]))
			}
		})
	}
}
func BenchmarkReset(b *testing.B) {
	s := NewScreen(80, 24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Reset()
	}
}
func BenchmarkScroll(b *testing.B) {
	for _, reverse := range []bool{false, true} {
		name := "up"
		if reverse {
			name = "down"
		}
		b.Run(name, func(b *testing.B) {
			s := NewScreen(80, 24)
			if !reverse {
				s.Cursor.Row = 23
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if reverse {
					s.ReverseIndex()
				} else {
					s.Index()
				}
			}
		})
	}
}
func BenchmarkDraw(b *testing.B) {
	for _, text := range []string{"hello terminal", "café 日本語 e\u0301 👩‍💻"} {
		b.Run(text, func(b *testing.B) {
			s := NewScreen(80, 24)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Draw(text)
				s.CarriageReturn()
			}
		})
	}
}
func BenchmarkDisplay(b *testing.B) {
	s := NewScreen(80, 24)
	s.Draw("hello terminal")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runtime.KeepAlive(s.Display())
	}
}
func BenchmarkResize(b *testing.B) {
	s := NewScreen(80, 24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Resize(24, 132)
		s.Resize(24, 80)
	}
}
func BenchmarkStreamScroll(b *testing.B) {
	s := NewScreen(80, 24)
	stream := NewStream(s, false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := stream.Feed("hello world\r\n"); err != nil {
			b.Fatal(err)
		}
	}
}
