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
	for _, path := range []string{"", "README", "service.rb"} {
		if _, err := DetectLanguage(path); err == nil {
			t.Fatalf("DetectLanguage(%q) expected error", path)
		}
	}
}

func TestDetectLanguageSupportsPython(t *testing.T) {
	spec, err := DetectLanguage("views.PY")
	if err != nil {
		t.Fatalf("DetectLanguage Python: %v", err)
	}
	if spec.Language != LanguagePython || spec.Name != "python" {
		t.Fatalf("expected python language spec, got %+v", spec)
	}
}

func TestSupportedLanguagesReturnsDefensiveCopy(t *testing.T) {
	languages := SupportedLanguages()
	if len(languages) != 2 {
		t.Fatalf("expected two supported languages, got %d", len(languages))
	}
	languages[0].Extensions[0] = ".mutated"
	again := SupportedLanguages()
	if again[0].Extensions[0] == ".mutated" {
		t.Fatalf("SupportedLanguages must return a defensive copy")
	}
}

func TestNewParserForPathRejectsUnsupportedPath(t *testing.T) {
	if p, _, err := NewParserForPath("app.rb"); err == nil {
		p.Close()
		t.Fatalf("expected unsupported path to fail")
	}
}

func TestNewParserForPathSupportsPython(t *testing.T) {
	p, spec, err := NewParserForPath("app.py")
	if err != nil {
		t.Fatalf("NewParserForPath Python: %v", err)
	}
	defer p.Close()
	if spec.Language != LanguagePython {
		t.Fatalf("expected python spec, got %+v", spec)
	}
}
