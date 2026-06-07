package main

import (
	"fmt"
	"strings"
)

// ANSI escape sequences.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiFgWhite = "\033[97m"
	ansiFgBlack = "\033[30m"
	ansiBgRed   = "\033[41m"  // internal (avoidable) padding
	ansiBgGray  = "\033[100m" // trailing (unavoidable) padding
)

// Cycling background colors for fields.
var fieldBG = []string{
	"\033[44m", // blue
	"\033[42m", // green
	"\033[43m", // yellow
	"\033[46m", // cyan
	"\033[45m", // magenta
}

// byteKind categorises each byte of the struct's memory.
type byteKind int

const (
	bkField byteKind = iota
	bkInternalPad
	bkTrailingPad
)

type byteCell struct {
	kind     byteKind
	fieldIdx int
	label    string // exactly cellW runes
}

const cellW = 5 // visible characters per cell

func buildByteMap(layout Layout) []byteCell {
	cells := make([]byteCell, layout.StructSize)

	for fi, field := range layout.Fields {
		for b := 0; b < field.Size; b++ {
			label := strings.Repeat(" ", cellW)
			if b == 0 {
				name := field.Name
				if len(name) > cellW {
					name = name[:cellW-1] + "…"
				}
				label = fmt.Sprintf("%-*s", cellW, name)
			}
			cells[field.Offset+b] = byteCell{
				kind:     bkField,
				fieldIdx: fi,
				label:    label,
			}
		}
		for p := 0; p < field.PadAfter; p++ {
			cells[field.Offset+field.Size+p] = byteCell{
				kind:  bkInternalPad,
				label: strings.Repeat("▒", cellW),
			}
		}
	}

	for i := layout.StructSize - layout.TrailingPad; i < layout.StructSize; i++ {
		cells[i] = byteCell{
			kind:  bkTrailingPad,
			label: strings.Repeat("░", cellW),
		}
	}

	return cells
}

func coloredCell(c byteCell) string {
	var color string
	switch c.kind {
	case bkInternalPad:
		color = ansiBgRed + ansiFgWhite
	case bkTrailingPad:
		color = ansiBgGray + ansiFgBlack
	default:
		color = fieldBG[c.fieldIdx%len(fieldBG)] + ansiFgWhite
	}
	return color + c.label + ansiReset
}

// renderGrid produces the colored byte-grid for a layout.
func renderGrid(layout Layout) string {
	if layout.StructSize == 0 {
		return "  (empty struct)\n"
	}

	cells := buildByteMap(layout)
	rowWidth := max(layout.StructAlign, 1)

	sep := strings.Repeat("─", cellW)

	// Build the top, mid, and bottom border for rowWidth columns.
	borderLine := func(left, cross, right string) string {
		parts := make([]string, rowWidth)
		for i := range parts {
			parts[i] = sep
		}
		return left + strings.Join(parts, cross) + right
	}

	top := borderLine("┌", "┬", "┐")
	mid := borderLine("├", "┼", "┤")
	bot := borderLine("└", "┴", "┘")

	// Column offset header.
	var header strings.Builder
	header.WriteString("       ") // indent matching row-label width
	for col := 0; col < rowWidth; col++ {
		label := fmt.Sprintf("%-*d", cellW, col)
		header.WriteString(label)
		if col < rowWidth-1 {
			header.WriteString(" ")
		}
	}
	header.WriteString("\n")

	var sb strings.Builder
	sb.WriteString(header.String())

	rows := (layout.StructSize + rowWidth - 1) / rowWidth
	for row := 0; row < rows; row++ {
		startByte := row * rowWidth

		if row == 0 {
			sb.WriteString("       " + top + "\n")
		} else {
			sb.WriteString("       " + mid + "\n")
		}

		// Row label (absolute byte offset).
		fmt.Fprintf(&sb, " %4d: │", startByte)

		for col := 0; col < rowWidth; col++ {
			idx := startByte + col
			if idx < len(cells) {
				sb.WriteString(coloredCell(cells[idx]))
			} else {
				sb.WriteString(strings.Repeat(" ", cellW))
			}
			sb.WriteString("│")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("       " + bot + "\n")
	return sb.String()
}

// renderLegend prints the color legend for field names in the layout.
func renderLegend(layout Layout) string {
	seen := make(map[string]bool)
	var sb strings.Builder
	sb.WriteString("  Fields: ")
	first := true
	for _, f := range layout.Fields {
		if seen[f.Name] {
			continue
		}
		seen[f.Name] = true
		idx := indexOf(layout.Fields, f.Name)
		color := fieldBG[idx%len(fieldBG)] + ansiFgWhite
		if !first {
			sb.WriteString("  ")
		}
		label := f.Name
		if len(label) > cellW {
			label = label[:cellW-1] + "…"
		}
		sb.WriteString(color + fmt.Sprintf(" %-*s ", cellW, label) + ansiReset)
		first = false
	}
	sb.WriteString("  ")
	sb.WriteString(ansiBgRed + ansiFgWhite + " ▒▒▒ " + ansiReset + " internal pad (avoidable)")
	sb.WriteString("  ")
	sb.WriteString(ansiBgGray + ansiFgBlack + " ░░░ " + ansiReset + " trailing pad (unavoidable)")
	sb.WriteString("\n")
	return sb.String()
}

func indexOf(fields []LayoutField, name string) int {
	for i, f := range fields {
		if f.Name == name {
			return i
		}
	}
	return 0
}

// banner prints a titled section header.
func banner(title string) string {
	line := strings.Repeat("─", 60)
	return fmt.Sprintf("\n%s%s%s\n%s\n", ansiBold, title, ansiReset, line)
}

// statsLine prints size/alignment/waste info for a layout.
func statsLine(layout Layout) string {
	totalWaste := layout.InternalPad + layout.TrailingPad
	return fmt.Sprintf(
		"  size: %d bytes   alignment: %d   internal padding: %d bytes   trailing padding: %d bytes   total waste: %d bytes\n",
		layout.StructSize, layout.StructAlign,
		layout.InternalPad, layout.TrailingPad, totalWaste,
	)
}
