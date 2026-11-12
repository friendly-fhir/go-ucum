package syntax_test

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/friendly-fhir/go-ucum/internal/syntax"
	"github.com/friendly-fhir/go-ucum/internal/syntax/syntaxtest"
)

// table is a fixed set of symbols chosen to exercise the ambiguous parts of
// the grammar. Identities are positions in the slices.
type table struct {
	atoms    []atom
	prefixes []string
}

type atom struct {
	code   string
	metric bool
}

func (t table) Atom(code string) (int, bool, bool) {
	for i, a := range t.atoms {
		if a.code == code {
			return i, a.metric, true
		}
	}
	return 0, false, false
}

func (t table) Prefix(code string) (int, bool) {
	for i, p := range t.prefixes {
		if p == code {
			return i, true
		}
	}
	return 0, false
}

var _ syntax.Lookup = table{}

const (
	meter = iota
	second
	gram
	inch
	tenStar
	am
	candela
	meterHg
	bracketed
)

const (
	kilo = iota
	centi
	deci
	deka
)

var symbols = table{
	atoms: []atom{
		meter:     {code: "m", metric: true},
		second:    {code: "s", metric: true},
		gram:      {code: "g", metric: true},
		inch:      {code: "[in_i]", metric: false},
		tenStar:   {code: "10*", metric: false},
		am:        {code: "am", metric: true},
		candela:   {code: "cd", metric: true},
		meterHg:   {code: "m[Hg]", metric: true},
		bracketed: {code: "[m/s2/Hz^(1/2)]", metric: false},
	},
	prefixes: []string{
		kilo:  "k",
		centi: "c",
		deci:  "d",
		deka:  "da",
	},
}

const maxExponent = 99

func TestParse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    syntax.Node
		wantErr error
	}{
		{
			name:  "atom",
			input: "m",
			want:  syntax.Symbol{Offset: 0, Atom: meter},
		}, {
			name:  "prefix with exponent",
			input: "km2",
			want:  syntax.Symbol{Offset: 0, HasPrefix: true, Prefix: kilo, Atom: meter, HasExponent: true, Exponent: 2},
		}, {
			name:  "whole atom before prefix split",
			input: "cd",
			want:  syntax.Symbol{Offset: 0, Atom: candela},
		}, {
			name:  "longest prefix",
			input: "dam",
			want:  syntax.Symbol{Offset: 0, HasPrefix: true, Prefix: deka, Atom: meter},
		}, {
			name:  "prefixed atom containing brackets",
			input: "cm[Hg]",
			want:  syntax.Symbol{Offset: 0, HasPrefix: true, Prefix: centi, Atom: meterHg},
		}, {
			name:  "bracketed atom with exponent",
			input: "[in_i]2",
			want:  syntax.Symbol{Offset: 0, Atom: inch, HasExponent: true, Exponent: 2},
		}, {
			name:  "operators inside brackets",
			input: "[m/s2/Hz^(1/2)]",
			want:  syntax.Symbol{Offset: 0, Atom: bracketed},
		}, {
			name:  "negative exponent",
			input: "10*-3",
			want:  syntax.Symbol{Offset: 0, Atom: tenStar, HasExponent: true, Exponent: -3},
		}, {
			name:  "signed positive exponent",
			input: "10*+3",
			want:  syntax.Symbol{Offset: 0, Atom: tenStar, HasExponent: true, Exponent: 3},
		}, {
			name:  "exponent at limit",
			input: "m-99",
			want:  syntax.Symbol{Offset: 0, Atom: meter, HasExponent: true, Exponent: -99},
		}, {
			name:  "leading solidus",
			input: "/s",
			want: syntax.Inverse{
				Operand: syntax.Symbol{Offset: 1, Atom: second},
			},
		}, {
			name:  "left associativity",
			input: "m/s/g",
			want: syntax.Binary{
				Op: syntax.Divide,
				Left: syntax.Binary{
					Op:    syntax.Divide,
					Left:  syntax.Symbol{Offset: 0, Atom: meter},
					Right: syntax.Symbol{Offset: 2, Atom: second},
				},
				Right: syntax.Symbol{Offset: 4, Atom: gram},
			},
		}, {
			name:  "parentheses",
			input: "g/(m.s)",
			want: syntax.Binary{
				Op:   syntax.Divide,
				Left: syntax.Symbol{Offset: 0, Atom: gram},
				Right: syntax.Group{
					Term: syntax.Binary{
						Op:    syntax.Multiply,
						Left:  syntax.Symbol{Offset: 3, Atom: meter},
						Right: syntax.Symbol{Offset: 5, Atom: second},
					},
				},
			},
		}, {
			name:  "factor",
			input: "4.[in_i]",
			want: syntax.Binary{
				Op:    syntax.Multiply,
				Left:  syntax.Factor{Value: big.NewInt(4)},
				Right: syntax.Symbol{Offset: 2, Atom: inch},
			},
		}, {
			name:  "factor beyond int64",
			input: "/100000000000000000000",
			want: syntax.Inverse{
				Operand: syntax.Factor{Value: new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)},
			},
		}, {
			name:  "standalone annotation",
			input: "{cells}",
			want:  syntax.Unity{},
		}, {
			name:  "annotated symbol",
			input: "m2{area}",
			want:  syntax.Symbol{Offset: 0, Atom: meter, HasExponent: true, Exponent: 2},
		}, {
			name:  "annotations as operands",
			input: "{a}.m{b}",
			want: syntax.Binary{
				Op:    syntax.Multiply,
				Left:  syntax.Unity{},
				Right: syntax.Symbol{Offset: 4, Atom: meter},
			},
		}, {
			name:  "operators inside annotation",
			input: "/{tot.(x)/[y]}",
			want: syntax.Inverse{
				Operand: syntax.Unity{},
			},
		}, {
			name:    "empty",
			input:   "",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "trailing operator",
			input:   "m/",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "leading multiplication",
			input:   ".m",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "double operator",
			input:   "m..s",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "unopened parenthesis",
			input:   "m)",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "unclosed parenthesis",
			input:   "g/(m",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "empty parentheses",
			input:   "()",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "exponent on parentheses",
			input:   "(m)2",
			wantErr: &syntax.Error{Offset: 3, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "solidus opening parentheses",
			input:   "(/m)",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "unclosed annotation",
			input:   "m{x",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "nested annotation",
			input:   "{a{b}}",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "space in annotation",
			input:   "{x y}",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "non-ASCII annotation",
			input:   "m{錠}",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "annotation before symbol",
			input:   "{a}m",
			wantErr: &syntax.Error{Offset: 3, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "annotation after factor",
			input:   "1{c}",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "stray closing brace",
			input:   "m}",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "unclosed bracket",
			input:   "k[in_i",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "nested bracket",
			input:   "[a[b]]",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "unmatched bracket",
			input:   "m]",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "space in bracket",
			input:   "[in i]",
			wantErr: &syntax.Error{Offset: 3, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "space between units",
			input:   "m s",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "zero factor",
			input:   "m/00",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "exponent without unit",
			input:   "+3",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "exponent beyond limit",
			input:   "m.s100",
			wantErr: &syntax.Error{Offset: 3, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "exponent beyond int",
			input:   "m99999999999999999999",
			wantErr: &syntax.Error{Offset: 1, Err: syntax.ErrInvalidSyntax},
		}, {
			name:    "unknown atom",
			input:   "m/xyz",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrUnknownUnit},
		}, {
			name:    "digits glued to unit",
			input:   "g/12s",
			wantErr: &syntax.Error{Offset: 2, Err: syntax.ErrUnknownUnit},
		}, {
			name:    "factor with exponent",
			input:   "10+3",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrUnknownUnit},
		}, {
			name:    "prefix on non-metric atom",
			input:   "k[in_i]",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrUnknownUnit},
		}, {
			name:    "prefix alone",
			input:   "k",
			wantErr: &syntax.Error{Offset: 0, Err: syntax.ErrUnknownUnit},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			node, err := syntax.Parse(tc.input, symbols, maxExponent)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, syntaxtest.EquateErrors()) {
				t.Fatalf("Parse(%q) error = %v, want %v", tc.input, got, want)
			}
			if got, want := node, tc.want; !cmp.Equal(got, want, syntaxtest.EquateNodes()) {
				t.Errorf("Parse(%q) = %v, want %v\n%s", tc.input, got, want, cmp.Diff(want, got, syntaxtest.EquateNodes()))
			}
		})
	}
}

