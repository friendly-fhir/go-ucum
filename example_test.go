package ucum_test

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/shopspring/decimal"

	"github.com/friendly-fhir/go-ucum"
)

func Example() {
	reader, err := os.Open("testdata/essence.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	registry, err := ucum.LoadXML(reader)
	if err != nil {
		log.Fatal(err)
	}

	engine, err := ucum.New(registry)
	if err != nil {
		log.Fatal(err)
	}

	from, err := engine.ParseUnit("mg/dL")
	if err != nil {
		log.Fatal(err)
	}

	to, err := engine.ParseUnit("g/L")
	if err != nil {
		log.Fatal(err)
	}

	if !engine.Compatible(from, to) {
		log.Fatal(errors.New("incompatible units"))
	}

	result, err := engine.Convert(decimal.RequireFromString("100"), from, to)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result, to)
	// Output: 1 g/L
}
