// Package syntax parses UCUM unit expressions as defined by the grammar in
// section 10 of the UCUM specification.
//
// Parsing needs to know which unit atoms and prefixes exist, because the
// boundary between a prefix and an atom is not marked in the text: "cd" is
// the candela, while "cm" is centi- applied to the meter. The parser
// therefore consults a [Lookup] but leaves the meaning of every symbol to its
// caller, so that the same parser serves both the case-sensitive and the
// case-insensitive forms of UCUM, and any set of definitions.
package syntax
