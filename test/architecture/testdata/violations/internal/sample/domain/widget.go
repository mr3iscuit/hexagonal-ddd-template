package domain

import (
	"net/http"
	"time"
)

type Clock interface {
	Now() time.Time
}

type WidgetEntity struct{}

type Widget struct {
	Name string
}

func (widget *Widget) SetName(name string) {
	widget.Name = name
}

func (widget Widget) PendingEvents() []string { return nil }

func (widget Widget) Weight() int {
	i := 42
	return i
}

func Stamp(request *http.Request) time.Time {
	_ = new(Widget)
	_ = &Gadget{}
	_ = time.Since(time.Until(time.Now()))
	return time.Now()
}

type WidgetChange struct{}

func (WidgetChange) EventName() string { return "widget.change" }
