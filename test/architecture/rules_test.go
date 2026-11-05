package architecture_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type rule struct {
	name    string
	check   func(project) []string
	planted []string
}

func rules() []rule {
	return []rule{
		{"domain imports only pure standard library packages", domainImportsArePure, []string{`imports "net/http"`}},
		{"application imports only the domain and the standard library", applicationImportsArePure, []string{`imports "database/sql"`}},
		{"adapters stay apart", adaptersStayApart, []string{"adapter imports another adapter"}},
		{"only the composition root imports outbound adapters", outboundAdaptersStayInTheRoot, []string{"only the composition root may import"}},
		{"the composition root never touches the domain", compositionRootAvoidsDomain, []string{"composition root imports the domain"}},
		{"imports are never dotted, blank or renamed", importsAreQualified, []string{"must not be dotted, blank or renamed"}},
		{"bounded contexts stay apart", contextsStayApart, []string{"imports another bounded context"}},
		{"the framework stays in the inbound adapter", frameworkStaysInTheInboundAdapter, []string{"framework used outside the inbound adapter"}},
		{"Chi's RequestID and Recoverer stay out", unsafeMiddlewareStaysOut, []string{"middleware.RequestID", "middleware.Recoverer"}},
		{"ports live in the application layer and end in Port", portsBelongToTheApplication, []string{"interface Clock", "interface WidgetRepository"}},
		{"repositories exist only for aggregate roots", repositoriesServeAggregates, []string{"GhostRepositoryPort has no aggregate root"}},
		{"domain events are past-tense facts", eventsArePastTense, []string{"WidgetChange must be a past-tense fact"}},
		{"domain state is private", domainStateIsPrivate, []string{"exported domain field Name"}},
		{"domain behaviour uses business verbs", behaviourUsesBusinessVerbs, []string{"SetName is not a business verb"}},
		{"aggregates reference each other only by ID", aggregatesReferenceByID, []string{"Gadget must reference Widget by ID"}},
		{"pointers are used only where something requires them", pointersHaveAReason, []string{"pointer without a reason: Widget", "pointer without a reason: &Gadget", "pointer without a reason: new(Widget)"}},
		{"types follow the naming rules of their layer", typesFollowLayerNames, []string{"WidgetEntity", "CreateWidgetCommandDTO", "widgetBody", "WidgetStore"}},
		{"every type has its own file", everyTypeHasItsOwnFile, []string{"Gadget2 belongs in gadget2.go"}},
		{"names are full words", namesAreFullWords, []string{"cfg is not a full word", "saveErr is not a full word", "i is not a full word"}},
		{"hard-coded numbers are constants", numbersAreConstants, []string{"42 must be a constant"}},
		{"hard-coded strings are constants", stringsAreConstants, []string{`"widget.change" must be a constant`}},
		{"structs are never converted directly", structsAreMappedExplicitly, []string{"direct struct conversion to WidgetResponseDTO"}},
		{"mappers only reshape data", mappersOnlyReshapeData, []string{"domain.NewWidget must stay visible in the use case"}},
		{"mappers live in mapper.go", mappersLiveInMapperFiles, []string{"toWidgetBody belongs in mapper.go"}},
		{"use cases hide no steps in helpers", useCasesShowEveryStep, []string{"method save hides a use case step", "function persistWidget hides a use case step"}},
		{"inbound adapters depend on ports", inboundAdaptersUsePorts, []string{"WidgetApplicationService must be reached through a port"}},
		{"inbound adapters use only domain errors", inboundAdaptersHideTheDomain, []string{"not domain.Widget"}},
		{"commands, queries and DTOs carry no domain types", boundaryDataIsPrimitive, []string{"CreateWidgetCommandDTO exposes a domain type"}},
		{"inbound ports expose no domain types", inboundPortsHideTheDomain, []string{"WidgetUseCasesPort exposes a domain type"}},
		{"the core never reads the clock", theCoreNeverReadsTheClock, []string{"time.Now must come through a port", "time.Since must come through a port", "time.Until must come through a port"}},
		{"the glossary names every domain type", glossaryNamesDomainTypes, []string{"Widget is missing from the glossary"}},
	}
}

