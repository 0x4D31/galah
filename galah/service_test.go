package galah

import (
	"context"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewServiceWithDefaults(t *testing.T) {
	// Ensure working directory is project root so default paths exist.
	wd, _ := os.Getwd()
	os.Chdir("..")
	t.Cleanup(func() { os.Chdir(wd) })

	svc, err := NewService(context.Background(), Options{
		LLMProvider: "openai",
		LLMModel:    "gpt-3.5-turbo-1106",
		LLMAPIKey:   "dummy",
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	if svc == nil {
		t.Fatal("NewService returned nil service")
	}
	// Cleanup files created by defaults
	t.Cleanup(func() {
		svc.Cache.Close()
		os.Remove(DefaultCacheDBFile)
		os.Remove(DefaultEventLogFile)
	})
}
