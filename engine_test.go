package ucum_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum"
)

const essence = "testdata/essence.xml"

var equateDecimals = cmp.Comparer(func(lhs, rhs decimal.Decimal) bool {
	return lhs.Equal(rhs)
})

// unitView is the observable state of a [ucum.Unit].
type unitView struct {
	Code      string
	Dimension string
	Special   bool
	Arbitrary bool
}

func view(u *ucum.Unit) *unitView {
	if u == nil {
		return nil
	}
	return &unitView{
		Code:      u.String(),
		Dimension: u.Dimension().String(),
		Special:   u.IsSpecial(),
		Arbitrary: u.IsArbitrary(),
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		opts    []ucum.Option
		wantErr error
	}{
		{
			name:    "cyclic definitions",
			path:    "testdata/invalid/cyclic.xml",
			wantErr: ucum.ErrCyclicDefinition,
		}, {
			name:    "unknown special function",
			path:    "testdata/invalid/function.xml",
			wantErr: ucum.ErrUnknownFunction,
		}, {
			name:    "zero precision",
			path:    essence,
			opts:    []ucum.Option{ucum.WithPrecision(0)},
			wantErr: ucum.ErrInvalidOption,
		}, {
			name:    "zero maximum exponent",
			path:    essence,
			opts:    []ucum.Option{ucum.WithMaxExponent(0)},
			wantErr: ucum.ErrInvalidOption,
		}, {
			name:    "negative maximum magnitude",
			path:    essence,
			opts:    []ucum.Option{ucum.WithMaxMagnitude(-1)},
			wantErr: ucum.ErrInvalidOption,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			registry := loadRegistry(t, tc.path)

			// Act
			engine, err := ucum.New(registry, tc.opts...)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("New() error = %v, want %v", got, want)
			}
			if got, want := engine, (*ucum.Engine)(nil); got != want {
				t.Errorf("New() = %v, want %v", got, want)
			}
		})
	}
}

func TestEngineParseUnit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		code    string
		want    *unitView
		wantErr error
	}{
		{
			name: "prefixed quotient",
			code: "mg/dL",
			want: &unitView{Code: "mg/dL", Dimension: "m-3.g"},
		}, {
			name: "annotation",
			code: "{cells}/uL",
			want: &unitView{Code: "{cells}/uL", Dimension: "m-3"},
		}, {
			name: "special unit",
			code: "Cel",
			want: &unitView{Code: "Cel", Dimension: "K", Special: true},
		}, {
			name: "arbitrary unit",
			code: "[iU]/mL",
			want: &unitView{Code: "[iU]/mL", Dimension: "m-3", Arbitrary: true},
		}, {
			name:    "invalid syntax",
			code:    "m/",
			wantErr: ucum.ErrInvalidSyntax,
		}, {
			name:    "unknown unit",
			code:    "furlong",
			wantErr: ucum.ErrUnknownUnit,
		}, {
			name:    "case-insensitive code",
			code:    "MG",
			wantErr: ucum.ErrUnknownUnit,
		}, {
			name:    "special unit in algebra",
			code:    "Cel/h",
			wantErr: ucum.ErrSpecialAlgebra,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t)

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

func TestEngineParseUnit_InvalidSyntax_ReportsLocation(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := newEngine(t)
	code := "g/(m.s"

	// Act
	_, err := sut.ParseUnit(code)

	// Assert
	parseErr := asParseError(t, err)
	if got, want := *parseErr, (ucum.ParseError{Input: code, Offset: 2, Err: ucum.ErrInvalidSyntax}); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Errorf("Engine.ParseUnit(%q) error = %+v, want %+v", code, got, want)
	}
}

