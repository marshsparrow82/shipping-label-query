package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Pos is a 1-indexed line/column location in a manifest source file.
type Pos struct {
	Line int
	Col  int
}

// Dims holds a package's length, width and height in a single unit.
type Dims struct {
	L, W, H float64
}

// Shipment is one parsed "shipment ... end" block.
type Shipment struct {
	ID         string
	Weight     float64
	WeightUnit string // "lb" or "kg"
	Dims       Dims
	DimsUnit   string // "in" or "cm"
	Service    string
	Pos        Pos // location of the "shipment" keyword, used in later error messages
}

// ParseError is a single, precisely located problem found while reading a
// manifest. Error() renders it the way a compiler would: file:line:col,
// the offending source line, and a caret under the exact character.
type ParseError struct {
	File string
	Pos  Pos
	Line string
	Msg  string
}

func (e *ParseError) Error() string {
	const indent = "    "
	caret := strings.Repeat(" ", e.Pos.Col-1) + "^"
	return fmt.Sprintf("%s:%d:%d: %s\n%s%s\n%s%s",
		e.File, e.Pos.Line, e.Pos.Col, e.Msg, indent, e.Line, indent, caret)
}

// ParseErrors is every problem found in one pass over a file. We collect all
// of them instead of stopping at the first, same as a real compiler would.
type ParseErrors []*ParseError

func (errs ParseErrors) Error() string {
	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(e.Error())
	}
	return b.String()
}

type token struct {
	text string
	col  int // 1-indexed
}

// tokenizeLine splits a line on runs of spaces/tabs, recording the 1-indexed
// column each token starts at. Columns are byte offsets: manifests are
// expected to be ASCII, so that's equivalent to rune offsets here.
func tokenizeLine(line string) []token {
	var toks []token
	i, n := 0, len(line)
	for i < n {
		for i < n && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}
		start := i
		for i < n && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		toks = append(toks, token{text: line[start:i], col: start + 1})
	}
	return toks
}

var requiredFields = []string{"weight", "dims", "service"}

