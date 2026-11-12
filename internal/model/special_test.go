package model_test

import (
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/model/modeltest"
)

var limits = model.Limits{
	Digits:       34,
	MaxMagnitude: 1000,
}

// Digits of agreement expected from each kind of result: exact results agree
// fully, logarithms to the requested precision, and tangents to the fixed
// precision of the decimal trigonometric functions.
const (
	exactDigits   = 1000
	logDigits     = 34
	tangentDigits = 15
)

func TestConversionToProper(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		function  string
		input     *big.Rat
		digits    int
		want      *big.Rat
		wantExact bool
		wantErr   error
	}{
		{
			name:      "body temperature in kelvin",
			function:  "Cel",
			input:     model.MustRat("37"),
			digits:    exactDigits,
			want:      model.MustRat("310.15"),
			wantExact: true,
		}, {
			name:      "fahrenheit on the rankine scale",
			function:  "degF",
			input:     model.MustRat("98.6"),
			digits:    exactDigits,
			want:      model.MustRat("558.27"),
			wantExact: true,
		}, {
			name:      "reaumur freezing point",
			function:  "degRe",
			input:     model.MustRat("0"),
			digits:    exactDigits,
			want:      model.MustRat("218.52"),
			wantExact: true,
		}, {
			name:     "prism diopter in radians",
			function: "tanTimes100",
			input:    model.MustRat("100"),
			digits:   tangentDigits,
			want:     model.MustRat("0.78539816339744830961566084581988"),
		}, {
			name:     "flat slope",
			function: "100tan",
			input:    model.MustRat("0"),
			digits:   tangentDigits,
			want:     model.MustRat("0"),
		}, {
			name:     "slope percent in degrees",
			function: "100tan",
			input:    model.MustRat("100"),
			digits:   tangentDigits,
			want:     model.MustRat("45"),
		}, {
			name:     "neutral pH",
			function: "pH",
			input:    model.MustRat("7"),
			digits:   logDigits,
			want:     model.MustRat("1e-7"),
		}, {
			name:     "decimal potency",
			function: "hpX",
			input:    model.MustRat("2"),
			digits:   logDigits,
			want:     model.MustRat("0.01"),
		}, {
			name:     "centesimal potency",
			function: "hpC",
			input:    model.MustRat("1"),
			digits:   logDigits,
			want:     model.MustRat("0.01"),
		}, {
			name:     "millesimal potency",
			function: "hpM",
			input:    model.MustRat("1"),
			digits:   logDigits,
			want:     model.MustRat("0.001"),
		}, {
			name:     "quinquagintamillesimal potency",
			function: "hpQ",
			input:    model.MustRat("1"),
			digits:   logDigits,
			want:     model.MustRat("0.00002"),
		}, {
			name:     "one neper",
			function: "ln",
			input:    model.MustRat("1"),
			digits:   logDigits,
			want:     model.MustRat("2.718281828459045235360287471352662497757"),
		}, {
			name:     "two bel",
			function: "lg",
			input:    model.MustRat("2"),
			digits:   logDigits,
			want:     model.MustRat("100"),
		}, {
			name:     "two bel of a field quantity",
			function: "lgTimes2",
			input:    model.MustRat("2"),
			digits:   logDigits,
			want:     model.MustRat("10"),
		}, {
			name:     "three bits",
			function: "ld",
			input:    model.MustRat("3"),
			digits:   logDigits,
			want:     model.MustRat("8"),
		}, {
			name:      "square of root",
			function:  "sqrt",
			input:     model.MustRat("3/2"),
			digits:    exactDigits,
			want:      model.MustRat("9/4"),
			wantExact: true,
		}, {
			name:     "magnitude above limit",
			function: "lg",
			input:    model.MustRat("1001"),
			wantErr:  model.ErrDomain,
		}, {
			name:     "magnitude below limit",
			function: "pH",
			input:    model.MustRat("1001"),
			wantErr:  model.ErrDomain,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustLookupConversion(t, tc.function)

			// Act
			proper, exact, err := sut.ToProper(tc.input, limits)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Conversion.ToProper(%v) error = %v, want %v", tc.input, got, want)
			}
			if got, want := proper, tc.want; !cmp.Equal(got, want, modeltest.EquateApprox(tc.digits)) {
				t.Errorf("Conversion.ToProper(%v) = %v, want %v", tc.input, got.FloatString(40), want.FloatString(40))
			}
			if got, want := exact, tc.wantExact; !cmp.Equal(got, want) {
				t.Errorf("Conversion.ToProper(%v) exact = %v, want %v", tc.input, got, want)
			}
		})
	}
}

