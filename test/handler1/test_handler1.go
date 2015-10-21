package handler1

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Caption() string {
	return "Test handler 1"
}

func (h *Handler) Description() string {
	return "First handler for tests"
}
