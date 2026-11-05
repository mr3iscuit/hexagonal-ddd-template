package domain

type Gadget struct {
	widget Widget
}

type Gadget2 struct{}

func (gadget Gadget) PendingEvents() []string { return nil }
