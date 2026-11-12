package model

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// ErrDomain reports a value that a special unit cannot express, such as the
// logarithm of a non-positive number, or a result whose magnitude exceeds the
// configured limit.
var ErrDomain = errors.New("ucum: value outside the domain of the unit")

// Limits bounds the work a [Conversion] may perform.
type Limits struct {
	// Digits is the number of significant digits an inexact result must be
	// accurate to.
	Digits int32

	// MaxMagnitude is the largest power of ten, in either direction, that a
	// result may reach.
	MaxMagnitude int32
}

// Conversion is the pair of functions that converts between a special unit
// and its proper unit.
type Conversion struct {
	scale scale
}

// LookupConversion returns the conversion with the given function name from
// the UCUM tables, and whether this package implements it.
func LookupConversion(name string) (Conversion, bool) {
	s, ok := scales[name]
	return Conversion{scale: s}, ok
}

// ToProper converts x, a value on the special unit's scale, to the number of
// proper units it denotes. It reports whether the result is exact.
//
// ToProper returns [ErrDomain] if the result cannot be expressed within limits.
func (c Conversion) ToProper(x *big.Rat, limits Limits) (*big.Rat, bool, error) {
	return c.scale.toProper(x, limits)
}

// FromProper converts y, a number of proper units, to a value on the special
// unit's scale. It reports whether the result is exact.
//
// FromProper returns [ErrDomain] if y is outside the domain of the function.
func (c Conversion) FromProper(y *big.Rat, limits Limits) (*big.Rat, bool, error) {
	return c.scale.fromProper(y, limits)
}

// scale is the relation between one kind of special unit and its proper unit.
type scale interface {
	toProper(x *big.Rat, limits Limits) (*big.Rat, bool, error)
	fromProper(y *big.Rat, limits Limits) (*big.Rat, bool, error)
}

var scales = map[string]scale{
	"Cel":         affine{offset: big.NewRat(27315, 100)},
	"degF":        affine{offset: big.NewRat(45967, 100)},
	"degRe":       affine{offset: big.NewRat(21852, 100)},
	"tanTimes100": tangent{degrees: false},
	"100tan":      tangent{degrees: true},
	"hpX":         exponential{base: 10, scale: big.NewRat(-1, 1)},
	"hpC":         exponential{base: 100, scale: big.NewRat(-1, 1)},
	"hpM":         exponential{base: 1000, scale: big.NewRat(-1, 1)},
	"hpQ":         exponential{base: 50000, scale: big.NewRat(-1, 1)},
	"pH":          exponential{base: 10, scale: big.NewRat(-1, 1)},
	"ln":          exponential{base: 0, scale: big.NewRat(1, 1)},
	"lg":          exponential{base: 10, scale: big.NewRat(1, 1)},
	"lgTimes2":    exponential{base: 10, scale: big.NewRat(1, 2)},
	"ld":          exponential{base: 2, scale: big.NewRat(1, 1)},
	"sqrt":        square{},
}

// guardDigits is the extra working precision carried through intermediate
// steps so that rounding in them does not reach the requested digits.
const guardDigits = 10

// affine is a temperature scale offset from its proper unit: y = x + offset.
type affine struct {
	offset *big.Rat
}

func (a affine) toProper(x *big.Rat, _ Limits) (*big.Rat, bool, error) {
	return new(big.Rat).Add(x, a.offset), true, nil
}

func (a affine) fromProper(y *big.Rat, _ Limits) (*big.Rat, bool, error) {
	return new(big.Rat).Sub(y, a.offset), true, nil
}

// tangent is a slope expressed as a percentage, x = 100 tan(θ), where the
// proper unit measures θ in radians or in degrees.
type tangent struct {
	degrees bool
}

// pi is π to the precision given in the UCUM tables.
var pi = decimal.RequireFromString("3.1415926535897932384626433832795028841971693993751058209749445923")

var hundred = decimal.NewFromInt(100)