// ParseManifest reads a manifest file and returns every shipment it
// describes. On any problem it returns a ParseErrors listing every issue
// found, not just the first.
func ParseManifest(file, src string) ([]Shipment, error) {
	lines := strings.Split(src, "\n")

	var errs ParseErrors
	var shipments []Shipment
	seenIDs := map[string]Pos{}

	addErr := func(lineNo, col int, msg string) {
		lineText := ""
		if lineNo-1 >= 0 && lineNo-1 < len(lines) {
			lineText = lines[lineNo-1]
		}
		errs = append(errs, &ParseError{File: file, Pos: Pos{lineNo, col}, Line: lineText, Msg: msg})
	}

	var cur *Shipment
	var curFields map[string]bool
	var curLine int

	for lineNo := 1; lineNo <= len(lines); lineNo++ {
		toks := tokenizeLine(lines[lineNo-1])
		if len(toks) == 0 {
			continue
		}
		kw := toks[0]

		if cur == nil {
			if kw.text != "shipment" {
				addErr(lineNo, kw.col, fmt.Sprintf("unexpected %q, expected \"shipment\"", kw.text))
				continue
			}
			if len(toks) != 2 {
				addErr(lineNo, kw.col, "\"shipment\" requires exactly one id, like: shipment SH1001")
				continue
			}
			id := toks[1]
			if prev, ok := seenIDs[id.text]; ok {
				addErr(lineNo, id.col, fmt.Sprintf("duplicate shipment id %q (first defined at line %d)", id.text, prev.Line))
			} else {
				seenIDs[id.text] = Pos{lineNo, id.col}
			}
			cur = &Shipment{ID: id.text, Pos: Pos{lineNo, kw.col}}
			curFields = map[string]bool{}
			curLine = lineNo
			continue
		}

		switch kw.text {
		case "weight":
			if curFields["weight"] {
				addErr(lineNo, kw.col, "duplicate \"weight\" field")
				continue
			}
			if len(toks) != 3 {
				addErr(lineNo, kw.col, "\"weight\" requires a number and a unit, like: weight 12.5 lb")
				continue
			}
			val, err := strconv.ParseFloat(toks[1].text, 64)
			if err != nil {
				addErr(lineNo, toks[1].col, fmt.Sprintf("invalid number %q", toks[1].text))
				continue
			}
			if val <= 0 {
				addErr(lineNo, toks[1].col, "weight must be greater than zero")
				continue
			}
			unit := toks[2].text
			if unit != "lb" && unit != "kg" {
				addErr(lineNo, toks[2].col, fmt.Sprintf("invalid unit %q (expected lb or kg)", unit))
				continue
			}
			cur.Weight = val
			cur.WeightUnit = unit
			curFields["weight"] = true

		case "dims":
			if curFields["dims"] {
				addErr(lineNo, kw.col, "duplicate \"dims\" field")
				continue
			}
			if len(toks) != 3 {
				addErr(lineNo, kw.col, "\"dims\" requires LxWxH and a unit, like: dims 10x8x6 in")
				continue
			}
			parts := strings.Split(strings.ToLower(toks[1].text), "x")
			if len(parts) != 3 {
				addErr(lineNo, toks[1].col, fmt.Sprintf("invalid dimensions %q (expected LxWxH, like 10x8x6)", toks[1].text))
				continue
			}
			var nums [3]float64
			bad := false
			col := toks[1].col
			for i, p := range parts {
				v, err := strconv.ParseFloat(p, 64)
				if err != nil || v <= 0 {
					addErr(lineNo, col, fmt.Sprintf("invalid dimension %q in %q (must be a positive number)", p, toks[1].text))
					bad = true
				}
				nums[i] = v
				col += len(p) + 1 // +1 for the separating "x"
			}
			if bad {
				continue
			}
			unit := toks[2].text
			if unit != "in" && unit != "cm" {
				addErr(lineNo, toks[2].col, fmt.Sprintf("invalid unit %q (expected in or cm)", unit))
				continue
			}
			cur.Dims = Dims{L: nums[0], W: nums[1], H: nums[2]}
			cur.DimsUnit = unit
			curFields["dims"] = true

		case "service":
			if curFields["service"] {
				addErr(lineNo, kw.col, "duplicate \"service\" field")
				continue
			}
			if len(toks) != 2 {
				addErr(lineNo, kw.col, "\"service\" requires exactly one value, like: service ground")
				continue
			}
			cur.Service = toks[1].text
			curFields["service"] = true

		case "end":
			if len(toks) != 1 {
				addErr(lineNo, toks[1].col, fmt.Sprintf("unexpected %q after \"end\"", toks[1].text))
			}
			for _, req := range requiredFields {
				if !curFields[req] {
					addErr(curLine, cur.Pos.Col, fmt.Sprintf("shipment %q is missing required field %q", cur.ID, req))
				}
			}
			if cur.WeightUnit != "" && cur.DimsUnit != "" {
				imperial := cur.WeightUnit == "lb"
				if imperial != (cur.DimsUnit == "in") {
					addErr(curLine, cur.Pos.Col, fmt.Sprintf(
						"shipment %q mixes unit systems (weight in %s, dims in %s); use lb with in, or kg with cm",
						cur.ID, cur.WeightUnit, cur.DimsUnit))
				}
			}
			shipments = append(shipments, *cur)
			cur = nil
			curFields = nil

		default:
			addErr(lineNo, kw.col, fmt.Sprintf("unknown field %q (expected weight, dims, service, or end)", kw.text))
		}
	}

	if cur != nil {
		addErr(curLine, cur.Pos.Col, fmt.Sprintf("shipment %q is missing \"end\"", cur.ID))
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return shipments, nil
}