var (
	clockReaders        = []string{"time.Now", "time.Since", "time.Until"}
	allowedStrings      = []string{"", "Allow", "Content-Type", "Location", "X-Request-Id", "application/json", "application/problem+json"}
	unsafeMiddleware    = []string{"middleware.RequestID", "middleware.Recoverer"}
	domainImports       = []string{"errors", "fmt", "slices", "strings", "time", "unicode", "unicode/utf8"}
	applicationImports  = []string{"context", "errors", "time"}
	allowedPointers     = []string{"http.Request", "http.Server", "slog.Logger", "http.MaxBytesError", "json.UnmarshalTypeError", "InMemoryNoteRepository"}
	allowedAddressOf    = []string{"http.Server", "InMemoryNoteRepository"}
	allowedShortNames   = []string{"_", "t", "ok", "id", "ID", "err", "ctx"}
	abbreviations       = []string{"addr", "arg", "args", "attr", "buf", "cfg", "cnt", "conf", "config", "ctl", "curr", "db", "doc", "docs", "dst", "env", "fn", "idx", "impl", "info", "len", "max", "mgr", "min", "msg", "num", "obj", "param", "params", "pkg", "prev", "ptr", "repo", "req", "res", "resp", "ret", "spec", "src", "srv", "str", "svc", "temp", "tmp", "val", "vals"}
	technicalDomainName = regexp.MustCompile(`(Entity|Model|VO|DTO|Impl|Manager|Service|Command|Query|Record)$`)
	applicationName     = regexp.MustCompile(`.+(Command|Query|DTO|ApplicationService|Port)$`)
	inboundName         = regexp.MustCompile(`.+(RequestDTO|ResponseDTO|Handler)$|^APIError$|^FieldError$`)
	outboundName        = regexp.MustCompile(`^(InMemory|System|Random|Postgres).+$|^[a-z].*Record$`)
	nonBusinessVerb     = regexp.MustCompile(`^(Set|Get|Update)[A-Z]`)
	mapperName          = regexp.MustCompile(`^to[A-Z]`)
	dataTypeName        = regexp.MustCompile(`(DTO|Record|Command|Query)$`)
	boundaryName        = regexp.MustCompile(`(Command|Query|DTO)$`)
	domainConstructor   = regexp.MustCompile(`^domain\.(New|Reconstitute)`)
)

func isOwnAdapter(imported string) bool {
	return strings.HasPrefix(imported, modulePath+"/") && strings.Contains(imported, "/adapters/")
}

func boundedContext(file sourceFile) string {
	segments := strings.Split(file.path, "/")
	if len(segments) < 2 || segments[0] != "internal" {
		return ""
	}
	return segments[1]
}

func isStandardLibrary(imported string) bool {
	return !strings.Contains(strings.SplitN(imported, "/", 2)[0], ".")
}

func domainImportsArePure(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isDomain) {
		for _, imported := range file.imports() {
			if !slices.Contains(domainImports, imported) {
				violations = append(violations, fmt.Sprintf("%s imports %q", file.path, imported))
			}
		}
	}
	return violations
}

func applicationImportsArePure(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isApplication) {
		ownDomain := modulePath + "/internal/" + boundedContext(file) + "/domain"
		for _, imported := range file.imports() {
			if !slices.Contains(applicationImports, imported) && imported != ownDomain {
				violations = append(violations, fmt.Sprintf("%s imports %q", file.path, imported))
			}
		}
	}
	return violations
}

func adaptersStayApart(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(func(file sourceFile) bool { return isInboundAdapter(file) || isOutboundAdapter(file) }) {
		for _, imported := range file.imports() {
			if isOwnAdapter(imported) {
				violations = append(violations, fmt.Sprintf("%s: adapter imports another adapter %q", file.path, imported))
			}
		}
	}
	return violations
}

func outboundAdaptersStayInTheRoot(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		if isCompositionRoot(file) || isInboundAdapter(file) || isOutboundAdapter(file) {
			continue
		}
		for _, imported := range file.imports() {
			if isOwnAdapter(imported) {
				violations = append(violations, fmt.Sprintf("%s imports %q: only the composition root may import adapters", file.path, imported))
			}
		}
	}
	return violations
}

func compositionRootAvoidsDomain(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isCompositionRoot) {
		for _, imported := range file.imports() {
			if strings.HasPrefix(imported, modulePath+"/") && strings.HasSuffix(imported, "/domain") {
				violations = append(violations, fmt.Sprintf("%s: composition root imports the domain", file.path))
			}
		}
	}
	return violations
}

func importsAreQualified(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isAnyCode) {
		for _, declaration := range file.syntax.Imports {
			if declaration.Name != nil {
				violations = append(violations, fmt.Sprintf("%s: import %s %s must not be dotted, blank or renamed", file.position(declaration), declaration.Name.Name, declaration.Path.Value))
			}
		}
	}
	return violations
}

