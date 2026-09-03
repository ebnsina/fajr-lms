package payment

import "fmt"

// Part is what one instalment of a plan costs. The parts add up to the total
// exactly: the remainder rides on the last one rather than being lost or
// quietly rounded up on every part.
func Part(totalMinor int64, parts, partNo int) (int64, error) {
	if totalMinor <= 0 {
		return 0, fmt.Errorf("payment: a plan needs an amount")
	}
	if parts < 2 || parts > 24 {
		return 0, fmt.Errorf("payment: a plan runs between 2 and 24 parts")
	}
	if partNo < 1 || partNo > parts {
		return 0, fmt.Errorf("payment: part %d is not in a plan of %d", partNo, parts)
	}
	each := totalMinor / int64(parts)
	if each < 1 {
		return 0, fmt.Errorf("payment: this is too small to split %d ways", parts)
	}
	if partNo == parts {
		return totalMinor - each*int64(parts-1), nil
	}
	return each, nil
}
