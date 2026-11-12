package ucum

import "github.com/friendly-fhir/go-ucum/internal/model"

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

// IsArbitrary reports whether the unit involves an arbitrary unit, such as
// "[iU]", which is only commensurable with units involving the same arbitrary
// units.
func (u *Unit) IsArbitrary() bool {
	return len(u.canonical.Arbitrary) > 0
}
