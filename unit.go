package ucum

import (
	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum/internal/model"
)

// Unit is a parsed UCUM unit expression, obtained from [Engine.ParseUnit].
//
// A Unit is immutable and may be shared freely. It can only be used with the
// [Engine] that parsed it.
type Unit struct {
	code      string
	canonical model.Canonical
	engine    *Engine
}

// String returns the expression the unit was parsed from.
func (u *Unit) String() string {
	return u.code
}

// Dimension returns the physical dimension of the unit.
func (u *Unit) Dimension() Dimension {
	return Dimension{exponents: u.canonical.Dimension}
}

// IsSpecial reports whether the unit is a special unit, such as "Cel" or
// "[pH]", which is not a ratio scale and converts through a function rather
// than a factor.
func (u *Unit) IsSpecial() bool {
	return u.canonical.Special != nil
}

// Compatible reports whether values can be converted from u to other, as
// [Engine.Compatible] does for the engine that parsed u.
func (u *Unit) Compatible(other *Unit) bool {
	return u.engine.Compatible(u, other)
}

// Convert converts value, measured in u, to the unit to, as [Engine.Convert]
// does for the engine that parsed u. It returns the same errors, including
// [ErrForeignUnit] if to was parsed by another engine.
func (u *Unit) Convert(value decimal.Decimal, to *Unit) (decimal.Decimal, error) {
	return u.engine.Convert(value, u, to)
}

// Coarser reports whether u is a larger unit than other, such as "cm" compared
// to "mm", so that one u measures more than one other.
//
// Coarser reports false if the units are not [Unit.Compatible], or if either
// is a special unit, which has no single magnitude to compare.
func (u *Unit) Coarser(other *Unit) bool {
	if u.IsSpecial() || other.IsSpecial() || !u.Compatible(other) {
		return false
	}
	return u.canonical.Factor.Cmp(other.canonical.Factor) > 0
}

// IsArbitrary reports whether the unit involves an arbitrary unit, such as
// "[iU]", which is only commensurable with units involving the same arbitrary
// units.
func (u *Unit) IsArbitrary() bool {
	return len(u.canonical.Arbitrary) > 0
}
