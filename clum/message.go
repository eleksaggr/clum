package clum

type eventType uint

const (
	Join eventType = iota
	Notify
	Leave
)

type LamportTime uint64

type Event struct {
	event       eventType
	lamportTime LamportTime
}
