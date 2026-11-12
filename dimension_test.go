package ucum_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDimensionMultiply(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lhs  string
		rhs  string
		want string
	}{
		{
			name: "velocity by time",
			lhs:  "m/s",
			rhs:  "s",
			want: "m",
		}, {
			name: "by dimensionless",
			lhs:  "g",
			rhs:  "%",
			want: "g",
		}, {
			name: "by inverse",
			lhs:  "m2",
			rhs:  "/m2",
			want: "1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			engine := newEngine(t)
			sut := mustParseUnit(t, engine, tc.lhs).Dimension()
			rhs := mustParseUnit(t, engine, tc.rhs).Dimension()
			want := mustParseUnit(t, engine, tc.want).Dimension()

			// Act
			product := sut.Multiply(rhs)

			// Assert
			if got, want := product, want; !cmp.Equal(got, want, cmp.AllowUnexported(want)) {
				t.Errorf("Dimension.Multiply(%v) = %v, want %v", rhs, got, want)
			}
		})
	}
}

func TestDimensionDivide(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lhs  string
		rhs  string
		want string
	}{
		{
			name: "length by time",
			lhs:  "m",
			rhs:  "s",
			want: "m/s",
		}, {
			name: "by itself",
			lhs:  "g/L",
			rhs:  "mg/dL",
			want: "1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			engine := newEngine(t)
			sut := mustParseUnit(t, engine, tc.lhs).Dimension()
			rhs := mustParseUnit(t, engine, tc.rhs).Dimension()
			want := mustParseUnit(t, engine, tc.want).Dimension()

			// Act
			quotient := sut.Divide(rhs)

			// Assert
			if got, want := quotient, want; !cmp.Equal(got, want, cmp.AllowUnexported(want)) {
				t.Errorf("Dimension.Divide(%v) = %v, want %v", rhs, got, want)
			}
		})
	}
}

func TestDimensionPow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		unit     string
		exponent int
		want     string
	}{
		{
			name:     "square",
			unit:     "m/s",
			exponent: 2,
			want:     "m2/s2",
		}, {
			name:     "inverse",
			unit:     "m/s",
			exponent: -1,
			want:     "s/m",
		}, {
			name:     "zeroth power",
			unit:     "g.m",
			exponent: 0,
			want:     "1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			engine := newEngine(t)
			sut := mustParseUnit(t, engine, tc.unit).Dimension()
			want := mustParseUnit(t, engine, tc.want).Dimension()

			// Act
			power := sut.Pow(tc.exponent)

			// Assert
			if got, want := power, want; !cmp.Equal(got, want, cmp.AllowUnexported(want)) {
				t.Errorf("Dimension.Pow(%d) = %v, want %v", tc.exponent, got, want)
			}
		})
	}
}

func TestDimensionIsDimensionless(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		unit string
		want bool
	}{
		{
			name: "count",
			unit: "mol",
			want: true,
		}, {
			name: "cancelled ratio",
			unit: "mg/kg",
			want: true,
		}, {
			name: "length",
			unit: "m",
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustParseUnit(t, newEngine(t), tc.unit).Dimension()

			// Act
			dimensionless := sut.IsDimensionless()

			// Assert
			if got, want := dimensionless, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Dimension.IsDimensionless() = %v, want %v", got, want)
			}
		})
	}
}

func TestDimensionString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		unit string
		want string
	}{
		{
			name: "dimensionless",
			unit: "%",
			want: "1",
		}, {
			name: "single base unit",
			unit: "[in_i]",
			want: "m",
		}, {
			name: "base unit order",
			unit: "g/(m.s2)",
			want: "m-1.s-2.g",
		}, {
			name: "remaining base units",
			unit: "rad.C2.cd/K",
			want: "rad.K-1.C2.cd",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustParseUnit(t, newEngine(t), tc.unit).Dimension()

			// Act
			text := sut.String()

			// Assert
			if got, want := text, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Dimension.String() = %q, want %q", got, want)
			}
		})
	}
}