func contextsStayApart(scanned project) []string {
	var violations []string
	internalPrefix := modulePath + "/internal/"
	for _, file := range scanned.filesIn(isProductionCode) {
		context := boundedContext(file)
		if context == "" {
			continue
		}
		for _, imported := range file.imports() {
			if strings.HasPrefix(imported, internalPrefix) && !strings.HasPrefix(imported, internalPrefix+context+"/") {
				violations = append(violations, fmt.Sprintf("%s imports another bounded context: %q", file.path, imported))
			}
		}
	}
	return violations
}

func frameworkStaysInTheInboundAdapter(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isAnyCode) {
		if inboundAdapterPattern.MatchString(file.path) {
			continue
		}
		for _, imported := range file.imports() {
			if !isStandardLibrary(imported) && !strings.HasPrefix(imported, modulePath) {
				violations = append(violations, fmt.Sprintf("%s: framework used outside the inbound adapter: %q", file.path, imported))
			}
		}
	}
	return violations
}

func unsafeMiddlewareStaysOut(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if selector, isSelector := node.(*ast.SelectorExpr); isSelector && slices.Contains(unsafeMiddleware, expressionName(selector)) {
				violations = append(violations, fmt.Sprintf("%s: %s must not be used", file.position(selector), expressionName(selector)))
			}
			return true
		})
	}
	return violations
}

func portsBelongToTheApplication(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		for _, declaration := range typeDeclarations(file) {
			if _, isInterface := declaration.Type.(*ast.InterfaceType); !isInterface {
				continue
			}
			name := declaration.Name.Name
			isPort := isApplication(file) && strings.HasSuffix(name, "Port")
			isDomainEvent := isDomain(file) && name == "Event"
			if !isPort && !isDomainEvent {
				violations = append(violations, fmt.Sprintf("%s: interface %s must be in the application layer, with a name that ends in Port", file.path, name))
			}
		}
	}
	return violations
}

func aggregateRoots(scanned project) []string {
	var roots []string
	for _, file := range scanned.filesIn(isDomain) {
		for _, function := range functionDeclarations(file) {
			if function.Name.Name == "PendingEvents" {
				roots = append(roots, receiverName(function))
			}
		}
	}
	return roots
}

func repositoriesServeAggregates(scanned project) []string {
	var violations []string
	roots := aggregateRoots(scanned)
	for _, file := range scanned.filesIn(isApplication) {
		for _, declaration := range typeDeclarations(file) {
			if aggregate, isRepository := strings.CutSuffix(declaration.Name.Name, "RepositoryPort"); isRepository && !slices.Contains(roots, aggregate) {
				violations = append(violations, fmt.Sprintf("%s: %s has no aggregate root", file.path, declaration.Name.Name))
			}
		}
	}
	return violations
}

func eventsArePastTense(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isDomain) {
		for _, function := range functionDeclarations(file) {
			if function.Name.Name == "EventName" && !strings.HasSuffix(receiverName(function), "ed") {
				violations = append(violations, fmt.Sprintf("%s: %s must be a past-tense fact", file.path, receiverName(function)))
			}
		}
	}
	return violations
}

func domainStateIsPrivate(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isDomain) {
		for _, declaration := range typeDeclarations(file) {
			structure, isStruct := declaration.Type.(*ast.StructType)
			if !isStruct {
				continue
			}
			for _, field := range structure.Fields.List {
				if len(field.Names) == 0 && ast.IsExported(expressionName(field.Type)) {
					violations = append(violations, fmt.Sprintf("%s: exported domain field %s", file.path, expressionName(field.Type)))
				}
				for _, name := range field.Names {
					if name.IsExported() {
						violations = append(violations, fmt.Sprintf("%s: exported domain field %s", file.path, name.Name))
					}
				}
			}
		}
	}
	return violations
}

func behaviourUsesBusinessVerbs(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isDomain) {
		for _, function := range functionDeclarations(file) {
			if nonBusinessVerb.MatchString(function.Name.Name) {
				violations = append(violations, fmt.Sprintf("%s: %s is not a business verb", file.path, function.Name.Name))
			}
		}
	}
	return violations
}

func aggregatesReferenceByID(scanned project) []string {
	var violations []string
	roots := aggregateRoots(scanned)
	for _, file := range scanned.filesIn(isDomain) {
		for _, declaration := range typeDeclarations(file) {
			structure, isStruct := declaration.Type.(*ast.StructType)
			if !isStruct {
				continue
			}
			for _, field := range structure.Fields.List {
				if fieldType := expressionName(field.Type); slices.Contains(roots, fieldType) && fieldType != declaration.Name.Name {
					violations = append(violations, fmt.Sprintf("%s: %s must reference %s by ID", file.path, declaration.Name.Name, fieldType))
				}
			}
		}
	}
	return violations
}

