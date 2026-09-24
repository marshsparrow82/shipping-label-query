package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: labelq <manifest-file>")
		os.Exit(2)
	}
	path := os.Args[1]

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "labelq: %v\n", err)
		os.Exit(1)
	}

	shipments, err := ParseManifest(path, string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(shipments) == 0 {
		fmt.Fprintf(os.Stderr, "labelq: no shipments found in %s\n", path)
		os.Exit(1)
	}

	fmt.Printf("%-10s %10s %10s %10s  %s\n", "ID", "ACTUAL", "DIM", "BILLABLE", "NOTE")
	for _, s := range shipments {
		r := ComputeBillable(s)
		note := ""
		if r.DimBased {
			note = "dimensional weight applies"
		}
		fmt.Printf("%-10s %8.0f%-2s %8.0f%-2s %8.0f%-2s  %s\n",
			s.ID,
			r.Actual, s.WeightUnit,
			r.Dimensional, s.WeightUnit,
			r.Billable, s.WeightUnit,
			note)
	}
}
