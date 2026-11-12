package ucum_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/friendly-fhir/go-ucum"
)

func TestLoadXML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "malformed document",
			input:   `<root>`,
			wantErr: ucum.ErrInvalidDefinition,
		}, {
			name:    "base unit outside UCUM",
			input:   `<root><base-unit Code="mol" CODE="MOL" dim="N"/></root>`,
			wantErr: ucum.ErrUnsupportedBase,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			reader := strings.NewReader(tc.input)

			// Act
			registry, err := ucum.LoadXML(reader)

			// Assert
			if got, want := err, tc.wantErr; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("LoadXML() error = %v, want %v", got, want)
			}
			if got, want := registry, (*ucum.Registry)(nil); got != want {
				t.Errorf("LoadXML() = %v, want %v", got, want)
			}
		})
	}
}

func TestRegistryVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "declared version",
			input: `<root version="2.2"><base-unit Code="m" CODE="M" dim="L"/></root>`,
			want:  "2.2",
		}, {
			name:  "undeclared version",
			input: `<root><base-unit Code="m" CODE="M" dim="L"/></root>`,
			want:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := mustLoadXML(t, tc.input)

			// Act
			version := sut.Version()

			// Assert
			if got, want := version, tc.want; !cmp.Equal(got, want) {
				t.Errorf("Registry.Version() = %q, want %q", got, want)
			}
		})
	}
}

func mustLoadXML(t *testing.T, input string) *ucum.Registry {
	t.Helper()
	registry, err := ucum.LoadXML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadXML() error = %v", err)
	}
	return registry
}
