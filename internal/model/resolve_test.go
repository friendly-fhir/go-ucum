package model_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/model/modeltest"
	"github.com/friendly-fhir/go-ucum/internal/syntax"
	"github.com/friendly-fhir/go-ucum/internal/syntax/syntaxtest"
)

const maxExponent = 99

// definitions returns a small but representative excerpt of the UCUM tables,
// with any extra units appended.
func definitions(extra ...model.Unit) *model.Definitions {
	return &model.Definitions{
		Prefixes: []model.Prefix{
			{Code: "k", CICode: "K", Value: model.MustRat("1e3")},
			{Code: "d", CICode: "D", Value: model.MustRat("1e-1")},
			{Code: "c", CICode: "C", Value: model.MustRat("1e-2")},
			{Code: "m", CICode: "M", Value: model.MustRat("1e-3")},
		},
		BaseUnits: []model.BaseUnit{
			{Code: "m", CICode: "M"},
			{Code: "s", CICode: "S"},
			{Code: "g", CICode: "G"},
			{Code: "rad", CICode: "RAD"},
			{Code: "K", CICode: "K"},
			{Code: "C", CICode: "C"},
			{Code: "cd", CICode: "CD"},
		},
		Units: append([]model.Unit{
			{Code: "10*", CICode: "10*", Value: model.MustRat("10"), Term: "1"},
			{Code: "l", CICode: "L", Metric: true, Value: model.MustRat("1"), Term: "dm3"},
			{Code: "L", CICode: "L", Metric: true, Value: model.MustRat("1"), Term: "l"},
			{Code: "[in_i]", CICode: "[IN_I]", Value: model.MustRat("2.54"), Term: "cm"},
			{Code: "mol", CICode: "MOL", Metric: true, Value: model.MustRat("6.0221367"), Term: "10*23"},
			{
				Code:     "Cel",
				CICode:   "CEL",
				Metric:   true,
				Special:  true,
				Function: &model.Function{Name: "Cel", Value: model.MustRat("1"), Term: "K"},
			}, {
				Code:     "[pH]",
				CICode:   "[PH]",
				Special:  true,
				Function: &model.Function{Name: "pH", Value: model.MustRat("1"), Term: "mol/l"},
			},
			{Code: "[iU]", CICode: "[IU]", Metric: true, Arbitrary: true, Value: model.MustRat("1"), Term: "1"},
			{Code: "[IU]", CICode: "[IU]", Metric: true, Arbitrary: true, Value: model.MustRat("1"), Term: "[iU]"},
			{Code: "[arb'U]", CICode: "[ARB'U]", Arbitrary: true, Value: model.MustRat("1"), Term: "1"},
		}, extra...),
	}
}

// dimension builds a dimension vector from base-unit exponents in the order
// m, s, g, rad, K, C, cd.
func dimension(exponents ...int) [len(model.BaseUnitCodes)]int {
	var d [len(model.BaseUnitCodes)]int
	copy(d[:], exponents)
	return d
}

func TestNewResolver(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		defs    *model.Definitions
		wantErr error
	}{
		{
			name: "self reference",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "x"},
			),
			wantErr: model.ErrCyclicDefinition,
		}, {
			name: "mutual reference",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "y"},
				model.Unit{Code: "y", CICode: "Y", Value: model.MustRat("1"), Term: "s/x"},
			),
			wantErr: model.ErrCyclicDefinition,
		}, {
			name: "reference through left operand",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "x.s"},
			),
			wantErr: model.ErrCyclicDefinition,
		}, {
			name: "reference through inverse",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "/x"},
			),
			wantErr: model.ErrCyclicDefinition,
		}, {
			name: "reference through group",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "s/(x)"},
			),
			wantErr: model.ErrCyclicDefinition,
		}, {
			name: "unknown function",
			defs: definitions(
				model.Unit{
					Code:     "x",
					CICode:   "X",
					Special:  true,
					Function: &model.Function{Name: "cosTimes100", Value: model.MustRat("1"), Term: "rad"},
				},
			),
			wantErr: model.ErrUnknownFunction,
		}, {
			name: "unknown reference",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "furlong"},
			),
			wantErr: syntax.ErrUnknownUnit,
		}, {
			name: "invalid term",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "m/"},
			),
			wantErr: syntax.ErrInvalidSyntax,
		}, {
			name: "invalid function term",
			defs: definitions(
				model.Unit{
					Code:     "x",
					CICode:   "X",
					Special:  true,
					Function: &model.Function{Name: "ln", Value: model.MustRat("1"), Term: "(1"},
				},
			),
			wantErr: syntax.ErrInvalidSyntax,
		}, {
			name: "special unit in algebra",
			defs: definitions(
				model.Unit{Code: "x", CICode: "X", Value: model.MustRat("1"), Term: "Cel/s"},
			),
			wantErr: model.ErrSpecialAlgebra,
		}, {
			name: "special unit defined over special unit",
			defs: definitions(
				model.Unit{
					Code:     "x",
					CICode:   "X",
					Special:  true,
					Function: &model.Function{Name: "Cel", Value: model.MustRat("1"), Term: "Cel"},
				},
			),
			wantErr: model.ErrSpecialAlgebra,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			resolver, err := model.NewResolver(tc.defs, maxExponent)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("NewResolver() error = %v, want %v", got, want)
			}
			if got, want := resolver, (*model.Resolver)(nil); got != want {
				t.Errorf("NewResolver() = %v, want %v", got, want)
			}
		})
	}
}