func TestEngineParseUnitInsensitive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		code    string
		want    *unitView
		wantErr error
	}{
		{
			name: "upper case",
			code: "MG/DL",
			want: &unitView{Code: "MG/DL", Dimension: "m-3.g"},
		}, {
			name: "special unit",
			code: "CEL",
			want: &unitView{Code: "CEL", Dimension: "K", Special: true},
		}, {
			name:    "invalid syntax",
			code:    "M/",
			wantErr: ucum.ErrInvalidSyntax,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t)

			// Act
			unit, err := sut.ParseUnitInsensitive(tc.code)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Engine.ParseUnitInsensitive(%q) error = %v, want %v", tc.code, got, want)
			}
			if got, want := view(unit), tc.want; !cmp.Equal(got, want) {
				t.Errorf("Engine.ParseUnitInsensitive(%q) = %+v, want %+v", tc.code, got, want)
			}
		})
	}
}

func TestEngineCompatible(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{
			name: "same dimension",
			from: "m",
			to:   "[in_i]",
			want: true,
		}, {
			name: "special and proper unit",
			from: "Cel",
			to:   "K",
			want: true,
		}, {
			name: "two special units",
			from: "Cel",
			to:   "[degF]",
			want: true,
		}, {
			name: "same arbitrary unit",
			from: "[iU]/mL",
			to:   "[IU]/L",
			want: true,
		}, {
			name: "different dimensions",
			from: "m",
			to:   "s",
			want: false,
		}, {
			name: "different arbitrary units",
			from: "[iU]",
			to:   "[arb'U]",
			want: false,
		}, {
			name: "arbitrary and dimensionless unit",
			from: "[iU]",
			to:   "1",
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t)
			from := mustParseUnit(t, sut, tc.from)
			to := mustParseUnit(t, sut, tc.to)

			// Act
			compatible := sut.Compatible(from, to)

			// Assert
			if got, want := compatible, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Engine.Compatible(%q, %q) = %v, want %v", tc.from, tc.to, got, want)
			}
		})
	}
}

func TestEngineCompatible_ForeignUnit_ReportsIncompatible(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := newEngine(t)
	from := mustParseUnit(t, sut, "m")
	to := mustParseUnit(t, newEngine(t), "m")

	// Act
	compatible := sut.Compatible(from, to)

	// Assert
	if got, want := compatible, false; !cmp.Equal(got, want) {
		t.Errorf("Engine.Compatible(%v, %v) = %v, want %v", from, to, got, want)
	}
}

