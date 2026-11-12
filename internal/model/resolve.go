package model

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"strings"

	"github.com/friendly-fhir/go-ucum/internal/syntax"
)

var (
	// ErrCyclicDefinition reports a unit whose definition depends on itself.
	ErrCyclicDefinition = errors.New("ucum: cyclic definition")

	// ErrUnknownFunction reports a special unit whose conversion function is
	// not one this package implements.
	ErrUnknownFunction = errors.New("ucum: unknown special function")

	// ErrSpecialAlgebra reports a special unit used in a product, quotient,
	// or power, which UCUM does not permit because special units are not
	// ratio scales.
	ErrSpecialAlgebra = errors.New("ucum: special unit used in unit algebra")
)

// Canonical is a unit reduced to the UCUM base units.
type Canonical struct {
	// Factor is the magnitude of one of the unit in base units. It is nil for
	// special units, which have no such factor.
	Factor *big.Rat

	// Dimension holds the exponent of each base unit, in the order of
	// [BaseUnitCodes].
	Dimension [len(BaseUnitCodes)]int

	// Arbitrary holds the exponent of each arbitrary unit the unit is defined
	// in, keyed by its case-sensitive code.
	Arbitrary map[string]int

	// Special is set for special units.
	Special *Special
}

// Special relates a special unit to its proper unit.
type Special struct {
	// Name is the name of the conversion function.
	Name string

	Conversion Conversion

	// Scale is the factor applied to a value before conversion, which is how
	// a prefix acts on a special unit.
	Scale *big.Rat

	// Proper is the magnitude, in base units, of one unit of the proper scale
	// the conversion produces.
	Proper *big.Rat
}

// Resolver reduces unit expressions to canonical form against one set of
// definitions. A Resolver is safe for concurrent use.
type Resolver struct {
	maxExponent int
	atoms       []atom
	prefixes    []Prefix
	sensitive   lookup
	insensitive lookup
}

type atom struct {
	code      string
	metric    bool
	canonical Canonical
}

