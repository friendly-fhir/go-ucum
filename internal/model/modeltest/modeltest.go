package modeltest

import (
	"math/big"

	"github.com/google/go-cmp/cmp"
)

// EquateRats returns a [cmp.Option] that treats two [big.Rat] values as equal
// when they denote the same number, regardless of how they are reduced.
func EquateRats() cmp.Option {
	return cmp.Comparer(func(lhs, rhs *big.Rat) bool {
		if lhs == nil || rhs == nil {
			return lhs == rhs
		}
		return lhs.Cmp(rhs) == 0
	})
}