func TestResolverResolve(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		code    string
		want    model.Canonical
		wantErr error
	}{
		{
			name: "base unit",
			code: "m",
			want: model.Canonical{
				Factor:    model.MustRat("1"),
				Dimension: dimension(1),
			},
		}, {
			name: "prefixed quotient through synonym chain",
			code: "mg/dL",
			want: model.Canonical{
				Factor:    model.MustRat("10"),
				Dimension: dimension(-3, 0, 1),
			},
		}, {
			name: "exponent applies to prefix",
			code: "cm3",
			want: model.Canonical{
				Factor:    model.MustRat("1e-6"),
				Dimension: dimension(3),
			},
		}, {
			name: "customary unit squared",
			code: "[in_i]2",
			want: model.Canonical{
				Factor:    model.MustRat("0.00064516"),
				Dimension: dimension(2),
			},
		}, {
			name: "inverse",
			code: "/s",
			want: model.Canonical{
				Factor:    model.MustRat("1"),
				Dimension: dimension(0, -1),
			},
		}, {
			name: "negative exponent",
			code: "cm-1",
			want: model.Canonical{
				Factor:    model.MustRat("100"),
				Dimension: dimension(-1),
			},
		}, {
			name: "zero exponent",
			code: "m0",
			want: model.Canonical{
				Factor: model.MustRat("1"),
			},
		}, {
			name: "factor and group",
			code: "g/(4.m.s)",
			want: model.Canonical{
				Factor:    model.MustRat("1/4"),
				Dimension: dimension(-1, -1, 1),
			},
		}, {
			name: "annotation is unity",
			code: "{cells}/mL",
			want: model.Canonical{
				Factor:    model.MustRat("1e6"),
				Dimension: dimension(-3),
			},
		}, {
			name: "dimensionless count",
			code: "mol/L",
			want: model.Canonical{
				Factor:    model.MustRat("6.0221367e26"),
				Dimension: dimension(-3),
			},
		}, {
			name: "special unit",
			code: "Cel",
			want: model.Canonical{
				Dimension: dimension(0, 0, 0, 0, 1),
				Special: &model.Special{
					Name:   "Cel",
					Scale:  model.MustRat("1"),
					Proper: model.MustRat("1"),
				},
			},
		}, {
			name: "prefixed special unit",
			code: "mCel{body}",
			want: model.Canonical{
				Dimension: dimension(0, 0, 0, 0, 1),
				Special: &model.Special{
					Name:   "Cel",
					Scale:  model.MustRat("1e-3"),
					Proper: model.MustRat("1"),
				},
			},
		}, {
			name: "special unit over compound proper unit",
			code: "[pH]",
			want: model.Canonical{
				Dimension: dimension(-3),
				Special: &model.Special{
					Name:   "pH",
					Scale:  model.MustRat("1"),
					Proper: model.MustRat("6.0221367e26"),
				},
			},
		}, {
			name: "arbitrary unit",
			code: "[iU]/mL",
			want: model.Canonical{
				Factor:    model.MustRat("1e6"),
				Dimension: dimension(-3),
				Arbitrary: map[string]int{"[iU]": 1},
			},
		}, {
			name: "inverse arbitrary unit",
			code: "/[iU]",
			want: model.Canonical{
				Factor:    model.MustRat("1"),
				Arbitrary: map[string]int{"[iU]": -1},
			},
		}, {
			name: "arbitrary synonym",
			code: "k[IU]",
			want: model.Canonical{
				Factor:    model.MustRat("1e3"),
				Arbitrary: map[string]int{"[iU]": 1},
			},
		}, {
			name: "squared arbitrary unit",
			code: "[arb'U]2",
			want: model.Canonical{
				Factor:    model.MustRat("1"),
				Arbitrary: map[string]int{"[arb'U]": 2},
			},
		}, {
			name: "arbitrary units cancel",
			code: "[iU]/[IU]",
			want: model.Canonical{
				Factor: model.MustRat("1"),
			},
		}, {
			name:    "special unit with exponent",
			code:    "Cel2",
			wantErr: &syntax.Error{Offset: 0, Err: model.ErrSpecialAlgebra},
		}, {
			name:    "special unit in quotient",
			code:    "m.Cel",
			wantErr: &syntax.Error{Offset: 2, Err: model.ErrSpecialAlgebra},
		}, {
			name:    "special unit in group",
			code:    "(Cel)",
			wantErr: &syntax.Error{Offset: 1, Err: model.ErrSpecialAlgebra},
		}, {
			name:    "special unit inverted",
			code:    "/Cel",
			wantErr: &syntax.Error{Offset: 1, Err: model.ErrSpecialAlgebra},
		}, {
			name:    "special unit in divisor",
			code:    "s/[pH]",
			wantErr: &syntax.Error{Offset: 2, Err: model.ErrSpecialAlgebra},
		}, {
			name:    "unknown unit",
			code:    "furlong",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrUnknownUnit},
		}, {
			name:    "case-insensitive code",
			code:    "MG",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrUnknownUnit},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustNewResolver(t, definitions(), maxExponent)

			// Act
			canonical, err := sut.Resolve(tc.code)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, syntaxtest.EquateErrors()) {
				t.Fatalf("Resolver.Resolve(%q) error = %v, want %v", tc.code, got, want)
			}
			if got, want := canonical, tc.want; !cmp.Equal(got, want, modeltest.EquateCanonicals()) {
				t.Errorf("Resolver.Resolve(%q) = %v, want %v\n%s", tc.code, got, want, cmp.Diff(want, got, modeltest.EquateCanonicals()))
			}
		})
	}
}

