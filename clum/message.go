package clum

import "github.com/nu7hatch/gouuid"

type eventType uint

const (
	// Join means the node is joining the cluster.
	Join eventType = iota
	// Leave means the node is leaving the cluster.
	Leave
	// Transfer means the node is transfering its members to the peer.
	Transfer
)

// Event is the representation of an event caused by a node.
type Event struct {
	SenderID uuid.UUID
	Event    eventType

	Origin Member

	Members []Member

	Time uint64
	Hops uint

	TransferRequired bool
}
