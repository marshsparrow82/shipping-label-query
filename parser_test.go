package main

import (
	"errors"
	"strings"
	"testing"
)

// wantErr is one expected entry in a ParseErrors result: the exact position
// and a substring that must appear in the message. We check substrings
// rather than full text so the tests don't break on wording tweaks that
// don't change where the error points.
type wantErr struct {
	line, col int
	contains  string
}

func checkErrs(t *testing.T, err error, want []wantErr) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want %d", len(want))
	}
	var errs ParseErrors
	if !errors.As(err, &errs) {
		t.Fatalf("got error of type %T, want ParseErrors: %v", err, err)
	}
	if len(errs) != len(want) {
		t.Fatalf("got %d errors, want %d:\n%v", len(errs), len(want), err)
	}
	for i, w := range want {
		got := errs[i]
		if got.Pos.Line != w.line || got.Pos.Col != w.col {
			t.Errorf("error %d: got pos %d:%d, want %d:%d (msg: %s)", i, got.Pos.Line, got.Pos.Col, w.line, w.col, got.Msg)
		}
		if !strings.Contains(got.Msg, w.contains) {
			t.Errorf("error %d: got msg %q, want it to contain %q", i, got.Msg, w.contains)
		}
	}
}

func TestParseManifestErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []wantErr
	}{
		{
			name: "unexpected top-level token",
			src:  "foo\n",
			want: []wantErr{{1, 1, `unexpected "foo", expected "shipment"`}},
		},
		{
			name: "shipment missing id",
			src:  "shipment\n",
			want: []wantErr{{1, 1, `"shipment" requires exactly one id`}},
		},
		{
			name: "shipment too many args",
			src:  "shipment A B\n",
			want: []wantErr{{1, 1, `"shipment" requires exactly one id`}},
		},
		{
			name: "duplicate shipment id",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n" +
				"shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{{6, 10, `duplicate shipment id "A" (first defined at line 1)`}},
		},
		{
			name: "duplicate weight field",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"weight 2 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{{3, 1, `duplicate "weight" field`}},
		},
		{
			// A field that fails to parse is never marked "seen", so it
			// also trips the missing-required-field check at "end".
			name: "weight missing unit",
			src: "shipment A\n" +
				"weight 1\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{2, 1, `"weight" requires a number and a unit`},
				{1, 1, `shipment "A" is missing required field "weight"`},
			},
		},
		{
			name: "weight invalid number",
			src: "shipment A\n" +
				"weight abc lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{2, 8, `invalid number "abc"`},
				{1, 1, `shipment "A" is missing required field "weight"`},
			},
		},
		{
			name: "weight not positive",
			src: "shipment A\n" +
				"weight 0 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{2, 8, `weight must be greater than zero`},
				{1, 1, `shipment "A" is missing required field "weight"`},
			},
		},
		{
			name: "weight invalid unit",
			src: "shipment A\n" +
				"weight 1 lbs\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{2, 10, `invalid unit "lbs" (expected lb or kg)`},
				{1, 1, `shipment "A" is missing required field "weight"`},
			},
		},
		{
			name: "dims missing unit",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{3, 1, `"dims" requires LxWxH and a unit`},
				{1, 1, `shipment "A" is missing required field "dims"`},
			},
		},
		{
			name: "dims wrong shape",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{3, 6, `invalid dimensions "1x1" (expected LxWxH`},
				{1, 1, `shipment "A" is missing required field "dims"`},
			},
		},
		{
			name: "dims non-positive component",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x0x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{3, 8, `invalid dimension "0" in "1x0x1"`},
				{1, 1, `shipment "A" is missing required field "dims"`},
			},
		},
		{
			name: "dims invalid unit",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 xx\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{3, 12, `invalid unit "xx" (expected in or cm)`},
				{1, 1, `shipment "A" is missing required field "dims"`},
			},
		},
		{
			name: "duplicate dims field",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"dims 2x2x2 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{{4, 1, `duplicate "dims" field`}},
		},
		{
			name: "duplicate service field",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"service y\n" +
				"end\n",
			want: []wantErr{{5, 1, `duplicate "service" field`}},
		},
		{
			name: "service missing value",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service\n" +
				"end\n",
			want: []wantErr{
				{4, 1, `"service" requires exactly one value`},
				{1, 1, `shipment "A" is missing required field "service"`},
			},
		},
		{
			name: "unknown field",
			src: "shipment A\n" +
				"foo 1\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{{2, 1, `unknown field "foo"`}},
		},
		{
			name: "trailing token after end",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end extra\n",
			want: []wantErr{{5, 5, `unexpected "extra" after "end"`}},
		},
		{
			name: "missing required field",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{{1, 1, `shipment "A" is missing required field "dims"`}},
		},
		{
			name: "mixed unit systems",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 cm\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{{1, 1, `shipment "A" mixes unit systems (weight in lb, dims in cm)`}},
		},
		{
			name: "unterminated shipment",
			src: "shipment A\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n",
			want: []wantErr{{1, 1, `shipment "A" is missing "end"`}},
		},
		{
			name: "multiple independent errors reported together",
			src: "shipment A\n" +
				"foo 1\n" +
				"weight 1 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n" +
				"shipment B\n" +
				"weight 1 lb\n" +
				"weight 2 lb\n" +
				"dims 1x1x1 in\n" +
				"service x\n" +
				"end\n",
			want: []wantErr{
				{2, 1, `unknown field "foo"`},
				{9, 1, `duplicate "weight" field`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest("test.manifest", tt.src)
			checkErrs(t, err, tt.want)
		})
	}
}

func TestParseManifestSuccess(t *testing.T) {
	src := "shipment SH1001\n" +
		"  weight 12.5 lb\n" +
		"  dims 10x8x6 in\n" +
		"  service ground\n" +
		"end\n" +
		"\n" +
		"shipment SH1002\n" +
		"  weight 2 kg\n" +
		"  dims 40x30x20 cm\n" +
		"  service express\n" +
		"end\n"

	shipments, err := ParseManifest("test.manifest", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shipments) != 2 {
		t.Fatalf("got %d shipments, want 2", len(shipments))
	}

	s1 := shipments[0]
	if s1.ID != "SH1001" || s1.Weight != 12.5 || s1.WeightUnit != "lb" ||
		s1.Dims != (Dims{10, 8, 6}) || s1.DimsUnit != "in" || s1.Service != "ground" {
		t.Errorf("unexpected shipment 1: %+v", s1)
	}
	if s1.Pos != (Pos{Line: 1, Col: 1}) {
		t.Errorf("shipment 1 pos: got %+v, want {1 1}", s1.Pos)
	}

	s2 := shipments[1]
	if s2.ID != "SH1002" || s2.Weight != 2 || s2.WeightUnit != "kg" ||
		s2.Dims != (Dims{40, 30, 20}) || s2.DimsUnit != "cm" || s2.Service != "express" {
		t.Errorf("unexpected shipment 2: %+v", s2)
	}
	if s2.Pos != (Pos{Line: 7, Col: 1}) {
		t.Errorf("shipment 2 pos: got %+v, want {7 1}", s2.Pos)
	}
}