func TestConversionFromProper(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		function  string
		input     *big.Rat
		digits    int
		want      *big.Rat
		wantExact bool
		wantErr   error
	}{
		{
			name:      "absolute zero in celsius",
			function:  "Cel",
			input:     model.MustRat("0"),
			digits:    exactDigits,
			want:      model.MustRat("-273.15"),
			wantExact: true,
		}, {
			name:      "absolute zero in fahrenheit",
			function:  "degF",
			input:     model.MustRat("0"),
			digits:    exactDigits,
			want:      model.MustRat("-459.67"),
			wantExact: true,
		}, {
			name:      "reaumur from proper",
			function:  "degRe",
			input:     model.MustRat("218.52"),
			digits:    exactDigits,
			want:      model.MustRat("0"),
			wantExact: true,
		}, {
			name:     "quarter turn of pi radians",
			function: "tanTimes100",
			input:    model.MustRat("0.78539816339744830961566084581988"),
			digits:   tangentDigits,
			want:     model.MustRat("100"),
		}, {
			name:     "forty-five degrees",
			function: "100tan",
			input:    model.MustRat("45"),
			digits:   tangentDigits,
			want:     model.MustRat("100"),
		}, {
			name:     "neutral concentration",
			function: "pH",
			input:    model.MustRat("1e-7"),
			digits:   logDigits,
			want:     model.MustRat("7"),
		}, {
			name:     "natural logarithm of ten",
			function: "ln",
			input:    model.MustRat("10"),
			digits:   logDigits,
			want:     model.MustRat("2.302585092994045684017991454684364207601"),
		}, {
			name:     "power ratio of one thousand",
			function: "lg",
			input:    model.MustRat("1000"),
			digits:   logDigits,
			want:     model.MustRat("3"),
		}, {
			name:     "field ratio of one hundred",
			function: "lgTimes2",
			input:    model.MustRat("100"),
			digits:   logDigits,
			want:     model.MustRat("4"),
		}, {
			name:     "bits for three states",
			function: "ld",
			input:    model.MustRat("3"),
			digits:   logDigits,
			want:     model.MustRat("1.584962500721156181453738943947816508759"),
		}, {
			name:     "centesimal from proper",
			function: "hpC",
			input:    model.MustRat("0.0001"),
			digits:   logDigits,
			want:     model.MustRat("2"),
		}, {
			name:     "square root of two",
			function: "sqrt",
			input:    model.MustRat("2"),
			digits:   logDigits,
			want:     model.MustRat("1.414213562373095048801688724209698078570"),
		}, {
			name:     "logarithm of zero",
			function: "pH",
			input:    model.MustRat("0"),
			wantErr:  model.ErrDomain,
		}, {
			name:     "logarithm of negative",
			function: "lg",
			input:    model.MustRat("-1"),
			wantErr:  model.ErrDomain,
		}, {
			name:     "square root of negative",
			function: "sqrt",
			input:    model.MustRat("-4"),
			wantErr:  model.ErrDomain,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustLookupConversion(t, tc.function)

			// Act
			value, exact, err := sut.FromProper(tc.input, limits)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Conversion.FromProper(%v) error = %v, want %v", tc.input, got, want)
			}
			if got, want := value, tc.want; !cmp.Equal(got, want, modeltest.EquateApprox(tc.digits)) {
				t.Errorf("Conversion.FromProper(%v) = %v, want %v", tc.input, got.FloatString(40), want.FloatString(40))
			}
			if got, want := exact, tc.wantExact; !cmp.Equal(got, want) {
				t.Errorf("Conversion.FromProper(%v) exact = %v, want %v", tc.input, got, want)
			}
		})
	}
}

func TestLookupConversion_UnknownName_ReportsMissing(t *testing.T) {
	t.Parallel()

	// Arrange
	name := "cosTimes100"

	// Act
	conversion, ok := model.LookupConversion(name)

	// Assert
	if got, want := ok, false; !cmp.Equal(got, want) {
		t.Errorf("LookupConversion(%q) ok = %v, want %v", name, got, want)
	}
	if got, want := conversion, (model.Conversion{}); !cmp.Equal(got, want, cmp.AllowUnexported(model.Conversion{})) {
		t.Errorf("LookupConversion(%q) = %v, want %v", name, got, want)
	}
}

func mustLookupConversion(t *testing.T, name string) model.Conversion {
	t.Helper()
	conversion, ok := model.LookupConversion(name)
	if !ok {
		t.Fatalf("LookupConversion(%q) ok = false, want true", name)
	}
	return conversion
}
