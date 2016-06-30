package clum

import (
	"encoding/gob"
	"errors"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/nu7hatch/gouuid"
)

const (
	// MaxConnFailures defines how many connection failures may happen, before the node terminates execution.
	maxConnFailures      = 5
	communicationTimeout = time.Second * 3
)

// Node is the representation of a node in the cluster.
type Node struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
	*net.TCPListener

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

	hostStr, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}

	addr := net.ParseIP(hostStr)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	node = &Node{
		ID: *id,

		Addr: addr,
		Port: uint16(port),

		eventQueue: make([]*Event, 0),
		queueMutex: &sync.Mutex{},

		stop: make(chan bool, 1),
	}

	if node.Listener, err = net.Listen("tcp", host); err != nil {
		return nil, err
	}

	return node, nil
}

// Join makes a node join a cluster by contacting a node under the address host.
func (node *Node) Join(host string) (err error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}

	event := Event{
		Event:    Join,
		SenderID: node.ID,
		Addr:     node.Addr,
		Port:     node.Port,
	}

	encoder := gob.NewEncoder(conn)
	if err = encoder.Encode(event); err != nil {
		return err
	}
	return nil
}

// Run starts a loop in which the node accepts incoming connections and lets them be handled by the handle-method, additionally the node will gossip with other nodes. Should the amount of connection failures exceed maxConnFailures, an error will be returned. The loop can be stopped in a controlled manner by calling the Stop-method.
func (node *Node) Run() (err error) {
	go node.gossip()

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
				if err := decoder.Decode(&event); err != nil {
					return
				}

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

func (node *Node) gossip() {
	lastGossipTime := time.Now()
loop:
	for {
		select {
		case <-node.stop:
			break loop
		default:
			if time.Since(lastGossipTime) > communicationTimeout {
				if len(node.eventQueue) != 0 {
					var event *Event
					node.queueMutex.Lock()
					event = node.eventQueue[0]
					node.eventQueue = node.eventQueue[1:]
					node.queueMutex.Unlock()

					// Select random peer to gossip with.
					memberIndex := rand.Intn(len(node.members))

					hostStr := node.members[memberIndex].Addr.String()
					portStr := strconv.Itoa(int(node.members[memberIndex].Port))
					hostPort := net.JoinHostPort(hostStr, portStr)

					conn, err := net.Dial("tcp", hostPort)
					if err != nil {
						continue
					}

					encoder := gob.NewEncoder(conn)
					encoder.Encode(event)
				}

				lastGossipTime = time.Now()
			}
		}
	}
}

func (node *Node) handle(event Event) (err error) {
	switch event.Event {
	case Join:
		node.members = append(node.members, &Member{
			ID:   event.SenderID,
			Addr: event.Addr,
			Port: event.Port,
		})

		node.queueMutex.Lock()
		node.eventQueue = append(node.eventQueue, &event)
		node.queueMutex.Unlock()
	case Leave:
		for i, member := range node.members {
			if member.ID == event.SenderID {
				// Remove the member from the memberlist.
				node.members = append(node.members[:i], node.members[i+1:]...)

				node.queueMutex.Lock()
				node.eventQueue = append(node.eventQueue, &event)
				node.queueMutex.Unlock()
			}
		}
	default:
		err = errors.New("Unknown event type received.")
	}

	return err
}

// Members returns the members of the node.
func (node *Node) Members() []*Member {
	return node.members
}

// Stop stops execution of the node.
func (node *Node) Stop() {
	// Set TCP timeout so listener will die.
	node.SetDeadline(time.Now().Add(time.Second))
	node.stop <- true
}
