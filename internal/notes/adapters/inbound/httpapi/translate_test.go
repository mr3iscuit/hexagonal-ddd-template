package httpapi

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
)

// A core error with no row becomes a 500, so a missing row must fail a test.
func TestEveryCoreErrorHasARowInBusinessErrors(t *testing.T) {
	table, err := os.ReadFile("translate.go")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for layer, path := range map[string]string{"domain": "../../../domain/errors.go", "application": "../../../application/errors.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			declaration, ok := node.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for _, name := range declaration.Names {
				if !name.IsExported() {
					continue
				}
				found++
				if !strings.Contains(string(table), layer+"."+name.Name+",") {
					t.Errorf("%s.%s has no row in businessErrors", layer, name.Name)
				}
			}
			return true
		})
	}
	if found == 0 || found != len(businessErrors) {
		t.Errorf("found %d core errors, but businessErrors has %d rows", found, len(businessErrors))
	}
}

func TestAnAPIErrorReadsAsItsDetail(t *testing.T) {
	if got := NewAPIError(http.StatusTeapot, "CUSTOM_CODE", "custom detail").Error(); got != "custom detail" {
		t.Errorf("got %q", got)
	}
}

func TestEveryBusinessErrorHasADistinctCode(t *testing.T) {
	seen := map[string]bool{}
	for _, business := range businessErrors {
		if seen[business.code] {
			t.Errorf("code %s is used twice", business.code)
		}
		seen[business.code] = true
	}
}

func TestTranslateReportsTheFixedMessageOfAWrappedError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, business := range businessErrors {
		got := translate(context.Background(), logger, fmt.Errorf("private client input: %w", business.target))
		if got.Status != business.status || got.Code != business.code || got.Category != categoryBusiness || got.Detail != business.target.Error() {
			t.Errorf("%v became %+v", business.target, got)
		}
	}
}