func TestEngineConvert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   string
		from    string
		to      string
		want    decimal.Decimal
		wantErr error
	}{
		{
			name:  "mass concentration",
			value: "100",
			from:  "mg/dL",
			to:    "g/L",
			want:  decimal.RequireFromString("1"),
		}, {
			name:  "terminating result is exact",
			value: "1",
			from:  "[in_i]",
			to:    "cm",
			want:  decimal.RequireFromString("2.54"),
		}, {
			name:  "repeating result is rounded",
			value: "1",
			from:  "m",
			to:    "[in_i]",
			want:  decimal.RequireFromString("39.37007874015748031496062992125984"),
		}, {
			name:  "small repeating result keeps significant digits",
			value: "1",
			from:  "s",
			to:    "h",
			want:  decimal.RequireFromString("0.0002777777777777777777777777777777778"),
		}, {
			name:  "zero",
			value: "0",
			from:  "m",
			to:    "[in_i]",
			want:  decimal.Zero,
		}, {
			name:  "special to proper unit",
			value: "37",
			from:  "Cel",
			to:    "K",
			want:  decimal.RequireFromString("310.15"),
		}, {
			name:  "between special units",
			value: "37",
			from:  "Cel",
			to:    "[degF]",
			want:  decimal.RequireFromString("98.6"),
		}, {
			name:  "prefixed special unit",
			value: "1",
			from:  "mCel",
			to:    "Cel",
			want:  decimal.RequireFromString("0.001"),
		}, {
			name:  "logarithmic unit",
			value: "7",
			from:  "[pH]",
			to:    "mol/L",
			want:  decimal.RequireFromString("0.0000001"),
		}, {
			name:  "level to ratio",
			value: "2",
			from:  "B",
			to:    "%",
			want:  decimal.RequireFromString("10000"),
		}, {
			name:  "same arbitrary unit",
			value: "1",
			from:  "[iU]/mL",
			to:    "[iU]/L",
			want:  decimal.RequireFromString("1000"),
		}, {
			name:  "arbitrary synonym",
			value: "1",
			from:  "k[IU]",
			to:    "[iU]",
			want:  decimal.RequireFromString("1000"),
		}, {
			name:    "different dimensions",
			value:   "1",
			from:    "m",
			to:      "s",
			wantErr: ucum.ErrIncompatible,
		}, {
			name:    "different arbitrary units",
			value:   "1",
			from:    "[iU]",
			to:      "[arb'U]",
			wantErr: ucum.ErrArbitrary,
		}, {
			name:    "outside special domain",
			value:   "0",
			from:    "mol/L",
			to:      "[pH]",
			wantErr: ucum.ErrDomain,
		}, {
			name:    "special result beyond magnitude",
			value:   "2000",
			from:    "B",
			to:      "1",
			wantErr: ucum.ErrDomain,
		}, {
			name:    "large input",
			value:   "1e1001",
			from:    "m",
			to:      "m",
			wantErr: ucum.ErrDomain,
		}, {
			name:    "small input",
			value:   "1e-1001",
			from:    "m",
			to:      "m",
			wantErr: ucum.ErrDomain,
		}, {
			name:    "large result",
			value:   "1e1000",
			from:    "km",
			to:      "m",
			wantErr: ucum.ErrDomain,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := newEngine(t)
			value := decimal.RequireFromString(tc.value)
			from := mustParseUnit(t, sut, tc.from)
			to := mustParseUnit(t, sut, tc.to)

			// Act
			result, err := sut.Convert(value, from, to)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Engine.Convert(%v, %q, %q) error = %v, want %v", value, tc.from, tc.to, got, want)
			}
			if got, want := result, tc.want; !cmp.Equal(got, want, equateDecimals) {
				t.Errorf("Engine.Convert(%v, %q, %q) = %v, want %v", value, tc.from, tc.to, got, want)
			}
		})
	}
}

func TestEngineConvert_ForeignUnit_ReturnsError(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := newEngine(t)
	other := newEngine(t)
	from := mustParseUnit(t, other, "m")
	to := mustParseUnit(t, other, "m")
	value := decimal.RequireFromString("1")

	// Act
	result, err := sut.Convert(value, from, to)

	// Assert
	if got, want := err, ucum.ErrForeignUnit; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Engine.Convert(%v, %q, %q) error = %v, want %v", value, from, to, got, want)
	}
	if got, want := result, (decimal.Decimal{}); !cmp.Equal(got, want, equateDecimals) {
		t.Errorf("Engine.Convert(%v, %q, %q) = %v, want %v", value, from, to, got, want)
	}
}

func loadRegistry(t *testing.T, path string) *ucum.Registry {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open(%q) error = %v", path, err)
	}
	defer file.Close()
	registry, err := ucum.LoadXML(file)
	if err != nil {
		t.Fatalf("LoadXML(%q) error = %v", path, err)
	}
	return registry
}

func newEngine(t *testing.T, opts ...ucum.Option) *ucum.Engine {
	t.Helper()
	engine, err := ucum.New(loadRegistry(t, essence), opts...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return engine
}

func mustParseUnit(t *testing.T, engine *ucum.Engine, code string) *ucum.Unit {
	t.Helper()
	unit, err := engine.ParseUnit(code)
	if err != nil {
		t.Fatalf("Engine.ParseUnit(%q) error = %v", code, err)
	}
	return unit
}

func asParseError(t *testing.T, err error) *ucum.ParseError {
	t.Helper()
	parseErr, ok := err.(*ucum.ParseError)
	if !ok {
		t.Fatalf("error = %#v, want *ucum.ParseError", err)
	}
	return parseErr
}