func pointersHaveAReason(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.StarExpr:
				if name := expressionName(typed); !slices.Contains(allowedPointers, name) {
					violations = append(violations, fmt.Sprintf("%s: pointer without a reason: %s", file.position(typed), name))
				}
			case *ast.CallExpr:
				if expressionName(typed.Fun) == "new" {
					violations = append(violations, fmt.Sprintf("%s: pointer without a reason: new(%s)", file.position(typed), expressionName(typed.Args[0])))
				}
			case *ast.UnaryExpr:
				if literal, isLiteral := typed.X.(*ast.CompositeLit); typed.Op == token.AND && isLiteral && !slices.Contains(allowedAddressOf, expressionName(literal.Type)) {
					violations = append(violations, fmt.Sprintf("%s: pointer without a reason: &%s", file.position(typed), expressionName(literal.Type)))
				}
			}
			return true
		})
	}
	return violations
}

func typesFollowLayerNames(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		for _, declaration := range typeDeclarations(file) {
			name := declaration.Name.Name
			var broken bool
			switch {
			case isDomain(file):
				broken = technicalDomainName.MatchString(name)
			case isApplication(file):
				broken = !applicationName.MatchString(name) || strings.HasSuffix(name, "CommandDTO") || strings.HasSuffix(name, "QueryDTO")
			case isInboundAdapter(file):
				broken = !inboundName.MatchString(name)
			case isOutboundAdapter(file):
				broken = !outboundName.MatchString(name)
			}
			if broken {
				violations = append(violations, fmt.Sprintf("%s: %s breaks the naming rule of its layer", file.path, name))
			}
		}
	}
	return violations
}

func everyTypeHasItsOwnFile(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		for _, declaration := range typeDeclarations(file) {
			if expected := snakeCase(declaration.Name.Name) + ".go"; file.name() != expected {
				violations = append(violations, fmt.Sprintf("%s: %s belongs in %s", file.path, declaration.Name.Name, expected))
			}
		}
	}
	return violations
}

func isFullName(name string) bool {
	if slices.Contains(allowedShortNames, name) {
		return true
	}
	if len(name) <= 2 {
		return false
	}
	for position, word := range splitWords(name) {
		lower := strings.ToLower(word)
		if slices.Contains(abbreviations, lower) || lower == "ctx" || (lower == "err" && position > 0) {
			return false
		}
	}
	return true
}

func namesAreFullWords(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isAnyCode) {
		for _, identifier := range declaredNames(file) {
			if !isFullName(identifier.Name) {
				violations = append(violations, fmt.Sprintf("%s: %s is not a full word", file.position(identifier), identifier.Name))
			}
		}
	}
	return violations
}

func numbersAreConstants(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		if file.name() == "constants.go" {
			continue
		}
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			literal, isLiteral := node.(*ast.BasicLit)
			if isLiteral && (literal.Kind == token.INT || literal.Kind == token.FLOAT) && literal.Value != "0" && literal.Value != "1" {
				violations = append(violations, fmt.Sprintf("%s: %s must be a constant", file.position(literal), literal.Value))
			}
			return true
		})
	}
	return violations
}

// stringsAreConstants allows paths, headers and content types. Schema tags and
// import paths aren't values, and sentinel errors stay together in errors.go.
func stringsAreConstants(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		if file.name() == "constants.go" || file.name() == "errors.go" {
			continue
		}
		schemaTags := map[*ast.BasicLit]bool{}
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.ImportSpec:
				return false
			case *ast.Field:
				schemaTags[typed.Tag] = true
			case *ast.BasicLit:
				if typed.Kind == token.STRING && !schemaTags[typed] && !isAllowedString(typed.Value) {
					violations = append(violations, fmt.Sprintf("%s: %s must be a constant", file.position(typed), typed.Value))
				}
			}
			return true
		})
	}
	return violations
}

func isAllowedString(literal string) bool {
	value, err := strconv.Unquote(literal)
	if err != nil {
		panic(err)
	}
	return slices.Contains(allowedStrings, value) || strings.HasPrefix(value, "/")
}

func structsAreMappedExplicitly(scanned project) []string {
	var violations []string
	var dataTypes []string
	for _, file := range scanned.filesIn(isProductionCode) {
		for _, declaration := range typeDeclarations(file) {
			if dataTypeName.MatchString(declaration.Name.Name) {
				dataTypes = append(dataTypes, declaration.Name.Name)
			}
		}
	}
	for _, file := range scanned.filesIn(isProductionCode) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}
			name := expressionName(call.Fun)
			if slices.Contains(dataTypes, name[strings.LastIndex(name, ".")+1:]) {
				violations = append(violations, fmt.Sprintf("%s: direct struct conversion to %s", file.position(call), name))
			}
			return true
		})
	}
	return violations
}

