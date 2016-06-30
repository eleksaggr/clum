package clum

import (
	"encoding/gob"
	"errors"
	"log"
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
	log.Printf("Trying to create new node on %v\n", host)
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

	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}

	if node.TCPListener, err = net.ListenTCP("tcp", tcpAddr); err != nil {
		return nil, err
	}

	log.Printf("Created node with ID %v\n", node.ID.String())
	return node, nil
}

// Join makes a node join a cluster by contacting a node under the address host.
func (node *Node) Join(host string) (err error) {
	log.Printf("Trying to join cluster on %v\n", host)
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

	if err = gob.NewEncoder(conn).Encode(event); err != nil {
		return err
	}

	log.Printf("Joining cluster was successful.\n")
	return nil
}

// Run starts a loop in which the node accepts incoming connections and lets them be handled by the handle-method, additionally the node will gossip with other nodes. Should the amount of connection failures exceed maxConnFailures, an error will be returned. The loop can be stopped in a controlled manner by calling the Stop-method.
func (node *Node) Run() (err error) {
	log.Printf("Starting gossip routine...\n")
	// go node.gossip()

	failCounter := 0
loop:
	for {
		select {
		case <-node.stop:
			log.Printf("Stopping handle routine.\n")
			break loop
		default:
			if failCounter >= maxConnFailures {
				err = errors.New("Maximum amount of connection failures exceeded.")
				break loop
			}

			log.Printf("Listening...\n")
			conn, err := node.Accept()
			if err != nil {
				log.Printf("Connection error occured, retrying listening...\n")
				failCounter++
				continue
			}

			go func(conn net.Conn) {
				defer conn.Close()

				var event Event
				decoder := gob.NewDecoder(conn)
				if err := decoder.Decode(&event); err != nil {
					log.Printf("An error occured during communication with a peer: %v\n", err)
					return
				}

				log.Printf("Passing event to handle...\n")
				if err := node.handle(event); err != nil {
					log.Printf("An error occured during handling of an event: %v\n", err)
					return
				}
			}(conn)
		}
	}

	log.Printf("Cleaning up...\n")
	close(node.stop)
	return err
}

func (node *Node) gossip() {
	lastGossipTime := time.Now()
loop:
	for {
		select {
		case <-node.stop:
			log.Printf("Stopping gossip routine.\n")
			break loop
		default:
			if time.Since(lastGossipTime) > communicationTimeout {
				if len(node.eventQueue) != 0 {
					log.Printf("Processing next event in queue...\n")
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
						log.Printf("Error during communcation with peer: %v\n", err)
						continue
					}

					if gob.NewEncoder(conn).Encode(event); err != nil {
						log.Printf("Error during communication with peer: %v\n", err)
						continue
					}
				}

				lastGossipTime = time.Now()
			}
		}
	}
}

func (node *Node) handle(event Event) (err error) {
	log.Printf("Handling an event...\n")
	switch event.Event {
	case Join:
		log.Printf("Join event received from peer.\n")
		log.Printf("Event: %v\n", event)
		node.members = append(node.members, &Member{
			ID:   event.SenderID,
			Addr: event.Addr,
			Port: event.Port,
		})

		node.queueMutex.Lock()
		node.eventQueue = append(node.eventQueue, &event)
		node.queueMutex.Unlock()
		log.Printf("Members: %v\n", node.members)
	case Leave:
		log.Printf("Leave event received from peer.\n")
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
		log.Printf("Unkown event received from peer.\n")
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
