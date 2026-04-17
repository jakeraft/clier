package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoRawUserFacingErrors enforces the central-presenter contract
// project-wide: every error path must either return a *domain.Fault or
// wrap an underlying error with fmt.Errorf("...: %w", err). Building a
// raw fmt.Errorf string outside the message catalog silently downgrades
// to KindInternal at the presenter, which hides the real failure from
// users.
//
// errors.New is permitted only at package level (sentinel vars).
//
// Update this test only when adding a new permitted pattern; do not
// individually exempt files. If a real exception is needed, refactor
// the call into a helper documented as such.
func TestNoRawUserFacingErrors(t *testing.T) {
	roots := []string{"cmd", "internal"}
	fset := token.NewFileSet()

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				pos := fset.Position(call.Pos())

				switch {
				case pkg.Name == "fmt" && sel.Sel.Name == "Errorf":
					if !errorfIsWrapOnly(call) {
						t.Errorf("%s:%d: raw fmt.Errorf — return *domain.Fault or use %%w wrap",
							pos.Filename, pos.Line)
					}
				case pkg.Name == "errors" && sel.Sel.Name == "New":
					if !isPackageLevelCall(f, call) {
						t.Errorf("%s:%d: errors.New inside function — return *domain.Fault",
							pos.Filename, pos.Line)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

func errorfIsWrapOnly(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	return strings.Contains(lit.Value, "%w")
}

func isPackageLevelCall(file *ast.File, call *ast.CallExpr) bool {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for _, v := range vs.Values {
				if containsCall(v, call) {
					return true
				}
			}
		}
	}
	return false
}

func containsCall(expr, target ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if n == target {
			found = true
			return false
		}
		return true
	})
	return found
}
