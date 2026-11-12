package xmldef

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"slices"
	"strings"

	"github.com/friendly-fhir/go-ucum/internal/model"
)

var (
	// ErrInvalidDefinition reports a definitions document that is malformed or
	// contains an incomplete definition.
	ErrInvalidDefinition = errors.New("ucum: invalid definition")

	// ErrUnsupportedBase reports a definitions document declaring a base unit
	// outside the seven UCUM base units.
	ErrUnsupportedBase = errors.New("ucum: unsupported base unit")
)

type document struct {
	XMLName      xml.Name   `xml:"root"`
	Version      string     `xml:"version,attr"`
	RevisionDate string     `xml:"revision-date,attr"`
	Prefixes     []prefix   `xml:"prefix"`
	BaseUnits    []baseUnit `xml:"base-unit"`
	Units        []unit     `xml:"unit"`
}

type prefix struct {
	Code   string `xml:"Code,attr"`
	CICode string `xml:"CODE,attr"`
	Value  value  `xml:"value"`
}

type baseUnit struct {
	Code   string `xml:"Code,attr"`
	CICode string `xml:"CODE,attr"`
}

type unit struct {
	Code        string `xml:"Code,attr"`
	CICode      string `xml:"CODE,attr"`
	IsMetric    string `xml:"isMetric,attr"`
	IsSpecial   string `xml:"isSpecial,attr"`
	IsArbitrary string `xml:"isArbitrary,attr"`
	Value       value  `xml:"value"`
}

type value struct {
	Unit     string    `xml:"Unit,attr"`
	Value    string    `xml:"value,attr"`
	Function *function `xml:"function"`
}

type function struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	Unit  string `xml:"Unit,attr"`
}

// Decode reads a ucum-essence document from r.
//
// Decode returns [ErrUnsupportedBase] if the document declares a base
// unit other than the seven UCUM base units, and [ErrInvalidDefinition]
// if the document is malformed or any prefix or unit lacks a complete, exact
// definition.
func Decode(r io.Reader) (*model.Definitions, error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charsetReader

	var doc document
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDefinition, err)
	}

	return doc.definitions()
}

func (d *document) definitions() (*model.Definitions, error) {
	defs := &model.Definitions{
		Version:      d.Version,
		RevisionDate: d.RevisionDate,
	}
	prefixes, atoms := symbols{}, symbols{}
	for _, p := range d.Prefixes {
		mp, err := p.model()
		if err != nil {
			return nil, err
		}
		if err := prefixes.add("prefix", mp.Code); err != nil {
			return nil, err
		}
		defs.Prefixes = append(defs.Prefixes, mp)
	}
	for _, b := range d.BaseUnits {
		if !slices.Contains(model.BaseUnitCodes[:], b.Code) {
			return nil, fmt.Errorf("%w: %q", ErrUnsupportedBase, b.Code)
		}
		if err := atoms.add("unit", b.Code); err != nil {
			return nil, err
		}
		defs.BaseUnits = append(defs.BaseUnits, model.BaseUnit{Code: b.Code, CICode: b.CICode})
	}
	for _, u := range d.Units {
		mu, err := u.model()
		if err != nil {
			return nil, err
		}
		if err := atoms.add("unit", mu.Code); err != nil {
			return nil, err
		}
		defs.Units = append(defs.Units, mu)
	}
	return defs, nil
}

// symbols records the case-sensitive codes already defined in one namespace.
// Case-insensitive codes may repeat: the published tables give synonyms such
// as "l" and "L" the same case-insensitive code.
type symbols map[string]struct{}

func (s symbols) add(kind, code string) error {
	if _, ok := s[code]; ok {
		return fmt.Errorf("%w: %s %q defined twice", ErrInvalidDefinition, kind, code)
	}
	s[code] = struct{}{}
	return nil
}

// charsetReader accepts the ASCII declaration used by the published
// documents. ASCII is a subset of UTF-8, so the input is used unchanged.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(label) {
	case "ascii", "us-ascii":
		return input, nil
	}
	return nil, fmt.Errorf("unsupported encoding %q", label)
}

func (p prefix) model() (model.Prefix, error) {
	if p.Code == "" {
		return model.Prefix{}, fmt.Errorf("%w: prefix without a code", ErrInvalidDefinition)
	}
	v, err := parseRat(p.Value.Value)
	if err != nil {
		return model.Prefix{}, fmt.Errorf("%w: prefix %q: %w", ErrInvalidDefinition, p.Code, err)
	}
	return model.Prefix{Code: p.Code, CICode: p.CICode, Value: v}, nil
}

func (u unit) model() (model.Unit, error) {
	if u.Code == "" {
		return model.Unit{}, fmt.Errorf("%w: unit without a code", ErrInvalidDefinition)
	}
	mu, err := u.decode()
	if err != nil {
		return model.Unit{}, fmt.Errorf("%w: unit %q: %w", ErrInvalidDefinition, u.Code, err)
	}
	return mu, nil
}

func (u unit) decode() (model.Unit, error) {
	mu := model.Unit{Code: u.Code, CICode: u.CICode}
	var err error
	if mu.Metric, err = u.flag("isMetric", u.IsMetric); err != nil {
		return mu, err
	}
	if mu.Special, err = u.flag("isSpecial", u.IsSpecial); err != nil {
		return mu, err
	}
	if mu.Arbitrary, err = u.flag("isArbitrary", u.IsArbitrary); err != nil {
		return mu, err
	}

	fn := u.Value.Function
	switch {
	case mu.Special && fn == nil:
		return mu, fmt.Errorf("special unit has no function")
	case !mu.Special && fn != nil:
		return mu, fmt.Errorf("function given for a unit that is not special")
	case mu.Special:
		if fn.Name == "" || fn.Unit == "" {
			return mu, fmt.Errorf("function needs a name and a unit")
		}
		v, err := parseRat(fn.Value)
		if err != nil {
			return mu, fmt.Errorf("function: %w", err)
		}
		mu.Function = &model.Function{Name: fn.Name, Value: v, Term: fn.Unit}
		return mu, nil
	}

	if u.Value.Unit == "" {
		return mu, fmt.Errorf("value has no unit")
	}
	if mu.Value, err = parseRat(u.Value.Value); err != nil {
		return mu, err
	}
	mu.Term = u.Value.Unit
	return mu, nil
}

func (unit) flag(name, s string) (bool, error) {
	switch s {
	case "yes":
		return true, nil
	case "no", "":
		return false, nil
	}
	return false, fmt.Errorf("%s must be yes or no, got %q", name, s)
}

// parseRat parses a magnitude, which must be an exact positive number.
func parseRat(s string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() <= 0 {
		return nil, fmt.Errorf("value %q is not an exact positive number", s)
	}
	return r, nil
}
