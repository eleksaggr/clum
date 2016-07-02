package clum

import "github.com/nu7hatch/gouuid"

type Operation uint

const (
	// Join means the node is joining the cluster.
	Join Operation = iota
	// Leave means the node is leaving the cluster.
	Leave
	// Transfer means the node is transfering its members to the peer.
	Transfer
	// NotifyJoin means the node is notifying another node of a join that happened.
	NotifyJoin
	// NotifyLeave means the node is notifying another node of a leave that happened.
	NotifyLeave
)

// Event is the representation of an event caused by a node.
type Event struct {
	SenderID  uuid.UUID
	Operation Operation

	Origin Member

	Members []*Member

	Time uint64
	Hops uint
}
