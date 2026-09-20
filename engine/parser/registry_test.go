package parser

import "testing"

func TestDetectLanguageSupportedJavaScriptExtensions(t *testing.T) {
	for _, path := range []string{"app.js", "component.jsx", "route.TS", "nested/path/file.Js"} {
		spec, err := DetectLanguage(path)
		if err != nil {
			t.Fatalf("DetectLanguage(%q): %v", path, err)
		}
		if spec.Language != LanguageJavaScript || spec.Name != "javascript" {
			t.Fatalf("DetectLanguage(%q) = %+v", path, spec)
		}
	}
}

func TestDetectLanguageRejectsUnsupportedInputs(t *testing.T) {
	for _, path := range []string{"", "README", "app.py", "service.rb"} {
		if _, err := DetectLanguage(path); err == nil {
			t.Fatalf("DetectLanguage(%q) expected error", path)
		}
	}
}

func TestSupportedLanguagesReturnsDefensiveCopy(t *testing.T) {
	languages := SupportedLanguages()
	if len(languages) != 1 {
		t.Fatalf("expected one phase-1 supported language, got %d", len(languages))
	}
	languages[0].Extensions[0] = ".mutated"
	again := SupportedLanguages()
	if again[0].Extensions[0] == ".mutated" {
		t.Fatalf("SupportedLanguages must return a defensive copy")
	}
}

func TestNewParserForPathRejectsUnsupportedPath(t *testing.T) {
	if p, _, err := NewParserForPath("app.py"); err == nil {
		p.Close()
		t.Fatalf("expected unsupported path to fail until Python grammar is integrated")
	}
}
