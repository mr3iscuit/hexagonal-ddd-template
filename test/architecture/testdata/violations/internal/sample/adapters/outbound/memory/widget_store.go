package memory

import application "github.com/mr3iscuit/hexagonal-ddd-template/internal/sample/application"

type WidgetStore struct{}

var _ = application.WidgetApplicationService{}
