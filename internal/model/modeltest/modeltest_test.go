package modeltest_test

import (
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/friendly-fhir/go-ucum/internal/model/modeltest"
)

func TestEquateRats(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lhs  *big.Rat
		rhs  *big.Rat
		want bool
	}{
		{
			name: "same value in different terms",
			lhs:  new(big.Rat).SetFrac64(2, 4),
			rhs:  big.NewRat(1, 2),
			want: true,
		}, {
			name: "different values",
			lhs:  big.NewRat(1, 2),
			rhs:  big.NewRat(1, 3),
			want: false,
		}, {
			name: "both nil",
			lhs:  nil,
			rhs:  nil,
			want: true,
		}, {
			name: "one nil",
			lhs:  big.NewRat(0, 1),
			rhs:  nil,
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opt := modeltest.EquateRats()

			// Act
			equal := cmp.Equal(tc.lhs, tc.rhs, opt)

			// Assert
			if got, want := equal, tc.want; !cmp.Equal(got, want) {
				t.Errorf("cmp.Equal(%v, %v, EquateRats()) = %v, want %v", tc.lhs, tc.rhs, got, want)
			}
		})
	}
}
