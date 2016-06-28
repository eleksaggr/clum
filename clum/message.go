package clum

type eventType uint

const (
	Join eventType = iota
	Gossip
	Leave
)

type LamportTime uint64

type Event struct {
	event       eventType
	lamportTime LamportTime
}
