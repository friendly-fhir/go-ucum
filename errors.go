package ucum

import (
	"errors"
	"fmt"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/syntax"
	"github.com/friendly-fhir/go-ucum/internal/xmldef"
)

// Errors reported while loading definitions and building an [Engine].
var (
	// ErrInvalidDefinition reports a definitions document that is malformed or
	// contains an incomplete definition.
	ErrInvalidDefinition = xmldef.ErrInvalidDefinition

	// ErrUnsupportedBase reports definitions that declare a base unit other
	// than the seven UCUM base units.
	ErrUnsupportedBase = xmldef.ErrUnsupportedBase

	// ErrCyclicDefinition reports a unit whose definition depends on itself.
	ErrCyclicDefinition = model.ErrCyclicDefinition

	// ErrUnknownFunction reports a special unit whose conversion function is
	// not implemented by this package.
	ErrUnknownFunction = model.ErrUnknownFunction

	// ErrInvalidOption reports an [Option] given a value it cannot accept.
	ErrInvalidOption = errors.New("ucum: invalid option")
)

// Errors reported while parsing a unit.
var (
	// ErrInvalidSyntax reports an expression that does not follow the UCUM
	// grammar.
	ErrInvalidSyntax = syntax.ErrInvalidSyntax

	// ErrUnknownUnit reports a symbol that names no unit, or a prefix applied
	// to a unit that does not accept prefixes.
	ErrUnknownUnit = syntax.ErrUnknownUnit

	// ErrSpecialAlgebra reports a special unit, such as "Cel", used in a
	// product, quotient, or power, which UCUM does not permit.
	ErrSpecialAlgebra = model.ErrSpecialAlgebra
)

// Errors reported while converting a value.
var (
	// ErrIncompatible reports units that measure different kinds of quantity.
	ErrIncompatible = errors.New("ucum: incompatible units")

	// ErrArbitrary reports units that involve different arbitrary units, such
	// as "[iU]" and "[arb'U]", which UCUM forbids comparing.
	ErrArbitrary = errors.New("ucum: arbitrary units are not commensurable")

	// ErrForeignUnit reports a [Unit] that was not parsed by the [Engine] it
	// is used with.
	ErrForeignUnit = errors.New("ucum: unit belongs to another engine")

	// ErrDomain reports a value that a unit cannot express, such as the pH of
	// a non-positive concentration, or a value whose magnitude exceeds the
	// engine's limit.
	ErrDomain = model.ErrDomain
)

// ParseError describes where and why a unit expression could not be parsed.
type ParseError struct {
	// Input is the expression that was parsed.
	Input string

	// Offset is the byte offset in Input at which the problem was detected.
	Offset int

	// Err is the cause, which wraps one of [ErrInvalidSyntax],
	// [ErrUnknownUnit], or [ErrSpecialAlgebra].
	Err error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%v (in %q at offset %d)", e.Err, e.Input, e.Offset)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

var _ error = (*ParseError)(nil)
