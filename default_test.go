package ucum_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum"
	"github.com/friendly-fhir/go-ucum/internal/model/modeltest"
)

func TestDefaultRegistry_GeneratedFromCommittedDocument_MatchesDocument(t *testing.T) {
	t.Parallel()

	// Arrange
	want := loadRegistry(t, "internal/data/ucum-essence.xml")

	// Act
	registry := ucum.DefaultRegistry

	// Assert
	opts := cmp.Options{cmp.AllowUnexported(ucum.Registry{}), modeltest.EquateRats()}
	if got, want := registry, want; !cmp.Equal(got, want, opts) {
		t.Errorf("DefaultRegistry mismatch (-want +got):\n%s", cmp.Diff(want, got, opts))
	}
}

func TestDefaultEngineConvert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
		from  string
		to    string
		want  decimal.Decimal
	}{
		{
			name:  "mass concentration",
			value: "100",
			from:  "mg/dL",
			to:    "g/L",
			want:  decimal.RequireFromString("1"),
		}, {
			name:  "blood pressure",
			value: "760",
			from:  "mm[Hg]",
			to:    "kPa",
			want:  decimal.RequireFromString("101.32472"),
		}, {
			name:  "avoirdupois pound",
			value: "1",
			from:  "[lb_av]",
			to:    "kg",
			want:  decimal.RequireFromString("0.45359237"),
		}, {
			name:  "body temperature",
			value: "98.6",
			from:  "[degF]",
			to:    "Cel",
			want:  decimal.RequireFromString("37"),
		}, {
			name:  "liter synonyms",
			value: "1",
			from:  "L",
			to:    "mL",
			want:  decimal.RequireFromString("1000"),
		}, {
			name:  "unit added in release 2.2",
			value: "4",
			from:  "[NTU]",
			to:    "1",
			want:  decimal.RequireFromString("4"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := ucum.DefaultEngine
			value := decimal.RequireFromString(tc.value)
			from := mustParseUnit(t, sut, tc.from)
			to := mustParseUnit(t, sut, tc.to)

			// Act
			result, err := sut.Convert(value, from, to)

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DefaultEngine.Convert(%v, %q, %q) error = %v, want %v", value, tc.from, tc.to, got, want)
			}
			if got, want := result, tc.want; !cmp.Equal(got, want, equateDecimals) {
				t.Errorf("DefaultEngine.Convert(%v, %q, %q) = %v, want %v", value, tc.from, tc.to, got, want)
			}
		})
	}
}
