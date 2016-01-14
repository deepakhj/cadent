/*
An interface that is the "runner" for various Line Processors
*/

package splitter

type Phase int

const (
	Parsed            Phase = iota // initial parsing on direct incoming
	AccumulatedParsed              // if Accumulated and parsed, so we know NOT to run the accumualtor again
	Sent                           // delivered to output queue
)

type Origin int

const (
	TCP Origin = iota
	UDP
	HTTP
	Other
)

type SplitItem interface {
	Key() string
	Line() string
	Fields() []string
	Phase() Phase
	SetPhase(Phase)
	Origin() Origin
	SetOrigin(Origin)
	IsValid() bool
}

type Splitter interface {
	ProcessLine(line string) (SplitItem, error)
	Name() (name string)
}
