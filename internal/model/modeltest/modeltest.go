package modeltest

import (
	"math/big"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/friendly-fhir/go-ucum/internal/model"
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

// EquateApprox returns a [cmp.Option] that treats two [big.Rat] values as equal
// when they agree to the given number of significant digits.
func EquateApprox(digits int) cmp.Option {
	tolerance := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil))
	return cmp.Comparer(func(lhs, rhs *big.Rat) bool {
		if lhs == nil || rhs == nil {
			return lhs == rhs
		}
		difference := new(big.Rat).Sub(lhs, rhs)
		difference.Abs(difference)
		scale := new(big.Rat).Abs(lhs)
		if other := new(big.Rat).Abs(rhs); other.Cmp(scale) > 0 {
			scale = other
		}
		return difference.Cmp(scale.Mul(scale, tolerance)) <= 0
	})
}

// EquateCanonicals returns a [cmp.Option] that compares [model.Canonical]
// values by meaning: magnitudes by value, an absent and an empty set of
// arbitrary units alike, and special units by the name of their conversion.
func EquateCanonicals() cmp.Option {
	return cmp.Options{
		EquateRats(),
		cmpopts.EquateEmpty(),
		cmpopts.IgnoreFields(model.Special{}, "Conversion"),
	}
}
