package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	APIKey                string
	GoogleSheetID         string
	GoogleCredentialsFile string
	Port                  string
	SheetRange            string
	Location              *time.Location
}

func Load() (Config, error) {
	apiKey := strings.TrimSpace(os.Getenv("API_KEY"))
	if apiKey == "" {
		return Config{}, fmt.Errorf("API_KEY is required")
	}

	sheetID := strings.TrimSpace(os.Getenv("GOOGLE_SHEET_ID"))
	if sheetID == "" {
		return Config{}, fmt.Errorf("GOOGLE_SHEET_ID is required")
	}

	credentialsFile := strings.TrimSpace(os.Getenv("GOOGLE_CREDENTIALS_FILE"))
	if credentialsFile == "" {
		return Config{}, fmt.Errorf("GOOGLE_CREDENTIALS_FILE is required")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	timezone := strings.TrimSpace(os.Getenv("TIMEZONE"))
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Config{}, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	sheetRange := strings.TrimSpace(os.Getenv("SHEET_RANGE"))
	if sheetRange == "" {
		sheetRange = "Sheet1!A:G"
	}

	return Config{
		APIKey:                apiKey,
		GoogleSheetID:         sheetID,
		GoogleCredentialsFile: credentialsFile,
		Port:                  port,
		SheetRange:            sheetRange,
		Location:              location,
	}, nil
}
