package application

import "github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/domain"

func toWidget(name string) (domain.Widget, error) {
	return domain.NewWidget(name)
}
