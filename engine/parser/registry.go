package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

type LanguageSpec struct {
	Language   Language
	Name       string
	Extensions []string
}

var registry = []LanguageSpec{
	{Language: LanguageJavaScript, Name: "javascript", Extensions: []string{".js", ".jsx", ".ts"}},
	{Language: LanguagePython, Name: "python", Extensions: []string{".py"}},
}

func DetectLanguage(path string) (LanguageSpec, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return LanguageSpec{}, fmt.Errorf("detect language: empty path")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return LanguageSpec{}, fmt.Errorf("detect language for %q: missing extension", path)
	}
	for _, spec := range registry {
		for _, candidate := range spec.Extensions {
			if ext == candidate {
				return cloneSpec(spec), nil
			}
		}
	}
	return LanguageSpec{}, fmt.Errorf("detect language for %q: unsupported extension %q", path, ext)
}

func SupportedLanguages() []LanguageSpec {
	out := make([]LanguageSpec, 0, len(registry))
	for _, spec := range registry {
		out = append(out, cloneSpec(spec))
	}
	return out
}

func NewParserForPath(path string) (*Parser, LanguageSpec, error) {
	spec, err := DetectLanguage(path)
	if err != nil {
		return nil, LanguageSpec{}, err
	}
	p, err := NewParser(spec.Language)
	if err != nil {
		return nil, LanguageSpec{}, fmt.Errorf("create parser for %q: %w", path, err)
	}
	return p, spec, nil
}

func cloneSpec(spec LanguageSpec) LanguageSpec {
	return LanguageSpec{Language: spec.Language, Name: spec.Name, Extensions: append([]string(nil), spec.Extensions...)}
}
