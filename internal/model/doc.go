// Package model holds UCUM definitions in a form that is independent of how
// they were obtained, so that definitions decoded from XML at run time and
// definitions compiled into the binary are interpreted identically.
//
// Magnitudes are kept as exact rationals so that definitions can be combined
// without accumulating rounding error.
package model
