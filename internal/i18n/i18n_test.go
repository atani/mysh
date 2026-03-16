package i18n

import (
	"os"
	"testing"
)

func TestTFallbackToEnglish(t *testing.T) {
	// Save and clear locale env vars
	envVars := []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"}
	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	current = detect()

	got := T(DriverMenuTitle)
	if got != "Connection driver:" {
		t.Errorf("expected English default, got: %q", got)
	}
}

func TestTJapaneseLocale(t *testing.T) {
	envVars := []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"}
	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	os.Setenv("LANG", "ja_JP.UTF-8")
	current = detect()

	got := T(DriverMenuTitle)
	if got != "接続ドライバ:" {
		t.Errorf("expected Japanese, got: %q", got)
	}
}

func TestTUnknownKey(t *testing.T) {
	got := T("nonexistent_key")
	if got != "nonexistent_key" {
		t.Errorf("expected key echoed back, got: %q", got)
	}
}

func TestDetectPriority(t *testing.T) {
	envVars := []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"}
	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	// LANGUAGE takes priority over LANG
	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("LANGUAGE", "ja")
	loc := detect()
	if loc[DriverMenuTitle] != "接続ドライバ:" {
		t.Error("LANGUAGE should take priority over LANG")
	}
}
