package mongodb_test

import (
	"testing"
	"time"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/mongodb"
)

func TestOptionsFromURIValid(t *testing.T) {
	uri := "mongodb://localhost:27017"
	timeout := 2 * time.Second

	opts, err := mongodb.OptionsFromURI(uri, timeout)
	if err != nil {
		t.Fatalf("OptionsFromURI() error = %v", err)
	}

	if opts.GetURI() != uri {
		t.Errorf("GetURI() = %q, want %q", opts.GetURI(), uri)
	}
	if opts.ConnectTimeout == nil || *opts.ConnectTimeout != timeout {
		t.Errorf("ConnectTimeout = %v, want %v", opts.ConnectTimeout, timeout)
	}
	if opts.ServerSelectionTimeout == nil || *opts.ServerSelectionTimeout != timeout {
		t.Errorf("ServerSelectionTimeout = %v, want %v", opts.ServerSelectionTimeout, timeout)
	}
}

func TestOptionsFromURIInvalid(t *testing.T) {
	_, err := mongodb.OptionsFromURI("not a valid uri", time.Second)
	if err == nil {
		t.Fatal("OptionsFromURI() expected error for invalid uri")
	}
}