func mappersOnlyReshapeData(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		if file.name() != "mapper.go" {
			continue
		}
		for _, function := range functionDeclarations(file) {
			if function.Name.Name == "toDomain" && isOutboundAdapter(file) {
				continue
			}
			ast.Inspect(function, func(node ast.Node) bool {
				if call, isCall := node.(*ast.CallExpr); isCall && domainConstructor.MatchString(expressionName(call.Fun)) {
					violations = append(violations, fmt.Sprintf("%s: %s must stay visible in the use case", file.position(call), expressionName(call.Fun)))
				}
				return true
			})
		}
	}
	return violations
}

func mappersLiveInMapperFiles(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isProductionCode) {
		for _, function := range functionDeclarations(file) {
			if function.Recv == nil && mapperName.MatchString(function.Name.Name) && file.name() != "mapper.go" {
				violations = append(violations, fmt.Sprintf("%s: %s belongs in mapper.go", file.path, function.Name.Name))
			}
		}
	}
	return violations
}

func useCasesShowEveryStep(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isApplication) {
		for _, function := range functionDeclarations(file) {
			name := function.Name.Name
			switch {
			case strings.HasSuffix(receiverName(function), "ApplicationService") && !function.Name.IsExported() && name != "load":
				violations = append(violations, fmt.Sprintf("%s: method %s hides a use case step", file.path, name))
			case function.Recv == nil && !function.Name.IsExported() && !mapperName.MatchString(name):
				violations = append(violations, fmt.Sprintf("%s: function %s hides a use case step", file.path, name))
			}
		}
	}
	return violations
}

func inboundAdaptersUsePorts(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isInboundAdapter) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if selector, isSelector := node.(*ast.SelectorExpr); isSelector && strings.HasSuffix(selector.Sel.Name, "ApplicationService") {
				violations = append(violations, fmt.Sprintf("%s: %s must be reached through a port", file.position(selector), selector.Sel.Name))
			}
			return true
		})
	}
	return violations
}

func inboundAdaptersHideTheDomain(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isInboundAdapter) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if selector, isSelector := node.(*ast.SelectorExpr); isSelector && expressionName(selector.X) == "domain" && !strings.HasPrefix(selector.Sel.Name, "Err") {
				violations = append(violations, fmt.Sprintf("%s: inbound adapters may only use domain errors, not domain.%s", file.position(selector), selector.Sel.Name))
			}
			return true
		})
	}
	return violations
}

func usesDomainType(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(inner ast.Node) bool {
		if selector, isSelector := inner.(*ast.SelectorExpr); isSelector && expressionName(selector.X) == "domain" {
			found = true
		}
		return true
	})
	return found
}

func boundaryDataIsPrimitive(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isApplication) {
		for _, declaration := range typeDeclarations(file) {
			if boundaryName.MatchString(declaration.Name.Name) && usesDomainType(declaration) {
				violations = append(violations, fmt.Sprintf("%s: %s exposes a domain type", file.path, declaration.Name.Name))
			}
		}
	}
	return violations
}

func inboundPortsHideTheDomain(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(isApplication) {
		for _, declaration := range typeDeclarations(file) {
			if strings.HasSuffix(declaration.Name.Name, "UseCasesPort") && usesDomainType(declaration) {
				violations = append(violations, fmt.Sprintf("%s: %s exposes a domain type", file.path, declaration.Name.Name))
			}
		}
	}
	return violations
}

func theCoreNeverReadsTheClock(scanned project) []string {
	var violations []string
	for _, file := range scanned.filesIn(func(file sourceFile) bool { return isDomain(file) || isApplication(file) }) {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if selector, isSelector := node.(*ast.SelectorExpr); isSelector && slices.Contains(clockReaders, expressionName(selector)) {
				violations = append(violations, fmt.Sprintf("%s: %s must come through a port", file.position(selector), expressionName(selector)))
			}
			return true
		})
	}
	return violations
}

func glossaryNamesDomainTypes(scanned project) []string {
	var violations []string
	glossary := glossaryText(scanned)
	for _, file := range scanned.filesIn(isDomain) {
		for _, declaration := range typeDeclarations(file) {
			if name := declaration.Name.Name; ast.IsExported(name) && !strings.Contains(glossary, "`"+name+"`") {
				violations = append(violations, fmt.Sprintf("%s: %s is missing from the glossary", file.path, name))
			}
		}
	}
	return violations
}
