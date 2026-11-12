package ucum

import (
	"errors"
	"fmt"
	"maps"
	"math/big"

	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/syntax"
)

// Engine parses and converts units against one [Registry]. An Engine is safe
// for concurrent use.
type Engine struct {
	resolver *model.Resolver
	limits   model.Limits
}

// New builds an engine from the definitions in registry.
//
// New returns [ErrInvalidOption] if an option is given an unacceptable value,
// and [ErrCyclicDefinition], [ErrUnknownFunction], or an error wrapping
// [ErrInvalidSyntax] or [ErrUnknownUnit] if the definitions in registry are
// not consistent with one another.
func New(registry *Registry, opts ...Option) (*Engine, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}
	resolver, err := model.NewResolver(registry.defs, cfg.maxExponent)
	if err != nil {
		return nil, err
	}
	return &Engine{
		resolver: resolver,
		limits: model.Limits{
			Digits:       cfg.precision,
			MaxMagnitude: cfg.maxMagnitude,
		},
	}, nil
}

// ParseUnit parses a case-sensitive UCUM unit expression, such as "mg/dL".
//
// ParseUnit returns a [*ParseError] wrapping [ErrInvalidSyntax] if code does
// not follow the UCUM grammar or uses an exponent beyond the engine's limit,
// [ErrUnknownUnit] if it names a unit that is not defined, and
// [ErrSpecialAlgebra] if it combines a special unit with other units.
func (e *Engine) ParseUnit(code string) (*Unit, error) {
	return e.parse(code, e.resolver.Resolve)
}

// ParseUnitInsensitive parses a case-insensitive UCUM unit expression, such
// as "MG/DL", for systems that cannot preserve case. Units it returns are
// interchangeable with those returned by [Engine.ParseUnit]. It reports errors
// as ParseUnit does.
func (e *Engine) ParseUnitInsensitive(code string) (*Unit, error) {
	return e.parse(code, e.resolver.ResolveInsensitive)
}

func (e *Engine) parse(code string, resolve func(string) (model.Canonical, error)) (*Unit, error) {
	canonical, err := resolve(code)
	if err != nil {
		var syntaxErr *syntax.Error
		errors.As(err, &syntaxErr)
		return nil, &ParseError{Input: code, Offset: syntaxErr.Offset, Err: syntaxErr.Err}
	}
	return &Unit{code: code, canonical: canonical, engine: e}, nil
}

// Compatible reports whether values can be converted between units from and
// to: both were parsed by e, they measure the same kind of quantity, and they
// involve the same arbitrary units.
func (e *Engine) Compatible(from, to *Unit) bool {
	return e.check(from, to) == nil
}

// Convert converts value from one unit to another.
//
// The result is exact when it can be written as a finite decimal and no
// special unit required a transcendental function; otherwise it is rounded to
// the engine's precision.
//
// Convert returns [ErrForeignUnit] if either unit was parsed by another
// engine, [ErrIncompatible] if the units measure different kinds of quantity,
// [ErrArbitrary] if they involve different arbitrary units, and [ErrDomain] if
// the value cannot be expressed in the target unit or exceeds the engine's
// magnitude limit.
func (e *Engine) Convert(value decimal.Decimal, from, to *Unit) (decimal.Decimal, error) {
	if err := e.check(from, to); err != nil {
		return decimal.Decimal{}, err
	}
	if err := e.checkMagnitude(value); err != nil {
		return decimal.Decimal{}, err
	}
	base, fromExact, err := e.toBase(value.Rat(), from.canonical)
	if err != nil {
		return decimal.Decimal{}, err
	}
	result, toExact, err := e.fromBase(base, to.canonical)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return e.decimal(result, fromExact && toExact)
}