func TestErrorError_UnknownUnit_DescribesOffsetAndCause(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := &syntax.Error{Offset: 2, Err: syntax.ErrUnknownUnit}

	// Act
	message := sut.Error()

	// Assert
	if got, want := strings.Contains(message, "offset 2"), true; !cmp.Equal(got, want) {
		t.Errorf("Error.Error() = %q, want it to contain the offset", message)
	}
	if got, want := strings.Contains(message, syntax.ErrUnknownUnit.Error()), true; !cmp.Equal(got, want) {
		t.Errorf("Error.Error() = %q, want it to contain the cause", message)
	}
}

func TestError_WrappedCause_MatchesSentinel(t *testing.T) {
	t.Parallel()

	// Arrange
	symbol := "xyz"

	// Act
	_, err := syntax.Parse(symbol, symbols, maxExponent)

	// Assert
	if got, want := err, syntax.ErrUnknownUnit; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Fatalf("Parse(%q) error = %v, want %v", symbol, got, want)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{"km2", "/s", "g/(m.s)", "4.[in_i]{x}", "10*-3", "[m/s2/Hz^(1/2)]", "m{"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// Act
		_, err := syntax.Parse(input, symbols, maxExponent)

		// Assert
		if got, want := offsetWithin(err, input), true; !cmp.Equal(got, want) {
			t.Errorf("Parse(%q) error = %v, want nil or a *syntax.Error with an offset inside the input", input, err)
		}
	})
}

func offsetWithin(err error, input string) bool {
	if err == nil {
		return true
	}
	var syntaxErr *syntax.Error
	return errors.As(err, &syntaxErr) && 0 <= syntaxErr.Offset && syntaxErr.Offset <= len(input)
}
