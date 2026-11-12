package ucum_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum"
)

func TestWithPrecision(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		digits int32
		want   decimal.Decimal
	}{
		{
			name:   "few digits",
			digits: 5,
			want:   decimal.RequireFromString("39.370"),
		}, {
			name:   "single digit",
			digits: 1,
			want:   decimal.RequireFromString("40"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t, ucum.WithPrecision(tc.digits))
			value := decimal.RequireFromString("1")
			from := mustParseUnit(t, sut, "m")
			to := mustParseUnit(t, sut, "[in_i]")

			// Act
			result, err := sut.Convert(value, from, to)

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Engine.Convert() error = %v, want %v", got, want)
			}
			if got, want := result, tc.want; !cmp.Equal(got, want, equateDecimals) {
				t.Errorf("Engine.Convert() = %v, want %v", got, want)
			}
		})
	}
}

func TestWithMaxExponent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		code    string
		want    *unitView
		wantErr error
	}{
		{
			name: "at limit",
			code: "m-2",
			want: &unitView{Code: "m-2", Dimension: "m-2"},
		}, {
			name:    "beyond limit",
			code:    "m3",
			wantErr: ucum.ErrInvalidSyntax,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t, ucum.WithMaxExponent(2))

			// Act
			unit, err := sut.ParseUnit(tc.code)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Engine.ParseUnit(%q) error = %v, want %v", tc.code, got, want)
			}
			if got, want := view(unit), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Engine.ParseUnit(%q) = %+v, want %+v", tc.code, got, want)
			}
		})
	}
}

func TestWithMaxMagnitude(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		want    decimal.Decimal
		wantErr error
	}{
		{
			name:  "at limit",
			value: "1000",
			want:  decimal.RequireFromString("1000"),
		}, {
			name:    "beyond limit",
			value:   "10000",
			wantErr: ucum.ErrDomain,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t, ucum.WithMaxMagnitude(3))
			value := decimal.RequireFromString(tc.value)
			unit := mustParseUnit(t, sut, "m")

			// Act
			result, err := sut.Convert(value, unit, unit)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Engine.Convert(%v) error = %v, want %v", value, got, want)
			}
			if got, want := result, tc.want; !cmp.Equal(got, want, equateDecimals) {
				t.Errorf("Engine.Convert(%v) = %v, want %v", value, got, want)
			}
		})
	}
}
