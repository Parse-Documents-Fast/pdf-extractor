package redis_test

import (
	"testing"
	"time"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
)

func TestOptionsFromURIValid(t *testing.T) {
	timeout := 2 * time.Second

	opts, err := redis.OptionsFromURI("redis://localhost:6379/0", timeout)
	if err != nil {
		t.Fatalf("OptionsFromURI() error = %v", err)
	}

	if opts.Addr != "localhost:6379" {
		t.Errorf("Addr = %q, want localhost:6379", opts.Addr)
	}
	if opts.DB != 0 {
		t.Errorf("DB = %d, want 0", opts.DB)
	}
	if opts.DialTimeout != timeout {
		t.Errorf("DialTimeout = %v, want %v", opts.DialTimeout, timeout)
	}
	if opts.ReadTimeout != timeout {
		t.Errorf("ReadTimeout = %v, want %v", opts.ReadTimeout, timeout)
	}
	if opts.WriteTimeout != timeout {
		t.Errorf("WriteTimeout = %v, want %v", opts.WriteTimeout, timeout)
	}
}

func TestOptionsFromURIInvalid(t *testing.T) {
	_, err := redis.OptionsFromURI("http://localhost:6379", time.Second)
	if err == nil {
		t.Fatal("OptionsFromURI() expected error for invalid scheme")
	}
}
