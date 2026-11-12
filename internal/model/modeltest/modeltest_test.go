package modeltest_test

import (
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/friendly-fhir/go-ucum/internal/model"
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

func TestEquateApprox(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		digits int
		lhs    *big.Rat
		rhs    *big.Rat
		want   bool
	}{
		{
			name:   "identical",
			digits: 3,
			lhs:    big.NewRat(1, 3),
			rhs:    big.NewRat(1, 3),
			want:   true,
		}, {
			name:   "within digits",
			digits: 3,
			lhs:    big.NewRat(1, 3),
			rhs:    big.NewRat(3333, 10000),
			want:   true,
		}, {
			name:   "beyond digits",
			digits: 6,
			lhs:    big.NewRat(1, 3),
			rhs:    big.NewRat(3333, 10000),
			want:   false,
		}, {
			name:   "larger right operand",
			digits: 3,
			lhs:    big.NewRat(9999, 10000),
			rhs:    big.NewRat(1, 1),
			want:   true,
		}, {
			name:   "both zero",
			digits: 3,
			lhs:    new(big.Rat),
			rhs:    new(big.Rat),
			want:   true,
		}, {
			name:   "both nil",
			digits: 3,
			lhs:    nil,
			rhs:    nil,
			want:   true,
		}, {
			name:   "one nil",
			digits: 3,
			lhs:    big.NewRat(1, 1),
			rhs:    nil,
			want:   false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opt := modeltest.EquateApprox(tc.digits)

			// Act
			equal := cmp.Equal(tc.lhs, tc.rhs, opt)

			// Assert
			if got, want := equal, tc.want; !cmp.Equal(got, want) {
				t.Errorf("cmp.Equal(%v, %v, EquateApprox(%d)) = %v, want %v", tc.lhs, tc.rhs, tc.digits, got, want)
			}
		})
	}
}

func TestEquateCanonicals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lhs  model.Canonical
		rhs  model.Canonical
		want bool
	}{
		{
			name: "equal factors in different terms",
			lhs:  model.Canonical{Factor: big.NewRat(2, 4)},
			rhs:  model.Canonical{Factor: big.NewRat(1, 2)},
			want: true,
		}, {
			name: "absent and empty arbitrary units",
			lhs:  model.Canonical{Factor: big.NewRat(1, 1), Arbitrary: map[string]int{}},
			rhs:  model.Canonical{Factor: big.NewRat(1, 1)},
			want: true,
		}, {
			name: "different arbitrary units",
			lhs:  model.Canonical{Factor: big.NewRat(1, 1), Arbitrary: map[string]int{"[iU]": 1}},
			rhs:  model.Canonical{Factor: big.NewRat(1, 1)},
			want: false,
		}, {
			name: "special units with the same function name",
			lhs:  model.Canonical{Special: &model.Special{Name: "Cel", Scale: big.NewRat(1, 1)}},
			rhs:  model.Canonical{Special: &model.Special{Name: "Cel", Scale: big.NewRat(1, 1)}},
			want: true,
		}, {
			name: "special units with different function names",
			lhs:  model.Canonical{Special: &model.Special{Name: "Cel"}},
			rhs:  model.Canonical{Special: &model.Special{Name: "degF"}},
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opt := modeltest.EquateCanonicals()

			// Act
			equal := cmp.Equal(tc.lhs, tc.rhs, opt)

			// Assert
			if got, want := equal, tc.want; !cmp.Equal(got, want) {
				t.Errorf("cmp.Equal(%v, %v, EquateCanonicals()) = %v, want %v", tc.lhs, tc.rhs, got, want)
			}
		})
	}
}
