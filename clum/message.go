package clum

import (
	"net"

	"github.com/nu7hatch/gouuid"
)

type eventType uint

const (
	// Join means the node is joining the cluster.
	Join eventType = iota
	// Leave means the node is leaving the cluster.
	Leave
	// Transfer means the node is transfering its members to the peer.
	Transfer
)

// LamportTime is a timestamp used to reorder messages.
type LamportTime uint64

// Event is the representation of an event caused by a node.
type Event struct {
	SenderID uuid.UUID
	Event    eventType

	Addr net.IP
	Port uint16

	Members []Member

	LamportTime LamportTime
}
