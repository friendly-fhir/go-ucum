package ucum_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum"
)

func TestUnitCompatible(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		unit  string
		other string
		want  bool
	}{
		{
			name:  "same dimension",
			unit:  "m",
			other: "[in_i]",
			want:  true,
		}, {
			name:  "special and proper unit",
			unit:  "Cel",
			other: "K",
			want:  true,
		}, {
			name:  "different dimensions",
			unit:  "m",
			other: "s",
			want:  false,
		}, {
			name:  "different arbitrary units",
			unit:  "[iU]",
			other: "[arb'U]",
			want:  false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			engine := newEngine(t)
			sut := mustParseUnit(t, engine, tc.unit)
			other := mustParseUnit(t, engine, tc.other)

			// Act
			compatible := sut.Compatible(other)

			// Assert
			if got, want := compatible, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Unit(%q).Compatible(%q) = %v, want %v", tc.unit, tc.other, got, want)
			}
		})
	}
}

func TestUnitCompatible_ForeignUnit_ReportsFalse(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := mustParseUnit(t, newEngine(t), "m")
	other := mustParseUnit(t, newEngine(t), "m")

	// Act
	compatible := sut.Compatible(other)

	// Assert
	if got, want := compatible, false; !cmp.Equal(got, want) {
		t.Errorf("Unit(%v).Compatible(%v) = %v, want %v", sut, other, got, want)
	}
}

func TestUnitConvert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		unit    string
		to      string
		want    decimal.Decimal
		wantErr error
	}{
		{
			name:  "proportional units",
			value: "5",
			unit:  "cm",
			to:    "mm",
			want:  decimal.RequireFromString("50"),
		}, {
			name:  "special unit",
			value: "37",
			unit:  "Cel",
			to:    "[degF]",
			want:  decimal.RequireFromString("98.6"),
		}, {
			name:    "different dimensions",
			value:   "1",
			unit:    "m",
			to:      "s",
			wantErr: ucum.ErrIncompatible,
		}, {
			name:    "outside special domain",
			value:   "0",
			unit:    "mol/L",
			to:      "[pH]",
			wantErr: ucum.ErrDomain,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			engine := newEngine(t)
			value := decimal.RequireFromString(tc.value)
			sut := mustParseUnit(t, engine, tc.unit)
			to := mustParseUnit(t, engine, tc.to)

			// Act
			result, err := sut.Convert(value, to)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Unit(%q).Convert(%v, %q) error = %v, want %v", tc.unit, value, tc.to, got, want)
			}
			if got, want := result, tc.want; !cmp.Equal(got, want, equateDecimals) {
				t.Errorf("Unit(%q).Convert(%v, %q) = %v, want %v", tc.unit, value, tc.to, got, want)
			}
		})
	}
}

func TestUnitConvert_ForeignUnit_ReturnsError(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := mustParseUnit(t, newEngine(t), "m")
	to := mustParseUnit(t, newEngine(t), "m")
	value := decimal.RequireFromString("1")

	// Act
	result, err := sut.Convert(value, to)

	// Assert
	if got, want := err, ucum.ErrForeignUnit; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Unit(%v).Convert(%v, %v) error = %v, want %v", sut, value, to, got, want)
	}
	if got, want := result, (decimal.Decimal{}); !cmp.Equal(got, want, equateDecimals) {
		t.Errorf("Unit(%v).Convert(%v, %v) = %v, want %v", sut, value, to, got, want)
	}
}

func TestUnitCoarser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		unit  string
		other string
		want  bool
	}{
		{
			name:  "larger prefix",
			unit:  "cm",
			other: "mm",
			want:  true,
		}, {
			name:  "smaller prefix",
			unit:  "mm",
			other: "cm",
			want:  false,
		}, {
			name:  "different systems",
			unit:  "[in_i]",
			other: "m",
			want:  false,
		}, {
			name:  "same unit",
			unit:  "m",
			other: "m",
			want:  false,
		}, {
			name:  "equal magnitudes",
			unit:  "mL",
			other: "cm3",
			want:  false,
		}, {
			name:  "same arbitrary unit",
			unit:  "[iU]/mL",
			other: "[IU]/L",
			want:  true,
		}, {
			name:  "different dimensions",
			unit:  "km",
			other: "s",
			want:  false,
		}, {
			name:  "different arbitrary units",
			unit:  "k[iU]",
			other: "[arb'U]",
			want:  false,
		}, {
			name:  "special unit",
			unit:  "Cel",
			other: "[degF]",
			want:  false,
		}, {
			name:  "special and proper unit",
			unit:  "kK",
			other: "Cel",
			want:  false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			engine := newEngine(t)
			sut := mustParseUnit(t, engine, tc.unit)
			other := mustParseUnit(t, engine, tc.other)

			// Act
			coarser := sut.Coarser(other)

			// Assert
			if got, want := coarser, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Unit(%q).Coarser(%q) = %v, want %v", tc.unit, tc.other, got, want)
			}
		})
	}
}

func TestUnitCoarser_ForeignUnit_ReportsFalse(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := mustParseUnit(t, newEngine(t), "km")
	other := mustParseUnit(t, newEngine(t), "m")

	// Act
	coarser := sut.Coarser(other)

	// Assert
	if got, want := coarser, false; !cmp.Equal(got, want) {
		t.Errorf("Unit(%v).Coarser(%v) = %v, want %v", sut, other, got, want)
	}
}
