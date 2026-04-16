package events

type DocumentCreatedEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
