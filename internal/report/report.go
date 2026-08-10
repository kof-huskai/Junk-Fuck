// Package report defines the outcome model of a cleanup operation.
package report

// Status describes the outcome of a single cleanup item.
type Status string

const (
	StatusDeleted Status = "deleted"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
)

// Item is the result of one candidate during cleanup.
type Item struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Status Status `json:"status"`
	Reason string `json:"reason"`
}

// Report summarizes a cleanup operation.
type Report struct {
	DryRun       bool   `json:"dryRun"`
	DeletedCount int    `json:"deletedCount"`
	SkippedCount int    `json:"skippedCount"`
	FailedCount  int    `json:"failedCount"`
	BytesFreed   int64  `json:"bytesFreed"`
	Items        []Item `json:"items"`
}

// Add appends an item result and updates the counters.
func (r *Report) Add(item Item) {
	r.Items = append(r.Items, item)
	switch item.Status {
	case StatusDeleted:
		r.DeletedCount++
	case StatusSkipped:
		r.SkippedCount++
	case StatusFailed:
		r.FailedCount++
	}
}
