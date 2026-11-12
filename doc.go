// Package ucum implements the Unified Code for Units of Measure (UCUM), the
// standard used by healthcare, laboratory, and scientific systems to exchange
// units such as "mg/dL" or "mm[Hg]" unambiguously between computers.
//
// UCUM unit codes are expressions rather than fixed names: any combination of
// prefixes, unit atoms, exponents, and factors that follows the grammar is a
// valid unit, and its meaning is derived from a table of definitions published
// by the UCUM organization as the ucum-essence XML document. Definitions are
// loaded into a [Registry] with [LoadXML], so that newer releases of the
// tables can be used without a newer release of this package.
package ucum
