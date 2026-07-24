package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoCodeReadsFieldsOffDefaultConfig is a structural guard, not a behavior
// test. Eleven shipped bugs came from one habit: give a setting a non-zero
// default in DefaultConfig, then decide "did the user set this?" by comparing
// the loaded value against that same default. Because Load unmarshals *over*
// DefaultConfig, the comparison is always false for anyone who typed the
// default value, so the setting silently did nothing — an explicit
// `version: "1.0.0"` got replaced by a git tag, a theme's page_size never
// reached a renderer, a language directory's book.yaml published English.
//
// The tell is always the same expression shape: a field selected off a fresh
// DefaultConfig(). Building a whole config with DefaultConfig() is fine;
// reaching into one of its fields is only ever done to answer a question
// IsSet (see set_keys.go) answers correctly. So this walks every Go file in
// the module and fails on `DefaultConfig().<field>`.
func TestNoCodeReadsFieldsOffDefaultConfig(t *testing.T) {
	moduleRoot := filepath.Join("..", "..")
	fset := token.NewFileSet()

	var offenders []string
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", "testdata":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // not this test's job to police syntax
		}

		// Method calls such as DefaultConfig().AllowRawHTML() are fine: they
		// ask the type a question instead of comparing a raw field value.
		called := map[ast.Expr]bool{}
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				called[call.Fun] = true
			}
			return true
		})

		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || called[ast.Expr(sel)] {
				return true
			}
			if !isDefaultConfigCall(sel.X) {
				return true
			}
			offenders = append(offenders, fset.Position(sel.Pos()).String()+": DefaultConfig()."+sel.Sel.Name)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk module: %v", err)
	}

	if len(offenders) > 0 {
		t.Errorf("code reads a field off DefaultConfig(), which cannot tell a configured value "+
			"from a default one — use cfg.IsSet(\"<dotted.key>\") instead:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// isDefaultConfigCall reports whether expr is a call to DefaultConfig(), in
// either its bare or package-qualified spelling.
func isDefaultConfigCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name == "DefaultConfig"
	case *ast.SelectorExpr:
		return fn.Sel.Name == "DefaultConfig"
	}
	return false
}
