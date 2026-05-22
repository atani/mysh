package i18n

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

var verbRe = regexp.MustCompile(`%[#+\- 0-9.]*[a-zA-Z%]`)

func formatVerbs(s string) []string {
	verbs := verbRe.FindAllString(s, -1)
	sort.Strings(verbs)
	return verbs
}

// TestLocaleKeyParity ensures the en and ja maps define exactly the same keys,
// so a newly added string is never localized in only one language.
func TestLocaleKeyParity(t *testing.T) {
	for key := range en {
		if _, ok := ja[key]; !ok {
			t.Errorf("key %q present in en but missing in ja", key)
		}
	}
	for key := range ja {
		if _, ok := en[key]; !ok {
			t.Errorf("key %q present in ja but missing in en", key)
		}
	}
}

// TestLocaleFormatVerbParity ensures the en and ja strings for each key use the
// same set of format verbs, so a translation can't drop or reorder a %s/%q/%v
// and crash at runtime.
func TestLocaleFormatVerbParity(t *testing.T) {
	for key, enVal := range en {
		jaVal, ok := ja[key]
		if !ok {
			continue // covered by TestLocaleKeyParity
		}
		enVerbs := formatVerbs(enVal)
		jaVerbs := formatVerbs(jaVal)
		if len(enVerbs) != len(jaVerbs) {
			t.Errorf("key %q: format verb count mismatch en=%v ja=%v", key, enVerbs, jaVerbs)
			continue
		}
		for i := range enVerbs {
			if enVerbs[i] != jaVerbs[i] {
				t.Errorf("key %q: format verb mismatch en=%v ja=%v", key, enVerbs, jaVerbs)
				break
			}
		}
	}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Setenv(%q): %v", key, err)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q): %v", key, err)
	}
}

func clearLocaleEnv(t *testing.T) map[string]string {
	t.Helper()
	envVars := []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"}
	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
		unsetEnv(t, v)
	}
	return saved
}

func restoreLocaleEnv(t *testing.T, saved map[string]string) {
	t.Helper()
	for k, v := range saved {
		if v != "" {
			setEnv(t, k, v)
		} else {
			unsetEnv(t, k)
		}
	}
}

func TestTFallbackToEnglish(t *testing.T) {
	saved := clearLocaleEnv(t)
	defer restoreLocaleEnv(t, saved)

	current = detect()

	got := T(DriverMenuTitle)
	if got != "Connection driver:" {
		t.Errorf("expected English default, got: %q", got)
	}
}

func TestTJapaneseLocale(t *testing.T) {
	saved := clearLocaleEnv(t)
	defer restoreLocaleEnv(t, saved)

	setEnv(t, "LANG", "ja_JP.UTF-8")
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
	saved := clearLocaleEnv(t)
	defer restoreLocaleEnv(t, saved)

	// LANGUAGE takes priority over LANG
	setEnv(t, "LANG", "en_US.UTF-8")
	setEnv(t, "LANGUAGE", "ja")
	loc := detect()
	if loc[DriverMenuTitle] != "接続ドライバ:" {
		t.Error("LANGUAGE should take priority over LANG")
	}
}
