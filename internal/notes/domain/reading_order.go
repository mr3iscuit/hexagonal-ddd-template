package domain

import (
	"slices"
	"strings"
)

// OrderForReading puts the most recently edited notes first. The ID settles a
// tie, so the order never changes between two reads.
func OrderForReading(notes []Note) []Note {
	ordered := slices.Clone(notes)
	slices.SortFunc(ordered, func(first, second Note) int {
		if order := second.editedAt.Compare(first.editedAt); order != 0 {
			return order
		}
		return strings.Compare(first.id.value, second.id.value)
	})
	return ordered
}
