package xmldef_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/model/modeltest"
	"github.com/friendly-fhir/go-ucum/internal/xmldef"
)

// essence wraps body in the root element of a ucum-essence document, using
// the same declaration as the published files.
func essence(body string) string {
	return `<?xml version="1.0" encoding="ascii"?>
<root xmlns="http://unitsofmeasure.org/ucum-essence" version="2.2" revision-date="2024-06-17">` +
		body + `</root>`
}

func TestDecode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    *model.Definitions
		wantErr error
	}{
		{
			name: "published document",
			input: essence(`
   <prefix Code="k" CODE="K"><name>kilo</name><value value="1e3">1 &#215; 10<sup>3</sup></value></prefix>
   <base-unit Code="m" CODE="M" dim="L"><name>meter</name></base-unit>
   <base-unit Code="K" CODE="K" dim="C"><name>kelvin</name></base-unit>
   <unit Code="[in_i]" CODE="[IN_I]" isMetric="no" class="intcust">
      <name>inch</name><value Unit="cm" UNIT="CM" value="254e-2">2.54</value>
   </unit>
   <unit Code="[iU]" CODE="[IU]" isMetric="yes" isArbitrary="yes" class="chemical">
      <value Unit="1" UNIT="1" value="1">1</value>
   </unit>
   <unit Code="[degF]" CODE="[DEGF]" isMetric="no" isSpecial="yes" class="heat">
      <value Unit="degf(5 K/9)" UNIT="DEGF(5 K/9)"><function name="degF" value="5" Unit="K/9"/></value>
   </unit>`),
			want: &model.Definitions{
				Version:      "2.2",
				RevisionDate: "2024-06-17",
				Prefixes: []model.Prefix{
					{Code: "k", CICode: "K", Value: big.NewRat(1000, 1)},
				},
				BaseUnits: []model.BaseUnit{
					{Code: "m", CICode: "M"},
					{Code: "K", CICode: "K"},
				},
				Units: []model.Unit{
					{
						Code:   "[in_i]",
						CICode: "[IN_I]",
						Value:  big.NewRat(254, 100),
						Term:   "cm",
					}, {
						Code:      "[iU]",
						CICode:    "[IU]",
						Metric:    true,
						Arbitrary: true,
						Value:     big.NewRat(1, 1),
						Term:      "1",
					}, {
						Code:    "[degF]",
						CICode:  "[DEGF]",
						Special: true,
						Function: &model.Function{
							Name:  "degF",
							Value: big.NewRat(5, 1),
							Term:  "K/9",
						},
					},
				},
			},
		}, {
			name:  "prefix and unit sharing a code",
			input: `<root><prefix Code="m" CODE="M"><value value="1e-3"/></prefix><base-unit Code="m" CODE="M"/></root>`,
			want: &model.Definitions{
				Prefixes: []model.Prefix{
					{Code: "m", CICode: "M", Value: big.NewRat(1, 1000)},
				},
				BaseUnits: []model.BaseUnit{
					{Code: "m", CICode: "M"},
				},
			},
		}, {
			name:  "synonyms sharing a case-insensitive code",
			input: `<root><unit Code="l" CODE="L"><value Unit="1" value="1"/></unit><unit Code="L" CODE="L"><value Unit="l" value="1"/></unit></root>`,
			want: &model.Definitions{
				Units: []model.Unit{
					{Code: "l", CICode: "L", Value: big.NewRat(1, 1), Term: "1"},
					{Code: "L", CICode: "L", Value: big.NewRat(1, 1), Term: "l"},
				},
			},
		}, {
			name:  "UTF-8 without declaration",
			input: `<root version="9.9"><base-unit Code="s" CODE="S"/></root>`,
			want: &model.Definitions{
				Version: "9.9",
				BaseUnits: []model.BaseUnit{
					{Code: "s", CICode: "S"},
				},
			},
		}, {
			name:    "malformed XML",
			input:   `<root><unit>`,
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "unsupported encoding",
			input:   `<?xml version="1.0" encoding="ebcdic"?><root/>`,
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "wrong root element",
			input:   `<ucum/>`,
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "base unit outside UCUM",
			input:   essence(`<base-unit Code="mol" CODE="MOL" dim="N"/>`),
			wantErr: xmldef.ErrUnsupportedBase,
		}, {
			name:    "prefix without code",
			input:   essence(`<prefix CODE="K"><value value="1e3"/></prefix>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "prefix with inexact value",
			input:   essence(`<prefix Code="k" CODE="K"><value value="many"/></prefix>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "prefix with zero value",
			input:   essence(`<prefix Code="k" CODE="K"><value value="0"/></prefix>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "unit with negative value",
			input:   essence(`<unit Code="x"><value Unit="m" value="-1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "unit without code",
			input:   essence(`<unit CODE="X"><value Unit="m" value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "invalid isMetric",
			input:   essence(`<unit Code="x" isMetric="maybe"><value Unit="m" value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "invalid isSpecial",
			input:   essence(`<unit Code="x" isSpecial="1"><value Unit="m" value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "invalid isArbitrary",
			input:   essence(`<unit Code="x" isArbitrary="true"><value Unit="m" value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "special unit without function",
			input:   essence(`<unit Code="x" isSpecial="yes"><value Unit="m" value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "function on proper unit",
			input:   essence(`<unit Code="x"><value Unit="m" value="1"><function name="ln" value="1" Unit="1"/></value></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "function without name",
			input:   essence(`<unit Code="x" isSpecial="yes"><value><function value="1" Unit="1"/></value></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "function without unit",
			input:   essence(`<unit Code="x" isSpecial="yes"><value><function name="ln" value="1"/></value></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "function with inexact value",
			input:   essence(`<unit Code="x" isSpecial="yes"><value><function name="ln" value="e" Unit="1"/></value></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "proper unit without term",
			input:   essence(`<unit Code="x"><value value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "duplicate prefix",
			input:   essence(`<prefix Code="k" CODE="K"><value value="1e3"/></prefix><prefix Code="k" CODE="KK"><value value="1e3"/></prefix>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "unit redefining base unit",
			input:   essence(`<base-unit Code="m" CODE="M" dim="L"/><unit Code="m" CODE="MM"><value Unit="1" value="1"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "duplicate base unit",
			input:   essence(`<base-unit Code="m" CODE="M" dim="L"/><base-unit Code="m" CODE="M" dim="L"/>`),
			wantErr: xmldef.ErrInvalidDefinition,
		}, {
			name:    "proper unit without value",
			input:   essence(`<unit Code="x"><value Unit="m"/></unit>`),
			wantErr: xmldef.ErrInvalidDefinition,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := strings.NewReader(tc.input)

			// Act
			defs, err := xmldef.Decode(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("Decode() error = %v, want %v", got, want)
			}
			if got, want := defs, tc.want; !cmp.Equal(got, want, modeltest.EquateRats()) {
				t.Errorf("Decode() = %v, want %v\n%s", got, want, cmp.Diff(want, got, modeltest.EquateRats()))
			}
		})
	}
}
