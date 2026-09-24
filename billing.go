package main

import "math"

// These are the standard domestic air-carrier dimensional-weight divisors:
// a 1-cubic-foot (1728 in^3) box divides down to ~12.4 lb under the
// imperial divisor, which is roughly what UPS/FedEx/USPS use for ground and
// air services absent a carrier-specific contract.
const (
	imperialDivisor = 139.0  // in^3 per lb
	metricDivisor   = 5000.0 // cm^3 per kg
)

// WeightResult is the answer to the one question this tool exists to
// answer: what does a carrier actually bill this package as.
type WeightResult struct {
	Actual      float64
	Dimensional float64
	Billable    float64
	DimBased    bool // true if dimensional weight, not actual weight, set the price
}

// ComputeBillable returns the billable weight for a shipment: the greater
// of its actual weight and its dimensional (volumetric) weight, each
// rounded up to the nearest whole unit the way carriers bill it.
func ComputeBillable(s Shipment) WeightResult {
	volume := s.Dims.L * s.Dims.W * s.Dims.H

	var dimWeight float64
	if s.DimsUnit == "in" {
		dimWeight = volume / imperialDivisor
	} else {
		dimWeight = volume / metricDivisor
	}
	dimWeight = math.Ceil(dimWeight)
	actual := math.Ceil(s.Weight)

	billable := actual
	dimBased := false
	if dimWeight > actual {
		billable = dimWeight
		dimBased = true
	}

	return WeightResult{
		Actual:      actual,
		Dimensional: dimWeight,
		Billable:    billable,
		DimBased:    dimBased,
	}
}
