package model

import (
	"fmt"
	"math/big"
)

// BaseUnitCodes are the case-sensitive codes of the base units a dimension
// vector is expressed in, in vector order. Definitions declaring any other base
// unit cannot be represented.
var BaseUnitCodes = [...]string{"m", "s", "g", "rad", "K", "C", "cd"}

// Definitions is one release of the UCUM tables.
type Definitions struct {
	Version      string
	RevisionDate string
	Prefixes     []Prefix
	BaseUnits    []BaseUnit
	Units        []Unit
}

// Prefix is a scaling prefix such as "k" (kilo) that may precede a metric
// unit atom.
type Prefix struct {
	Code   string
	CICode string
	Value  *big.Rat
}

// BaseUnit is one of the units every other unit is ultimately defined in.
type BaseUnit struct {
	Code   string
	CICode string
}

// Unit is a defined unit atom.
//
// A proper unit is Value times the unit expressed by the case-sensitive UCUM
// term Term. A special unit has no such linear definition; it is instead
// related to a proper unit through Function.
type Unit struct {
	Code      string
	CICode    string
	Metric    bool
	Special   bool
	Arbitrary bool
	Value     *big.Rat
	Term      string
	Function  *Function
}

// Function relates a special unit to the proper unit Value times Term through
// the named pair of conversion functions.
type Function struct {
	Name  string
	Value *big.Rat
	Term  string
}

// MustRat parses an exact rational such as "254/100" or "1e-3".
// It panics if s is not a valid rational; it exists for definitions that are
// known to be valid, such as generated code.
func MustRat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic(fmt.Sprintf("model: invalid rational %q", s))
	}
	return r
}
