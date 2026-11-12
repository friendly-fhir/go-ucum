// Package model is the semantic model of UCUM: the definitions published in
// the UCUM tables, and what those definitions mean.
//
// Definitions are held in a form that is independent of how they were
// obtained, so that definitions decoded from XML at run time and definitions
// compiled into the binary are interpreted identically. Magnitudes are exact
// rationals, so that definitions can be combined without accumulating rounding
// error.
//
// Special units, such as degree Celsius or pH, are not ratio scales and
// cannot be related to their proper units by a factor. Their definitions use
// a pair of conversion functions instead, which [LookupConversion] provides.
// Conversions report whether their results are exact, so that callers decide
// how and when to round.
package model
