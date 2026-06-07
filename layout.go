package main

import (
	"fmt"
	"sort"
	"strings"
)

// LayoutField is a FieldSpec placed at a resolved offset within the struct.
type LayoutField struct {
	FieldSpec
	Offset   int
	PadAfter int // internal padding bytes between this field and the next
}

// Layout is the fully computed memory layout of a struct.
type Layout struct {
	Fields      []LayoutField
	StructSize  int
	StructAlign int
	InternalPad int // total avoidable padding
	TrailingPad int // unavoidable trailing padding
}

func computeLayout(fields []FieldSpec) Layout {
	if len(fields) == 0 {
		return Layout{StructAlign: 1}
	}

	structAlign := 1
	for _, f := range fields {
		if f.Align > structAlign {
			structAlign = f.Align
		}
	}

	layoutFields := make([]LayoutField, len(fields))
	offset := 0
	internalPad := 0

	for i, f := range fields {
		// Advance offset to the next multiple of this field's alignment.
		if rem := offset % f.Align; rem != 0 {
			pad := f.Align - rem
			if i > 0 {
				layoutFields[i-1].PadAfter = pad
				internalPad += pad
			}
			offset += pad
		}
		layoutFields[i] = LayoutField{
			FieldSpec: f,
			Offset:    offset,
		}
		offset += f.Size
	}

	trailingPad := 0
	if rem := offset % structAlign; rem != 0 {
		trailingPad = structAlign - rem
	}

	return Layout{
		Fields:      layoutFields,
		StructSize:  offset + trailingPad,
		StructAlign: structAlign,
		InternalPad: internalPad,
		TrailingPad: trailingPad,
	}
}

// optimizeLayout sorts fields largest-alignment-first (greedy optimal) then
// recomputes the layout.
func optimizeLayout(fields []FieldSpec) Layout {
	sorted := make([]FieldSpec, len(fields))
	copy(sorted, fields)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Align != sorted[j].Align {
			return sorted[i].Align > sorted[j].Align
		}
		return sorted[i].Size > sorted[j].Size
	})
	return computeLayout(sorted)
}

// formatStruct renders the optimized field order as valid Go source.
func formatStruct(name string, layout Layout) string {
	maxName := 0
	for _, f := range layout.Fields {
		if len(f.Name) > maxName {
			maxName = len(f.Name)
		}
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", name))
	for _, f := range layout.Fields {
		sb.WriteString(fmt.Sprintf("\t%-*s %s\n", maxName, f.Name, f.TypeStr))
	}
	sb.WriteString("}")
	return sb.String()
}
