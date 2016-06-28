package clum

import (
	"errors"
	"net"

	"github.com/nu7hatch/gouuid"
)

const (
	// MaxConnFailures defines how many connection failures may happen, before the node terminates execution.
	maxConnFailures = 5
)

// Node is the representation of a node in the cluster.
type Node struct {
	ID uuid.UUID

	Host string
	net.Listener

	stop chan bool
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

		stop: make(chan bool, 1),
	}

	if node.Listener, err = net.Listen("tcp", host); err != nil {
		return nil, err
	}

	return node, nil
}

func (node *Node) Run() (err error) {
	failCounter := 0
loop:
	for {
		select {
		case <-node.stop:
			break loop
		default:
			if failCounter >= maxConnFailures {
				err = errors.New("Maximum amount of connection failures exceeded.")
				break loop
			}
			conn, err := node.Accept()
			if err != nil {
				failCounter++
				continue
			}
			go func(conn net.Conn) {
				defer conn.Close()
				// Handle client here.
			}(conn)
		}
	}
	// Close the stop channel.
	close(node.stop)
	return err
}
