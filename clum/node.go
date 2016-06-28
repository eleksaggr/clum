package clum

import (
	"encoding/gob"
	"errors"
	"net"
	"strconv"
	"sync"

	"github.com/nu7hatch/gouuid"
)

const (
	// MaxConnFailures defines how many connection failures may happen, before the node terminates execution.
	maxConnFailures = 5
)

// Node is the representation of a node in the cluster.
type Node struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
	net.Listener

	members []*Member

	eventQueue []*Event
	queueMutex *sync.Mutex

	stop chan bool
}

// New creates a new Node that listens on the address host.
func New(host string) (node *Node, err error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	host, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}

	addr := net.ParseIP(host)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	node = &Node{
		ID: *id,

		Addr: addr,
		Port: uint16(port),

		members: make([]*Member, 0),

		eventQueue: make([]*Event, 0),
		queueMutex: &sync.Mutex{},

		stop: make(chan bool, 1),
	}

	if node.Listener, err = net.Listen("tcp", host); err != nil {
		return nil, err
	}

	return node, nil
}

// Run starts a loop in which the node accepts incoming connections and lets them be handled by the handle-method. Should the amount of connection failures exceed maxConnFailures, an error will be returned. The loop can be stopped in a controlled manner by calling the Stop-method.
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

				var event Event
				decoder := gob.NewDecoder(conn)
				decoder.Decode(&event)

				if err := node.handle(event); err != nil {
					return
				}
			}(conn)
		}
	}
	// Close the stop channel.
	close(node.stop)
	return err
}

func (node *Node) handle(event Event) (err error) {
	switch event.Event {
	case Join:
		node.members = append(node.members, &Member{
			ID:   event.SenderId,
			Addr: event.Addr,
			Port: event.Port,
		})

		node.queueMutex.Lock()
		node.eventQueue = append(node.eventQueue, &event)
		node.queueMutex.Unlock()
	case Gossip:
	case Leave:
	default:
		err = errors.New("Unknown event type received.")
	}

	return err
}

// Stop stops execution of the node.
func (node *Node) Stop() {
	node.stop <- true
}
