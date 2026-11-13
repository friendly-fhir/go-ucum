package ucum_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