func (t tangent) toProper(x *big.Rat, limits Limits) (*big.Rat, bool, error) {
	places := limits.Digits + guardDigits
	theta := toDecimal(x, limits).DivRound(hundred, places).Atan()
	if t.degrees {
		theta = theta.Mul(decimal.NewFromInt(180)).DivRound(pi, places)
	}
	return theta.Rat(), false, nil
}

func (t tangent) fromProper(y *big.Rat, limits Limits) (*big.Rat, bool, error) {
	theta := toDecimal(y, limits)
	if t.degrees {
		theta = theta.Mul(pi).DivRound(decimal.NewFromInt(180), limits.Digits+guardDigits)
	}
	return theta.Tan().Mul(hundred).Rat(), false, nil
}

// exponential is a logarithmic scale: y = base**(scale × x), where a base of
// zero denotes Euler's number.
type exponential struct {
	base  int64
	scale *big.Rat
}

func (e exponential) toProper(x *big.Rat, limits Limits) (*big.Rat, bool, error) {
	places := limits.Digits + guardDigits
	power := toDecimal(new(big.Rat).Mul(e.scale, x), limits)
	magnitude := power.Mul(e.log10Base(places)).Round(places)

	// 10**magnitude is computed as 10**fraction shifted by the whole part, so
	// that large magnitudes cost no more than small ones.
	whole := magnitude.Floor()
	if whole.Abs().GreaterThan(decimal.NewFromInt32(limits.MaxMagnitude)) {
		return nil, false, fmt.Errorf("%w: 10^%v exceeds the magnitude limit", ErrDomain, whole)
	}
	fraction, _ := magnitude.Sub(whole).Mul(e.ln10(places)).ExpTaylor(places)
	return fraction.Shift(int32(whole.IntPart())).Rat(), false, nil
}

func (e exponential) fromProper(y *big.Rat, limits Limits) (*big.Rat, bool, error) {
	if y.Sign() <= 0 {
		return nil, false, fmt.Errorf("%w: logarithm of %v", ErrDomain, y.RatString())
	}
	places := limits.Digits + guardDigits

	// Ln fails only for non-positive values, which are rejected above.
	ln, _ := toDecimal(y, limits).Ln(places)
	power := ln.DivRound(e.ln10(places), places).DivRound(e.log10Base(places), places)
	return new(big.Rat).Quo(power.Rat(), e.scale), false, nil
}

// log10Base returns the base-10 logarithm of the base.
func (e exponential) log10Base(places int32) decimal.Decimal {
	switch e.base {
	case 0:
		return decimal.NewFromInt(1).DivRound(e.ln10(places), places)
	case 10:
		return decimal.NewFromInt(1)
	}
	ln, _ := decimal.NewFromInt(e.base).Ln(places)
	return ln.DivRound(e.ln10(places), places)
}

func (exponential) ln10(places int32) decimal.Decimal {
	ln, _ := decimal.NewFromInt(10).Ln(places)
	return ln
}

// square relates a unit to the square of its proper unit: y = x².
type square struct{}

func (square) toProper(x *big.Rat, _ Limits) (*big.Rat, bool, error) {
	return new(big.Rat).Mul(x, x), true, nil
}

func (square) fromProper(y *big.Rat, limits Limits) (*big.Rat, bool, error) {
	if y.Sign() < 0 {
		return nil, false, fmt.Errorf("%w: square root of %v", ErrDomain, y.RatString())
	}
	bits := uint(limits.Digits+guardDigits) * 4
	root := new(big.Float).SetPrec(bits).SetRat(y)
	root.Sqrt(root)
	x, _ := root.Rat(nil)
	return x, false, nil
}

var (
	_ scale = affine{}
	_ scale = tangent{}
	_ scale = exponential{}
	_ scale = square{}
)

// toDecimal returns r rounded to the requested significant digits plus guard
// digits.
func toDecimal(r *big.Rat, limits Limits) decimal.Decimal {
	return decimal.NewFromBigRat(r, limits.Digits+guardDigits-int32(magnitude(r)))
}

// magnitude returns the power of ten of r's leading digit, within one.
func magnitude(r *big.Rat) int {
	if r.Sign() == 0 {
		return 0
	}
	return len(new(big.Int).Abs(r.Num()).String()) - len(r.Denom().String())
}
