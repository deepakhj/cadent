package prereg

// the acctual filter action
type FilterItem interface {
	Name() string
	Type() string
	Rejecting() bool
	Init() error
	Match(string) (bool, bool, error) // match, reject, error
	Backend() string
	SetBackend(string) (string, error)
	ToString() string
}

// allow the filter items to use this to Send to queues if desired
type BackendItem interface {
	Send(string)
}
