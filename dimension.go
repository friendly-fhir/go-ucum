package ucum

import (
	"strconv"
	"strings"

	"github.com/friendly-fhir/go-ucum/internal/model"
)

// Dimension is the physical dimension of a unit, expressed as integer powers
// of the seven UCUM base units: meter, second, gram, radian, kelvin, coulomb,
// and candela.
//
// Two units are commensurable only if their dimensions are equal. Dimension
// values are comparable with ==, and the zero value is the dimension of a
// dimensionless quantity.
type Dimension struct {
	exponents [len(model.BaseUnitCodes)]int
}

// Multiply returns the dimension of the product of quantities with dimensions
// d and other.
func (d Dimension) Multiply(other Dimension) Dimension {
	for i, e := range other.exponents {
		d.exponents[i] += e
	}
	return d
}

// Divide returns the dimension of the quotient of a quantity with dimension d
// by a quantity with dimension other.
func (d Dimension) Divide(other Dimension) Dimension {
	for i, e := range other.exponents {
		d.exponents[i] -= e
	}
	return d
}

// Pow returns the dimension of a quantity with dimension d raised to the
// given integer power.
func (d Dimension) Pow(exponent int) Dimension {
	for i := range d.exponents {
		d.exponents[i] *= exponent
	}
	return d
}

// IsDimensionless reports whether d describes a pure number.
func (d Dimension) IsDimensionless() bool {
	return d == Dimension{}
}

// String returns d as a UCUM term over the base units, such as "m.s-2", or
// "1" for a dimensionless quantity. The result is valid UCUM syntax.
func (d Dimension) String() string {
	var sb strings.Builder
	for i, e := range d.exponents {
		if e == 0 {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(model.BaseUnitCodes[i])
		if e != 1 {
			sb.WriteString(strconv.Itoa(e))
		}
	}
	if sb.Len() == 0 {
		return "1"
	}
	return sb.String()
}
