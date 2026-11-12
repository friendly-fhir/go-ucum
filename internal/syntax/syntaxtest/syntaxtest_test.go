package syntaxtest_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/friendly-fhir/go-ucum/internal/syntax"
	"github.com/friendly-fhir/go-ucum/internal/syntax/syntaxtest"
)

func TestEquateNodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lhs  syntax.Node
		rhs  syntax.Node
		want bool
	}{
		{
			name: "equal factors",
			lhs:  syntax.Factor{Value: big.NewInt(24)},
			rhs:  syntax.Factor{Value: new(big.Int).SetUint64(24)},
			want: true,
		}, {
			name: "different factors",
			lhs:  syntax.Factor{Value: big.NewInt(24)},
			rhs:  syntax.Factor{Value: big.NewInt(60)},
			want: false,
		}, {
			name: "missing factor value",
			lhs:  syntax.Factor{Value: big.NewInt(24)},
			rhs:  syntax.Factor{},
			want: false,
		}, {
			name: "both factor values missing",
			lhs:  syntax.Factor{},
			rhs:  syntax.Factor{},
			want: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opt := syntaxtest.EquateNodes()

			// Act
			equal := cmp.Equal(tc.lhs, tc.rhs, opt)

			// Assert
			if got, want := equal, tc.want; !cmp.Equal(got, want) {
				t.Errorf("cmp.Equal(%v, %v, EquateNodes()) = %v, want %v", tc.lhs, tc.rhs, got, want)
			}
		})
	}
}

func TestEquateErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lhs  error
		rhs  error
		want bool
	}{
		{
			name: "wrapped cause at same offset",
			lhs:  &syntax.Error{Offset: 3, Err: fmt.Errorf("%w: detail", syntax.ErrUnknownUnit)},
			rhs:  &syntax.Error{Offset: 3, Err: syntax.ErrUnknownUnit},
			want: true,
		}, {
			name: "expectation first",
			lhs:  &syntax.Error{Offset: 3, Err: syntax.ErrUnknownUnit},
			rhs:  &syntax.Error{Offset: 3, Err: fmt.Errorf("%w: detail", syntax.ErrUnknownUnit)},
			want: true,
		}, {
			name: "different offset",
			lhs:  &syntax.Error{Offset: 3, Err: syntax.ErrUnknownUnit},
			rhs:  &syntax.Error{Offset: 4, Err: syntax.ErrUnknownUnit},
			want: false,
		}, {
			name: "different cause",
			lhs:  &syntax.Error{Offset: 3, Err: syntax.ErrUnknownUnit},
			rhs:  &syntax.Error{Offset: 3, Err: syntax.ErrInvalidSyntax},
			want: false,
		}, {
			name: "missing error",
			lhs:  &syntax.Error{Offset: 3, Err: syntax.ErrUnknownUnit},
			rhs:  (*syntax.Error)(nil),
			want: false,
		}, {
			name: "both missing",
			lhs:  (*syntax.Error)(nil),
			rhs:  (*syntax.Error)(nil),
			want: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			opt := syntaxtest.EquateErrors()

			// Act
			equal := cmp.Equal(tc.lhs, tc.rhs, opt)

			// Assert
			if got, want := equal, tc.want; !cmp.Equal(got, want) {
				t.Errorf("cmp.Equal(%v, %v, EquateErrors()) = %v, want %v", tc.lhs, tc.rhs, got, want)
			}
		})
	}
}
