package ucum

import (
	"io"

	"github.com/friendly-fhir/go-ucum/internal/model"
	"github.com/friendly-fhir/go-ucum/internal/xmldef"
)

// Registry is one release of the UCUM definitions: the prefixes, base units,
// and unit atoms that unit expressions are interpreted against. A Registry is
// immutable and may be shared by any number of engines.
type Registry struct {
	defs *model.Definitions
}

// LoadXML reads a registry from a ucum-essence XML document, the format in
// which the UCUM organization publishes its definitions.
//
// LoadXML returns [ErrUnsupportedBase] if the document declares a base unit
// other than the seven UCUM base units, and [ErrInvalidDefinition] if the
// document is malformed or any definition in it is incomplete. Whether the
// definitions are consistent with one another is not checked until they are
// used.
func LoadXML(r io.Reader) (*Registry, error) {
	defs, err := xmldef.Decode(r)
	if err != nil {
		return nil, err
	}
	return &Registry{defs: defs}, nil
}

// Version returns the UCUM release the registry was published as, such as
// "2.2", or the empty string if the source did not state one.
func (r *Registry) Version() string {
	return r.defs.Version
}
