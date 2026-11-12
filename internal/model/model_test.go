package model_test

import (
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/model/modeltest"
)

func TestMustRat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  *big.Rat
	}{
		{
			name:  "fraction",
			input: "254/100",
			want:  big.NewRat(127, 50),
		}, {
			name:  "scientific notation",
			input: "980665e-5",
			want:  big.NewRat(196133, 20000),
		}, {
			name:  "decimal",
			input: "2.54",
			want:  big.NewRat(127, 50),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			rat := model.MustRat(tc.input)

			// Assert
			if got, want := rat, tc.want; !cmp.Equal(got, want, modeltest.EquateRats()) {
				t.Errorf("MustRat(%q) = %v, want %v", tc.input, got, want)
			}
		})
	}
}

func TestMustRat_InvalidInput_Panics(t *testing.T) {
	t.Parallel()

	// Arrange
	input := "not a number"

	// Act
	panicked := panics(func() { model.MustRat(input) })

	// Assert
	if got, want := panicked, true; !cmp.Equal(got, want) {
		t.Errorf("MustRat(%q) panicked = %v, want %v", input, got, want)
	}
}

func panics(fn func()) (panicked bool) {
	defer func() {
		panicked = recover() != nil
	}()
	fn()
	return false
}
