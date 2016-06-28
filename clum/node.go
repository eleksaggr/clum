package clum

import (
	"net"

	"github.com/nu7hatch/gouuid"
)

// Node is the representation of a node in the cluster.
type Node struct {
	ID uuid.UUID

	Host string
	net.Listener
}

// New creates a new Node that listens on the address host.
func New(host string) (node *Node, err error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	node = &Node{
		ID: *id,

		Host: host,
	}

	if node.Listener, err = net.Listen("tcp", host); err != nil {
		return nil, err
	}

	return node, nil
}
