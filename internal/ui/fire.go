package ui

import (
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var fireChars = []string{" ", "░", "▒", "▓", "█"}

var firePalette = []lipgloss.Color{
	lipgloss.Color("16"),  // black
	lipgloss.Color("52"),  // dark red
	lipgloss.Color("88"),  // red
	lipgloss.Color("124"), // bright red
	lipgloss.Color("160"), // red-orange
	lipgloss.Color("196"), // orange
	lipgloss.Color("208"), // light orange
	lipgloss.Color("214"), // amber
	lipgloss.Color("220"), // yellow-orange
	lipgloss.Color("226"), // yellow
	lipgloss.Color("228"), // light yellow
	lipgloss.Color("230"), // near white
}

type fireGrid struct {
	width  int
	height int
	pixels []int // heat values, 0 = cold, len(firePalette)-1 = hot
}

func newFireGrid(width, height int) *fireGrid {
	f := &fireGrid{
		width:  width,
		height: height,
		pixels: make([]int, width*height),
	}
	maxHeat := len(firePalette) - 1
	for x := 0; x < width; x++ {
		f.pixels[(height-1)*width+x] = maxHeat
	}
	return f
}

func (f *fireGrid) step() {
	for y := 0; y < f.height-1; y++ {
		for x := 0; x < f.width; x++ {
			srcY := y + 1
			srcX := x + rand.Intn(3) - 1
			if srcX < 0 {
				srcX = 0
			}
			if srcX >= f.width {
				srcX = f.width - 1
			}
			decay := rand.Intn(2)
			heat := f.pixels[srcY*f.width+srcX] - decay
			if heat < 0 {
				heat = 0
			}
			f.pixels[y*f.width+x] = heat
		}
	}
}

func (f *fireGrid) render() string {
	var b strings.Builder
	maxHeat := len(firePalette) - 1
	for y := 0; y < f.height; y++ {
		if y > 0 {
			b.WriteString("\n")
		}
		for x := 0; x < f.width; x++ {
			heat := f.pixels[y*f.width+x]
			charIdx := heat * (len(fireChars) - 1) / maxHeat
			color := firePalette[heat]
			style := lipgloss.NewStyle().Foreground(color)
			b.WriteString(style.Render(fireChars[charIdx]))
		}
	}
	return b.String()
}

func (f *fireGrid) renderWithMessage(msg string) string {
	var b strings.Builder
	maxHeat := len(firePalette) - 1
	msgRow := f.height / 3
	msgStart := (f.width - len(msg)) / 2
	msgEnd := msgStart + len(msg)
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Bold(true)

	for y := 0; y < f.height; y++ {
		if y > 0 {
			b.WriteString("\n")
		}
		for x := 0; x < f.width; x++ {
			if y == msgRow && x >= msgStart && x < msgEnd {
				ch := string(msg[x-msgStart])
				b.WriteString(msgStyle.Render(ch))
				continue
			}
			heat := f.pixels[y*f.width+x]
			charIdx := heat * (len(fireChars) - 1) / maxHeat
			color := firePalette[heat]
			style := lipgloss.NewStyle().Foreground(color)
			b.WriteString(style.Render(fireChars[charIdx]))
		}
	}
	return b.String()
}