// NewResolver resolves every unit in defs. Expressions later given to the
// resolver may not use exponents whose magnitude exceeds maxExponent; the
// definitions themselves are trusted and not limited.
//
// NewResolver returns [ErrCyclicDefinition], [ErrUnknownFunction], or
// [ErrSpecialAlgebra] if a definition cannot be resolved for that reason, and
// an error wrapping [syntax.ErrInvalidSyntax] or [syntax.ErrUnknownUnit] if a
// definition is not a valid expression over the other definitions.
func NewResolver(defs *Definitions, maxExponent int) (*Resolver, error) {
	r := &Resolver{
		maxExponent: maxExponent,
		prefixes:    defs.Prefixes,
		sensitive:   lookup{atoms: map[string]int{}, prefixes: map[string]int{}},
		insensitive: lookup{atoms: map[string]int{}, prefixes: map[string]int{}, fold: true},
	}
	r.sensitive.resolver, r.insensitive.resolver = r, r

	for i, p := range defs.Prefixes {
		r.sensitive.addPrefix(p.Code, i)
		r.insensitive.addPrefix(p.CICode, i)
	}
	for _, b := range defs.BaseUnits {
		r.addAtom(b.Code, b.CICode, true)
	}
	for _, u := range defs.Units {
		r.addAtom(u.Code, u.CICode, u.Metric)
	}

	resolution := &resolution{
		resolver: r,
		units:    defs.Units,
		states:   make([]state, len(r.atoms)),
	}
	for i := range r.atoms {
		if err := resolution.atom(i); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Resolve reduces a case-sensitive UCUM expression to canonical form.
//
// Resolve returns a [*syntax.Error] wrapping [syntax.ErrInvalidSyntax],
// [syntax.ErrUnknownUnit], or [ErrSpecialAlgebra] if code is not a valid unit.
func (r *Resolver) Resolve(code string) (Canonical, error) {
	return r.resolve(code, r.sensitive)
}

// ResolveInsensitive reduces a case-insensitive UCUM expression to canonical
// form. It reports errors as [Resolver.Resolve] does.
func (r *Resolver) ResolveInsensitive(code string) (Canonical, error) {
	return r.resolve(code, r.insensitive)
}

func (r *Resolver) resolve(code string, symbols lookup) (Canonical, error) {
	node, err := syntax.Parse(code, symbols, r.maxExponent)
	if err != nil {
		return Canonical{}, err
	}
	if sym, ok := node.(syntax.Symbol); ok && r.atoms[sym.Atom].canonical.Special != nil {
		return r.special(sym)
	}
	return r.evaluate(node)
}

// special returns the canonical form of a special unit standing alone, the
// only position in which UCUM permits one.
func (r *Resolver) special(sym syntax.Symbol) (Canonical, error) {
	if sym.HasExponent {
		return Canonical{}, r.algebraError(sym)
	}
	c := r.atoms[sym.Atom].canonical
	special := *c.Special
	if sym.HasPrefix {
		special.Scale = r.prefixes[sym.Prefix].Value
	}
	c.Special = &special
	return c, nil
}

func (r *Resolver) evaluate(n syntax.Node) (Canonical, error) {
	switch n := n.(type) {
	case syntax.Binary:
		left, err := r.evaluate(n.Left)
		if err != nil {
			return Canonical{}, err
		}
		right, err := r.evaluate(n.Right)
		if err != nil {
			return Canonical{}, err
		}
		if n.Op == syntax.Divide {
			return left.quo(right), nil
		}
		return left.mul(right), nil
	case syntax.Inverse:
		operand, err := r.evaluate(n.Operand)
		if err != nil {
			return Canonical{}, err
		}
		return unity().quo(operand), nil
	case syntax.Group:
		return r.evaluate(n.Term)
	case syntax.Factor:
		return Canonical{Factor: new(big.Rat).SetInt(n.Value)}, nil
	case syntax.Unity:
		return unity(), nil
	default:
		return r.symbol(n.(syntax.Symbol))
	}
}

func (r *Resolver) symbol(sym syntax.Symbol) (Canonical, error) {
	c := r.atoms[sym.Atom].canonical
	if c.Special != nil {
		return Canonical{}, r.algebraError(sym)
	}
	if sym.HasPrefix {
		c = c.mul(Canonical{Factor: r.prefixes[sym.Prefix].Value})
	}
	if sym.HasExponent {
		c = c.pow(sym.Exponent)
	}
	return c, nil
}

func (r *Resolver) algebraError(sym syntax.Symbol) error {
	return &syntax.Error{
		Offset: sym.Offset,
		Err:    fmt.Errorf("%w: %q", ErrSpecialAlgebra, r.atoms[sym.Atom].code),
	}
}

func (r *Resolver) addAtom(code, ciCode string, metric bool) {
	i := len(r.atoms)
	r.atoms = append(r.atoms, atom{code: code, metric: metric})
	r.sensitive.addAtom(code, i)
	r.insensitive.addAtom(ciCode, i)
}

// lookup resolves the symbols of one form of UCUM. Case-insensitive symbols
// are stored and queried in upper case.
type lookup struct {
	resolver *Resolver
	atoms    map[string]int
	prefixes map[string]int
	fold     bool
}

func (l lookup) Atom(code string) (int, bool, bool) {
	i, ok := l.atoms[l.key(code)]
	if !ok {
		return 0, false, false
	}
	return i, l.resolver.atoms[i].metric, true
}

func (l lookup) Prefix(code string) (int, bool) {
	i, ok := l.prefixes[l.key(code)]
	return i, ok
}

// addAtom records an atom unless its code is taken. Synonyms such as "l" and
// "L" share a case-insensitive code, in which case the first one is kept.
func (l lookup) addAtom(code string, i int) {
	if _, ok := l.atoms[l.key(code)]; !ok {
		l.atoms[l.key(code)] = i
	}
}

func (l lookup) addPrefix(code string, i int) {
	l.prefixes[l.key(code)] = i
}

func (l lookup) key(code string) string {
	if l.fold {
		return strings.ToUpper(code)
	}
	return code
}

var _ syntax.Lookup = lookup{}

type state int

const (
	unresolved state = iota
	resolving
	resolved
)

// resolution resolves the definitions of a [Resolver] in dependency order,
// detecting cycles among them.
type resolution struct {
	resolver *Resolver
	units    []Unit
	states   []state
}

func (rs *resolution) atom(i int) error {
	switch rs.states[i] {
	case resolved:
		return nil
	case resolving:
		return fmt.Errorf("%w: unit %q", ErrCyclicDefinition, rs.resolver.atoms[i].code)
	}
	rs.states[i] = resolving

	c := rs.baseUnit(i)
	if base := len(rs.resolver.atoms) - len(rs.units); i >= base {
		var err error
		if c, err = rs.unit(&rs.units[i-base]); err != nil {
			return err
		}
	}
	rs.resolver.atoms[i].canonical = c
	rs.states[i] = resolved
	return nil
}

// baseUnit returns the canonical form of atom i if it is a base unit, and
// unity otherwise.
func (rs *resolution) baseUnit(i int) Canonical {
	c := unity()
	for slot, code := range BaseUnitCodes {
		if code == rs.resolver.atoms[i].code {
			c.Dimension[slot] = 1
		}
	}
	return c
}

func (rs *resolution) unit(u *Unit) (Canonical, error) {
	if u.Special {
		return rs.specialUnit(u)
	}
	term, err := rs.term(u.Code, u.Term)
	if err != nil {
		return Canonical{}, err
	}
	c := term.mul(Canonical{Factor: u.Value})
	if u.Arbitrary && len(c.Arbitrary) == 0 {
		c.Arbitrary = map[string]int{u.Code: 1}
	}
	return c, nil
}

func (rs *resolution) specialUnit(u *Unit) (Canonical, error) {
	conversion, ok := LookupConversion(u.Function.Name)
	if !ok {
		return Canonical{}, fmt.Errorf("%w: unit %q uses %q", ErrUnknownFunction, u.Code, u.Function.Name)
	}
	proper, err := rs.term(u.Code, u.Function.Term)
	if err != nil {
		return Canonical{}, err
	}
	return Canonical{
		Dimension: proper.Dimension,
		Arbitrary: proper.Arbitrary,
		Special: &Special{
			Name:       u.Function.Name,
			Conversion: conversion,
			Scale:      big.NewRat(1, 1),
			Proper:     new(big.Rat).Mul(u.Function.Value, proper.Factor),
		},
	}, nil
}

// term resolves the expression a unit is defined by, first resolving every
// unit the expression refers to.
func (rs *resolution) term(code, term string) (Canonical, error) {
	r := rs.resolver
	node, err := syntax.Parse(term, r.sensitive, math.MaxInt)
	if err != nil {
		return Canonical{}, fmt.Errorf("unit %q: %w", code, err)
	}
	if err := rs.dependencies(node); err != nil {
		return Canonical{}, err
	}
	c, err := r.evaluate(node)
	if err != nil {
		return Canonical{}, fmt.Errorf("unit %q: %w", code, err)
	}
	return c, nil
}

func (rs *resolution) dependencies(n syntax.Node) error {
	switch n := n.(type) {
	case syntax.Binary:
		if err := rs.dependencies(n.Left); err != nil {
			return err
		}
		return rs.dependencies(n.Right)
	case syntax.Inverse:
		return rs.dependencies(n.Operand)
	case syntax.Group:
		return rs.dependencies(n.Term)
	case syntax.Symbol:
		return rs.atom(n.Atom)
	}
	return nil
}

func unity() Canonical {
	return Canonical{Factor: big.NewRat(1, 1)}
}

func (c Canonical) mul(other Canonical) Canonical {
	return c.combine(other, 1)
}

func (c Canonical) quo(other Canonical) Canonical {
	return c.combine(other, -1)
}

// combine returns c × other**sign for a sign of 1 or -1.
func (c Canonical) combine(other Canonical, sign int) Canonical {
	result := Canonical{Dimension: c.Dimension}
	if sign > 0 {
		result.Factor = new(big.Rat).Mul(c.Factor, other.Factor)
	} else {
		result.Factor = new(big.Rat).Quo(c.Factor, other.Factor)
	}
	for i, e := range other.Dimension {
		result.Dimension[i] += sign * e
	}
	result.Arbitrary = maps.Clone(c.Arbitrary)
	for code, e := range other.Arbitrary {
		if result.Arbitrary == nil {
			result.Arbitrary = map[string]int{}
		}
		result.Arbitrary[code] += sign * e
		if result.Arbitrary[code] == 0 {
			delete(result.Arbitrary, code)
		}
	}
	return result
}

func (c Canonical) pow(exponent int) Canonical {
	if exponent == 0 {
		return unity()
	}
	result := Canonical{Factor: c.ratPow(exponent)}
	for i, e := range c.Dimension {
		result.Dimension[i] = e * exponent
	}
	for code, e := range c.Arbitrary {
		if result.Arbitrary == nil {
			result.Arbitrary = map[string]int{}
		}
		result.Arbitrary[code] = e * exponent
	}
	return result
}

func (c Canonical) ratPow(exponent int) *big.Rat {
	n := big.NewInt(int64(exponent))
	n.Abs(n)
	num := new(big.Int).Exp(c.Factor.Num(), n, nil)
	den := new(big.Int).Exp(c.Factor.Denom(), n, nil)
	if exponent < 0 {
		num, den = den, num
	}
	return new(big.Rat).SetFrac(num, den)
}
