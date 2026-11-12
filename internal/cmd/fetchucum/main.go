// Command fetchucum downloads the ucum-essence XML document of a tagged UCUM
// release from the UCUM organization's repository.
//
// It exists so that the definitions compiled into package ucum come from a
// known, pinned release, and so that moving to a newer release is a matter of
// changing the tag passed to it and regenerating:
//
//	go run ./internal/cmd/fetchucum -tag v2.2 -out internal/data/ucum-essence.xml
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const source = "https://raw.githubusercontent.com/ucum-org/ucum/%s/ucum-essence.xml"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fetchucum:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("fetchucum", flag.ContinueOnError)
	tag := flags.String("tag", "", "release tag of the UCUM repository, such as v2.2")
	out := flags.String("out", "", "path to write the document to")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *tag == "" || *out == "" {
		return fmt.Errorf("both -tag and -out are required")
	}
	return fetch(fmt.Sprintf(source, *tag), *out)
}

// fetch downloads url to path, replacing path only once the download has
// completed so that a failed download never leaves a truncated document.
func fetch(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: %s", url, resp.Status)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("get %s: %w", url, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
