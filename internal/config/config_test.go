package config

import (
	"testing"
	"time"
)

func TestLoadReadsRequiredEnvAndDefaults(t *testing.T) {
	t.Setenv("API_KEY", "secret")
	t.Setenv("GOOGLE_SHEET_ID", "sheet-id")
	t.Setenv("GOOGLE_CREDENTIALS_FILE", "/tmp/service-account.json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.APIKey != "secret" {
		t.Fatalf("APIKey = %q, want %q", cfg.APIKey, "secret")
	}
	if cfg.GoogleSheetID != "sheet-id" {
		t.Fatalf("GoogleSheetID = %q, want %q", cfg.GoogleSheetID, "sheet-id")
	}
	if cfg.GoogleCredentialsFile != "/tmp/service-account.json" {
		t.Fatalf("GoogleCredentialsFile = %q, want %q", cfg.GoogleCredentialsFile, "/tmp/service-account.json")
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.SheetRange != "Sheet1!A:G" {
		t.Fatalf("SheetRange = %q, want %q", cfg.SheetRange, "Sheet1!A:G")
	}
	if cfg.Location.String() != "Asia/Jakarta" {
		t.Fatalf("Location = %q, want %q", cfg.Location.String(), "Asia/Jakarta")
	}
}

func TestLoadReadsOptionalEnv(t *testing.T) {
	t.Setenv("API_KEY", "secret")
	t.Setenv("GOOGLE_SHEET_ID", "sheet-id")
	t.Setenv("GOOGLE_CREDENTIALS_FILE", "/tmp/service-account.json")
	t.Setenv("PORT", "9090")
	t.Setenv("TIMEZONE", "UTC")
	t.Setenv("SHEET_RANGE", "Expenses!A:I")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.SheetRange != "Expenses!A:I" {
		t.Fatalf("SheetRange = %q, want %q", cfg.SheetRange, "Expenses!A:I")
	}
	if cfg.Location != time.UTC {
		t.Fatalf("Location = %v, want UTC", cfg.Location)
	}
}

func TestLoadRequiresAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_SHEET_ID", "sheet-id")
	t.Setenv("GOOGLE_CREDENTIALS_FILE", "/tmp/service-account.json")

	_, err := Load()
	if err == nil {
		t.Fatal("Load returned nil error, want missing API_KEY error")
	}
}

func TestLoadRejectsInvalidTimezone(t *testing.T) {
	t.Setenv("API_KEY", "secret")
	t.Setenv("GOOGLE_SHEET_ID", "sheet-id")
	t.Setenv("GOOGLE_CREDENTIALS_FILE", "/tmp/service-account.json")
	t.Setenv("TIMEZONE", "not-a-timezone")

	_, err := Load()
	if err == nil {
		t.Fatal("Load returned nil error, want invalid timezone error")
	}
}
