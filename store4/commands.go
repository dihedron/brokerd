package cluster

// CommandType represents the type of command.
type CommandType int8

const (
	// Get is the "Get" command type.
	Get CommandType = iota
	// Set is the "Set" command type.
	Set
	// Delete is the "Delete" command type.
	Delete
)

// Command is the Finite State Machine command.
type Command struct {
	Type  CommandType `json:"type"`
	Key   string      `json:"key"`
	Value string      `json:"value,omitempty"`
}
