package clum

import (
	"sync"

	"github.com/nu7hatch/gouuid"
)

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

// EventQueue is a thread-safe queue holding events.
type EventQueue struct {
	events []Event
	mutex  *sync.Mutex
}

// Enqueue adds an event to the tail of the queue.
func (e *EventQueue) Enqueue(event Event) {
	e.mutex.Lock()
	e.events = append(e.events, event)
	e.mutex.Unlock()
}

// Dequeue gets an element from the head of the queue and removes it.
func (e *EventQueue) Dequeue() (event Event) {
	e.mutex.Lock()
	event = e.events[0]
	e.events = e.events[1:]
	e.mutex.Unlock()

	return event
}

// Length returns the amount of events currently in the queue.
func (e *EventQueue) Length() int {
	return len(e.events)
}
