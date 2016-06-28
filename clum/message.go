package clum

import (
	"net"

	"github.com/nu7hatch/gouuid"
)

type eventType uint

const (
	Join eventType = iota
	Gossip
	Leave
)

type LamportTime uint64

type Event struct {
	SenderId uuid.UUID
	Event    eventType

	Addr net.IP
	Port uint16

	LamportTime LamportTime
}