func (e *Engine) check(from, to *Unit) error {
	switch {
	case from.engine != e || to.engine != e:
		return ErrForeignUnit
	case from.canonical.Dimension != to.canonical.Dimension:
		return fmt.Errorf("%w: %q is %v but %q is %v", ErrIncompatible, from, from.Dimension(), to, to.Dimension())
	case !maps.Equal(from.canonical.Arbitrary, to.canonical.Arbitrary):
		return fmt.Errorf("%w: %q and %q", ErrArbitrary, from, to)
	}
	return nil
}

func (e *Engine) checkMagnitude(value decimal.Decimal) error {
	if value.IsZero() {
		return nil
	}
	magnitude := int64(value.Exponent()) + int64(value.NumDigits()) - 1
	return e.checkPower(magnitude)
}

func (e *Engine) checkPower(magnitude int64) error {
	if magnitude > int64(e.limits.MaxMagnitude) || magnitude < -int64(e.limits.MaxMagnitude) {
		return fmt.Errorf("%w: 10^%d exceeds the magnitude limit", ErrDomain, magnitude)
	}
	return nil
}

// toBase converts a value in the unit c to a number of base units.
func (e *Engine) toBase(value *big.Rat, c model.Canonical) (*big.Rat, bool, error) {
	if c.Special == nil {
		return value.Mul(value, c.Factor), true, nil
	}
	scaled := new(big.Rat).Mul(value, c.Special.Scale)
	proper, exact, err := c.Special.Conversion.ToProper(scaled, e.limits)
	if err != nil {
		return nil, false, err
	}
	return proper.Mul(proper, c.Special.Proper), exact, nil
}

// fromBase converts a number of base units to a value in the unit c.
func (e *Engine) fromBase(base *big.Rat, c model.Canonical) (*big.Rat, bool, error) {
	if c.Special == nil {
		return base.Quo(base, c.Factor), true, nil
	}
	proper := new(big.Rat).Quo(base, c.Special.Proper)
	value, exact, err := c.Special.Conversion.FromProper(proper, e.limits)
	if err != nil {
		return nil, false, err
	}
	return value.Quo(value, c.Special.Scale), exact, nil
}

// decimal returns r as a decimal: exactly if it is exact and terminates, and
// rounded to the engine's precision otherwise.
func (e *Engine) decimal(r *big.Rat, exact bool) (decimal.Decimal, error) {
	if r.Sign() == 0 {
		return decimal.Zero, nil
	}
	magnitude := e.floorLog10(r)
	if err := e.checkPower(int64(magnitude)); err != nil {
		return decimal.Decimal{}, err
	}
	if places, ok := e.terminatingPlaces(r); exact && ok {
		return decimal.NewFromBigRat(r, places), nil
	}
	return decimal.NewFromBigRat(r, e.limits.Digits-1-int32(magnitude)), nil
}

// floorLog10 returns the power of ten of the leading digit of r, which must
// not be zero.
func (e *Engine) floorLog10(r *big.Rat) int {
	abs := new(big.Rat).Abs(r)

	// The difference in digit counts is the exact power or one too high.
	k := len(abs.Num().String()) - len(abs.Denom().String())
	if abs.Cmp(e.pow10(k)) < 0 {
		k--
	}
	return k
}

func (*Engine) pow10(n int) *big.Rat {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(max(n, -n))), nil)
	if n < 0 {
		return new(big.Rat).SetFrac(big.NewInt(1), p)
	}
	return new(big.Rat).SetInt(p)
}

// terminatingPlaces reports whether r has a finite decimal expansion, and if
// so the number of places after the decimal point it needs.
func (*Engine) terminatingPlaces(r *big.Rat) (int32, bool) {
	den := new(big.Int).Set(r.Denom())
	var places [2]int32
	for i, prime := range []int64{2, 5} {
		p := big.NewInt(prime)
		remainder := new(big.Int)
		for {
			quotient, mod := new(big.Int).QuoRem(den, p, remainder)
			if mod.Sign() != 0 {
				break
			}
			den = quotient
			places[i]++
		}
	}
	return max(places[0], places[1]), den.Cmp(big.NewInt(1)) == 0
}
