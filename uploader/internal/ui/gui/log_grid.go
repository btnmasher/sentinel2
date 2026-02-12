//go:build !headless

package gui

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var (
	ansiColorDefault = &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 220, G: 220, B: 220, A: 255}}
	ansiColorDebug   = &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 140, G: 180, B: 255, A: 255}}
	ansiColorInfo    = &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 180, G: 240, B: 180, A: 255}}
	ansiColorWarn    = &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 255, G: 220, B: 130, A: 255}}
	ansiColorError   = &widget.CustomTextGridStyle{FGColor: color.NRGBA{R: 255, G: 150, B: 150, A: 255}}
	ansiBoldDefault  = &widget.CustomTextGridStyle{
		FGColor:   color.NRGBA{R: 220, G: 220, B: 220, A: 255},
		TextStyle: fyne.TextStyle{Bold: true},
	}
	ansiStyleCache = map[string]*widget.CustomTextGridStyle{}
)

func parseANSITextGridRows(input string) []widget.TextGridRow {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	out := make([]widget.TextGridRow, 0, len(lines))
	for _, line := range lines {
		out = append(out, parseANSITextGridRow(line))
	}
	return out
}

func parseANSITextGridRow(line string) widget.TextGridRow {
	row := widget.TextGridRow{Cells: make([]widget.TextGridCell, 0, len(line))}
	var style widget.TextGridStyle = ansiColorDefault
	i := 0
	for i < len(line) {
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			end := strings.IndexByte(line[i+2:], 'm')
			if end >= 0 {
				seq := line[i+2 : i+2+end]
				style = parseANSIStyle(seq, style)
				i += end + 3
				continue
			}
		}
		row.Cells = append(row.Cells, widget.TextGridCell{
			Rune:  rune(line[i]),
			Style: style,
		})
		i++
	}
	if len(row.Cells) == 0 {
		row.Cells = append(row.Cells, widget.TextGridCell{Rune: ' ', Style: ansiColorDefault})
	}
	return row
}

func parseANSIStyle(seq string, current widget.TextGridStyle) widget.TextGridStyle {
	if seq == "" {
		return ansiColorDefault
	}
	parts := strings.Split(seq, ";")
	style := current
	for i := 0; i < len(parts); i++ {
		code, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}
		switch code {
		case 0:
			style = ansiColorDefault
		case 1:
			style = ansiBoldDefault
		case 34, 94:
			style = ansiColorDebug
		case 32, 92:
			style = ansiColorInfo
		case 33, 93:
			style = ansiColorWarn
		case 31, 91:
			style = ansiColorError
		case 39:
			style = ansiColorDefault
		case 38:
			// Extended foreground color: 38;5;<index> or 38;2;<r>;<g>;<b>.
			if i+2 < len(parts) && parts[i+1] == "5" {
				if idx, idxErr := strconv.Atoi(parts[i+2]); idxErr == nil {
					style = ansiStyleFromColor(ansi256ToColor(idx), false)
				}
				i += 2
				continue
			}
			if i+4 < len(parts) && parts[i+1] == "2" {
				r, rErr := strconv.Atoi(parts[i+2])
				g, gErr := strconv.Atoi(parts[i+3])
				b, bErr := strconv.Atoi(parts[i+4])
				if rErr == nil && gErr == nil && bErr == nil {
					style = ansiStyleFromColor(color.NRGBA{
						R: uint8(clampColor(r)),
						G: uint8(clampColor(g)),
						B: uint8(clampColor(b)),
						A: 255,
					}, false)
				}
				i += 4
				continue
			}
		}
	}
	return style
}

func ansiStyleFromColor(fg color.NRGBA, bold bool) widget.TextGridStyle {
	key := strconv.Itoa(int(fg.R)) + ":" + strconv.Itoa(int(fg.G)) + ":" + strconv.Itoa(int(fg.B)) + ":" + strconv.FormatBool(bold)
	if cached, ok := ansiStyleCache[key]; ok {
		return cached
	}
	style := &widget.CustomTextGridStyle{
		FGColor: fg,
	}
	if bold {
		style.TextStyle = fyne.TextStyle{Bold: true}
	}
	ansiStyleCache[key] = style
	return style
}

func clampColor(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func ansi256ToColor(index int) color.NRGBA {
	if index < 0 {
		index = 0
	}
	if index > 255 {
		index = 255
	}

	base := []color.NRGBA{
		{0, 0, 0, 255},
		{128, 0, 0, 255},
		{0, 128, 0, 255},
		{128, 128, 0, 255},
		{0, 0, 128, 255},
		{128, 0, 128, 255},
		{0, 128, 128, 255},
		{192, 192, 192, 255},
		{128, 128, 128, 255},
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{255, 255, 0, 255},
		{0, 0, 255, 255},
		{255, 0, 255, 255},
		{0, 255, 255, 255},
		{255, 255, 255, 255},
	}
	if index < 16 {
		return base[index]
	}
	if index <= 231 {
		c := index - 16
		r := c / 36
		g := (c % 36) / 6
		b := c % 6
		scale := []uint8{0, 95, 135, 175, 215, 255}
		return color.NRGBA{R: scale[r], G: scale[g], B: scale[b], A: 255}
	}
	v := uint8(8 + (index-232)*10)
	return color.NRGBA{R: v, G: v, B: v, A: 255}
}