func TestResolverResolveInsensitive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		code    string
		want    model.Canonical
		wantErr error
	}{
		{
			name: "upper case",
			code: "MG/DL",
			want: model.Canonical{
				Factor:    model.MustRat("10"),
				Dimension: dimension(-3, 0, 1),
			},
		}, {
			name: "case is not significant",
			code: "mg/dl",
			want: model.Canonical{
				Factor:    model.MustRat("10"),
				Dimension: dimension(-3, 0, 1),
			},
		}, {
			name: "shared code of synonyms",
			code: "L",
			want: model.Canonical{
				Factor:    model.MustRat("1e-3"),
				Dimension: dimension(3),
			},
		}, {
			name: "prefixed special unit",
			code: "MCEL",
			want: model.Canonical{
				Dimension: dimension(0, 0, 0, 0, 1),
				Special: &model.Special{
					Name:   "Cel",
					Scale:  model.MustRat("1e-3"),
					Proper: model.MustRat("1"),
				},
			},
		}, {
			name: "bracketed atom with exponent",
			code: "[in_i]2",
			want: model.Canonical{
				Factor:    model.MustRat("0.00064516"),
				Dimension: dimension(2),
			},
		}, {
			name:    "unknown unit",
			code:    "FURLONG",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrUnknownUnit},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustNewResolver(t, definitions(), maxExponent)

			// Act
			canonical, err := sut.ResolveInsensitive(tc.code)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, syntaxtest.EquateErrors()) {
				t.Fatalf("Resolver.ResolveInsensitive(%q) error = %v, want %v", tc.code, got, want)
			}
			if got, want := canonical, tc.want; !cmp.Equal(got, want, modeltest.EquateCanonicals()) {
				t.Errorf("Resolver.ResolveInsensitive(%q) = %v, want %v\n%s", tc.code, got, want, cmp.Diff(want, got, modeltest.EquateCanonicals()))
			}
		})
	}
}

func mustNewResolver(t *testing.T, defs *model.Definitions, maxExponent int) *model.Resolver {
	t.Helper()
	resolver, err := model.NewResolver(defs, maxExponent)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	return resolver
}

func TestResolverResolve_ExponentLimit_AppliesOnlyToInput(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := mustNewResolver(t, definitions(), 2)
	code := "mol3"

	// Act
	canonical, err := sut.Resolve(code)

	// Assert
	if got, want := err, error(&syntax.Error{Offset: 3, Err: syntax.ErrInvalidSyntax}); !cmp.Equal(got, want, syntaxtest.EquateErrors()) {
		t.Fatalf("Resolver.Resolve(%q) error = %v, want %v", code, got, want)
	}
	if got, want := canonical, (model.Canonical{}); !cmp.Equal(got, want, modeltest.EquateCanonicals()) {
		t.Errorf("Resolver.Resolve(%q) = %v, want %v", code, got, want)
	}
}
