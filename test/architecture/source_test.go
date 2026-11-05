package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

const modulePath = "github.com/mr3iscuit/hexagonal-ddd-template"

var (
	domainPattern          = regexp.MustCompile(`^internal/[a-z]+/domain/[^/]+$`)
	applicationPattern     = regexp.MustCompile(`^internal/[a-z]+/application/[^/]+$`)
	inboundAdapterPattern  = regexp.MustCompile(`^internal/[a-z]+/adapters/inbound/[a-z]+/[^/]+$`)
	outboundAdapterPattern = regexp.MustCompile(`^internal/[a-z]+/adapters/outbound/[a-z]+/[^/]+$`)
	compositionRootPattern = regexp.MustCompile(`^cmd/[a-z]+/[^/]+$`)
)

// sourceFile is one parsed Go file, with a path relative to the project root.
type sourceFile struct {
	path     string
	syntax   *ast.File
	location *token.FileSet
}

func (file sourceFile) position(node ast.Node) string {
	return file.path + ":" + strconv.Itoa(file.location.Position(node.Pos()).Line)
}

func (file sourceFile) name() string { return filepath.Base(file.path) }

func (file sourceFile) isTest() bool { return strings.HasSuffix(file.path, "_test.go") }

func (file sourceFile) imports() []string {
	paths := make([]string, 0, len(file.syntax.Imports))
	for _, declaration := range file.syntax.Imports {
		paths = append(paths, strings.Trim(declaration.Path.Value, `"`))
	}
	return paths
}

func isDomain(file sourceFile) bool { return domainPattern.MatchString(file.path) && !file.isTest() }

func isApplication(file sourceFile) bool {
	return applicationPattern.MatchString(file.path) && !file.isTest()
}

func isInboundAdapter(file sourceFile) bool {
	return inboundAdapterPattern.MatchString(file.path) && !file.isTest()
}

func isOutboundAdapter(file sourceFile) bool {
	return outboundAdapterPattern.MatchString(file.path) && !file.isTest()
}

func isCompositionRoot(file sourceFile) bool {
	return compositionRootPattern.MatchString(file.path) && !file.isTest()
}

func isProductionCode(file sourceFile) bool { return !file.isTest() }

func isAnyCode(sourceFile) bool { return true }

type project struct {
	root  string
	files []sourceFile
}

func (scanned project) filesIn(layer func(sourceFile) bool) []sourceFile {
	var matching []sourceFile
	for _, file := range scanned.files {
		if layer(file) {
			matching = append(matching, file)
		}
	}
	return matching
}

func loadProject(t *testing.T, root string) project {
	t.Helper()
	location := token.NewFileSet()
	scanned := project{root: root}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "testdata" && path != root {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		syntax, err := parser.ParseFile(location, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		scanned.files = append(scanned.files, sourceFile{path: filepath.ToSlash(relative), syntax: syntax, location: location})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return scanned
}

func glossaryText(scanned project) string {
	paths, err := filepath.Glob(filepath.Join(scanned.root, "docs", "domain", "*.md"))
	if err != nil {
		panic(err)
	}
	var text strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		text.Write(content)
	}
	return text.String()
}

// splitWords breaks a Go name into words and keeps acronyms together, so that
// APIError becomes API and Error.
func splitWords(name string) []string {
	var words []string
	characters := []rune(name)
	start := 0
	for position := 1; position < len(characters); position++ {
		upper := unicode.IsUpper(characters[position])
		afterLower := !unicode.IsUpper(characters[position-1])
		beforeLower := position+1 < len(characters) && unicode.IsLower(characters[position+1])
		if upper && (afterLower || beforeLower) {
			words = append(words, string(characters[start:position]))
			start = position
		}
	}
	return append(words, string(characters[start:]))
}

func snakeCase(name string) string {
	return strings.ToLower(strings.Join(splitWords(name), "_"))
}

func typeDeclarations(file sourceFile) []*ast.TypeSpec {
	var declarations []*ast.TypeSpec
	for _, declaration := range file.syntax.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			declarations = append(declarations, specification.(*ast.TypeSpec))
		}
	}
	return declarations
}

func functionDeclarations(file sourceFile) []*ast.FuncDecl {
	var declarations []*ast.FuncDecl
	for _, declaration := range file.syntax.Decls {
		if function, isFunction := declaration.(*ast.FuncDecl); isFunction {
			declarations = append(declarations, function)
		}
	}
	return declarations
}

// expressionName renders a type or call target, such as http.Request.
func expressionName(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return expressionName(typed.X) + "." + typed.Sel.Name
	case *ast.StarExpr:
		return expressionName(typed.X)
	case *ast.IndexExpr:
		return expressionName(typed.X)
	}
	return ""
}

func receiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	return expressionName(function.Recv.List[0].Type)
}

// declaredNames returns every name a file declares.
func declaredNames(file sourceFile) []*ast.Ident {
	var names []*ast.Ident
	add := func(expression ast.Expr) {
		if identifier, isIdentifier := expression.(*ast.Ident); isIdentifier {
			names = append(names, identifier)
		}
	}
	ast.Inspect(file.syntax, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			names = append(names, typed.Name)
		case *ast.TypeSpec:
			names = append(names, typed.Name)
		case *ast.Field:
			names = append(names, typed.Names...)
		case *ast.ValueSpec:
			names = append(names, typed.Names...)
		case *ast.AssignStmt:
			if typed.Tok == token.DEFINE {
				for _, assigned := range typed.Lhs {
					add(assigned)
				}
			}
		case *ast.RangeStmt:
			if typed.Tok == token.DEFINE {
				add(typed.Key)
				add(typed.Value)
			}
		}
		return true
	})
	return names
}
