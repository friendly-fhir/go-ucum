package ucum_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/friendly-fhir/go-ucum"
)

func TestParseErrorError_UnknownUnit_DescribesInputAndOffset(t *testing.T) {
	t.Parallel()

	// Arrange
	sut := &ucum.ParseError{Input: "mg/furlong", Offset: 3, Err: ucum.ErrUnknownUnit}

	// Act
	message := sut.Error()

	// Assert
	if got, want := strings.Contains(message, `"mg/furlong"`), true; !cmp.Equal(got, want) {
		t.Errorf("ParseError.Error() = %q, want it to contain the input", message)
	}
	if got, want := strings.Contains(message, "offset 3"), true; !cmp.Equal(got, want) {
		t.Errorf("ParseError.Error() = %q, want it to contain the offset", message)
	}
	if got, want := strings.Contains(message, ucum.ErrUnknownUnit.Error()), true; !cmp.Equal(got, want) {
		t.Errorf("ParseError.Error() = %q, want it to contain the cause", message)
	}
}
