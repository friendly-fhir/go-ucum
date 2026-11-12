package ucum

import "fmt"

// Option configures an [Engine].
type Option interface {
	apply(*config) error
}

type config struct {
	precision    int32
	maxExponent  int
	maxMagnitude int32
}

func defaultConfig() config {
	return config{
		precision:    34,
		maxExponent:  99,
		maxMagnitude: 1000,
	}
}

type option func(*config) error

func (o option) apply(c *config) error {
	return o(c)
}

var _ Option = option(nil)

// WithPrecision sets the number of significant digits to which conversion
// results are rounded when they cannot be represented exactly. The default is
// 34, the precision of IEEE 754 decimal128.
func WithPrecision(digits int32) Option {
	return option(func(c *config) error {
		if digits < 1 {
			return fmt.Errorf("%w: precision %d is not positive", ErrInvalidOption, digits)
		}
		c.precision = digits
		return nil
	})
}

// WithMaxExponent sets the largest exponent magnitude a unit expression may
// use, such as the 3 in "m3". Larger exponents are rejected as invalid
// syntax, which bounds the cost of parsing untrusted input. The default is 99.
func WithMaxExponent(n int) Option {
	return option(func(c *config) error {
		if n < 1 {
			return fmt.Errorf("%w: maximum exponent %d is not positive", ErrInvalidOption, n)
		}
		c.maxExponent = n
		return nil
	})
}

// WithMaxMagnitude sets the largest power of ten, in either direction, that a
// converted value may reach. Values beyond it are rejected with [ErrDomain],
// which bounds the cost of converting untrusted input. The default is 1000.
func WithMaxMagnitude(n int32) Option {
	return option(func(c *config) error {
		if n < 1 {
			return fmt.Errorf("%w: maximum magnitude %d is not positive", ErrInvalidOption, n)
		}
		c.maxMagnitude = n
		return nil
	})
}
