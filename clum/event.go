package clum

import "github.com/nu7hatch/gouuid"

type operation uint

const (
	// Join means the node is joining the cluster.
	Join operation = iota
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
	Operation operation

	Origin *Member

	Members []*Member

	Time uint64
}
