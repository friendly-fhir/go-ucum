package ucum

import "github.com/friendly-fhir/go-ucum/internal/xmldef"

// Errors reported while loading definitions. Use [errors.Is] to test for them.
var (
	// ErrInvalidDefinition reports a definitions document that is malformed or
	// internally inconsistent.
	ErrInvalidDefinition = xmldef.ErrInvalidDefinition

	// ErrUnsupportedBase reports definitions that declare a base unit other
	// than the seven UCUM base units.
	ErrUnsupportedBase = xmldef.ErrUnsupportedBase
)
