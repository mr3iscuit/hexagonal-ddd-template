// Package architecture_test enforces the DDD and hexagonal rules on the source
// code, so that breaking one fails go test instead of relying on review.
package architecture_test

import (
	"strings"
	"testing"
)

const (
	projectRoot = "../.."
	fixtureRoot = "testdata/violations"
)

func TestProjectFollowsEveryRule(t *testing.T) {
	project := loadProject(t, projectRoot)
	for _, layer := range []func(sourceFile) bool{isDomain, isApplication, isInboundAdapter, isOutboundAdapter, isCompositionRoot} {
		if len(project.filesIn(layer)) == 0 {
			t.Fatal("a layer has no files, so the rules would pass by checking nothing")
		}
	}
	for _, architectureRule := range rules() {
		t.Run(architectureRule.name, func(t *testing.T) {
			for _, violation := range architectureRule.check(project) {
				t.Error(violation)
			}
		})
	}
}

// A rule that can't fail proves nothing, so each one must catch the violation
// planted for it.
func TestEveryRuleCatchesItsPlantedViolation(t *testing.T) {
	fixture := loadProject(t, fixtureRoot)
	for _, architectureRule := range rules() {
		t.Run(architectureRule.name, func(t *testing.T) {
			if len(architectureRule.planted) == 0 {
				t.Fatal("the rule has no planted violation")
			}
			violations := strings.Join(architectureRule.check(fixture), "\n")
			for _, expected := range architectureRule.planted {
				if !strings.Contains(violations, expected) {
					t.Errorf("expected a violation containing %q, got %q", expected, violations)
				}
			}
		})
	}
}
