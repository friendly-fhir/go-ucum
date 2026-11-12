package test

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum"
)

const (
	suitePath = "testdata/UcumFunctionalTests.xml"
	suiteURL  = "https://raw.githubusercontent.com/FHIR/Ucum-java/10089515484cdfd291fa6d2b74b989f60d12fcb3/src/test/resources/UcumFunctionalTests.xml"
)

type suite struct {
	Validation []validationCase `xml:"validation>case"`
	Conversion []conversionCase `xml:"conversion>case"`
}

type validationCase struct {
	ID     string `xml:"id,attr"`
	Unit   string `xml:"unit,attr"`
	Valid  bool   `xml:"valid,attr"`
	Reason string `xml:"reason,attr"`
}

// wantErr returns the error expected from parsing the case: none for a valid
// unit, and any error for an invalid one, since the suite does not say which.
func (vc validationCase) wantErr() error {
	if vc.Valid {
		return nil
	}
	return cmpopts.AnyError
}

type conversionCase struct {
	ID      string `xml:"id,attr"`
	Value   string `xml:"value,attr"`
	Source  string `xml:"srcUnit,attr"`
	Target  string `xml:"dstUnit,attr"`
	Outcome string `xml:"outcome,attr"`
}

var (
	cases    suite
	casesErr error
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		cases, casesErr = loadSuite()
	}
	os.Exit(m.Run())
}

func loadSuite() (suite, error) {
	if err := download(); err != nil {
		return suite{}, err
	}
	file, err := os.Open(suitePath)
	if err != nil {
		return suite{}, err
	}
	defer file.Close()
	var s suite
	if err := xml.NewDecoder(file).Decode(&s); err != nil {
		return suite{}, fmt.Errorf("decode %s: %w", suitePath, err)
	}
	return s, nil
}

// download fetches the suite unless a previous run already stored it.
func download() error {
	if _, err := os.Stat(suitePath); err == nil {
		return nil
	}
	resp, err := http.Get(suiteURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: %s", suiteURL, resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(suitePath), 0o755); err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("get %s: %w", suiteURL, err)
	}
	return os.WriteFile(suitePath, body, 0o644)
}

func requireSuite(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("conformance suite skipped in short mode")
	}
	if casesErr != nil {
		t.Skipf("conformance suite unavailable: %v", casesErr)
	}
}

// deviations are suite cases that contradict the UCUM specification itself,
// keyed by case identifier.
var deviations = map[string]string{
	"1-117": "UCUM section 10 permits an annotation only after a unit symbol or on its own, never after a factor",
}

func TestValidation(t *testing.T) {
	requireSuite(t)
	t.Parallel()

	for _, tc := range cases.Validation {
		t.Run(tc.ID+" "+tc.Unit, func(t *testing.T) {
			t.Parallel()
			if reason, ok := deviations[tc.ID]; ok {
				t.Skip(reason)
			}

			// Arrange
			sut := ucum.DefaultEngine

			// Act
			_, err := sut.ParseUnit(tc.Unit)

			// Assert
			if got, want := err, tc.wantErr(); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Errorf("DefaultEngine.ParseUnit(%q) error = %v, want %v (%s)", tc.Unit, got, want, tc.Reason)
			}
		})
	}
}

func TestConversion(t *testing.T) {
	requireSuite(t)
	t.Parallel()

	for _, tc := range cases.Conversion {
		t.Run(tc.ID, func(t *testing.T) {
			t.Parallel()

			// Arrange
			sut := ucum.DefaultEngine
			value := decimal.RequireFromString(tc.Value)
			outcome := decimal.RequireFromString(tc.Outcome)
			from := mustParseUnit(t, sut, tc.Source)
			to := mustParseUnit(t, sut, tc.Target)

			// Act
			result, err := sut.Convert(value, from, to)
			rounded := result.Round(-outcome.Exponent())

			// Assert
			if got, want := err, error(nil); !cmp.Equal(got, want, cmpopts.EquateErrors()) {
				t.Fatalf("DefaultEngine.Convert(%v, %q, %q) error = %v, want %v", value, tc.Source, tc.Target, got, want)
			}
			if got, want := rounded, outcome; !got.Equal(want) {
				t.Errorf("DefaultEngine.Convert(%v, %q, %q) = %v, want %v", value, tc.Source, tc.Target, result, want)
			}
		})
	}
}

func mustParseUnit(t *testing.T, engine *ucum.Engine, code string) *ucum.Unit {
	t.Helper()
	unit, err := engine.ParseUnit(code)
	if err != nil {
		t.Fatalf("DefaultEngine.ParseUnit(%q) error = %v", code, err)
	}
	return unit
}
