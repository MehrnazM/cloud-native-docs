package model

type Status string

const (
	StatusPending   Status = "pending"
	StatusProcessed Status = "processed"
	StatusFailed    Status = "failed"
	StatusDone      Status = "done"
)
