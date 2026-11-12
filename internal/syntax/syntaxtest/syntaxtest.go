package syntaxtest

import (
	"errors"
	"math/big"

	"github.com/google/go-cmp/cmp"

	"github.com/friendly-fhir/go-ucum/internal/syntax"
)

// EquateNodes returns a [cmp.Option] that compares parsed expressions by
// meaning, treating [syntax.Factor] values as equal when they hold the same
// number.
func EquateNodes() cmp.Option {
	return cmp.Comparer(func(lhs, rhs *big.Int) bool {
		if lhs == nil || rhs == nil {
			return lhs == rhs
		}
		return lhs.Cmp(rhs) == 0
	})
}

// EquateErrors returns a [cmp.Option] that treats two [*syntax.Error] values as
// equal when they report the same offset and either wraps the cause of the
// other, so that an expectation can name only the sentinel it requires.
func EquateErrors() cmp.Option {
	return cmp.Comparer(func(lhs, rhs *syntax.Error) bool {
		if lhs == nil || rhs == nil {
			return lhs == rhs
		}
		return lhs.Offset == rhs.Offset && (errors.Is(lhs.Err, rhs.Err) || errors.Is(rhs.Err, lhs.Err))
	})
}
