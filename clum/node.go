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
	MaximumHops          = 10
)

// Node is the representation of a node in the cluster.
type Node struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
	*net.TCPListener

	members []*Member

	eventQueue []*Event
	mutex      *sync.Mutex

	clock LogicalClock

	stop chan bool
}

// New creates a new Node that listens on the address host.
func New(host string) (node *Node, err error) {
	log.Printf("Trying to create new node on %v\n", host)

	// Generate a unique UUID for the node.
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	// Split the hostname and the port from the host argument.
	hostStr, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}

	// Bring host and port into the correct format.
	addr := net.ParseIP(hostStr)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	// Create a new node with the given details.
	node = &Node{
		ID: *id,

		Addr: addr,
		Port: uint16(port),

		eventQueue: make([]*Event, 0),
		mutex:      &sync.Mutex{},

		clock: &LamportClock{},

		stop: make(chan bool, 1),
	}

	// Create a tcp listener with the host argument.
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

	event := &Event{
		Origin: Member{
			ID:   node.ID,
			Addr: node.Addr,
			Port: node.Port,
		},
		TransferRequired: true,
	}

	host, portStr, err := net.SplitHostPort(host)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	return node.sendToMember(&Member{
		Addr: net.ParseIP(host),
		Port: uint16(port),
	}, event)
}

// Run starts a loop in which the node accepts incoming connections and lets them be handled by the handle-method, additionally the node will gossip with other nodes. Should the amount of connection failures exceed maxConnFailures, an error will be returned. The loop can be stopped in a controlled manner by calling the Stop-method.
func (node *Node) Run() (err error) {
	go node.gossip()

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
					log.Printf("An error occured during communication with a peer: %v\n", err)
					return
				}
				if event.Hops > MaximumHops {
					// Discard the message.
					log.Printf("Maximum hops exceeded for message.\n")
					return
				}

				if event.Time > node.clock.Time() {
					node.clock.Set(event.Time)
				}
				node.clock.Increment()

				if err := node.handle(&event); err != nil {
					log.Printf("An error occured during handling of an event: %v\n", err)
					return
				}
				log.Printf("Members after operation: %v\n", len(node.members))
			}(conn)
		}
	}
	log.Printf("Trying to leave cluster...\n")
	event := &Event{
		Event:    Leave,
		SenderID: node.ID,
	}

	if err = node.sendToRandomMember(event); err != nil {
		log.Printf("Error during communication with a peer: %v\n", err)
	}

	log.Printf("Cleaning up...\n")
	close(node.stop)
	return err
}

// sendToMember sends an Event to a Member.
func (node *Node) sendToMember(member *Member, event *Event) (err error) {
	if member == nil || event == nil {
		return errors.New("Member/Event may not be nil.")
	}

	// Convert host and port to a string.
	host := member.Addr.String()
	port := strconv.Itoa(int(member.Port))
	hostPort := net.JoinHostPort(host, port)

	// Connect to member.
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Set event sender id.
	event.SenderID = node.ID
	// Set time in event and increment it.
	event.Time = node.clock.Time()
	node.clock.Increment()

	// Send event to member.
	if gob.NewEncoder(conn).Encode(event); err != nil {
		return err
	}
	return nil
}

func (node *Node) sendToRandomMember(event *Event) (err error) {
	if event == nil {
		return errors.New("Event may not be nil.")
	}

	if len(node.members) == 0 {
		return errors.New("No members registered.")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	index := rand.Intn(len(node.members))

	if err := node.sendToMember(node.members[index], event); err != nil {
		return err
	}
	return nil
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
					event := node.eventQueue[0]
					node.eventQueue = node.eventQueue[1:]
					if err := node.sendToRandomMember(event); err != nil {
						log.Printf("An error occured during communication with a peer: %v\n", err)
						continue
					}
				}
				lastGossipTime = time.Now()
			}
		}
	}
}

func (node *Node) handle(event *Event) (err error) {
	switch event.Event {
	case Join:
		log.Printf("[EVENT] Join")
		node.addEvent(event)

		if event.Origin.ID == node.ID {
			// Ignore this event, since it originated at the local node.
			return nil
		}

		if event.TransferRequired {
			event.TransferRequired = false
			response := &Event{
				Event:            Transfer,
				SenderID:         node.ID,
				Origin:           event.Origin,
				TransferRequired: false,
			}

			var members []Member
			for _, m := range node.members {
				members = append(members, *m)
			}
			members = append(members, Member{
				ID:   node.ID,
				Addr: node.Addr,
				Port: node.Port,
			})

			response.Members = members
			if err = node.sendToMember(&event.Origin, response); err != nil {
				return err
			}
		}

		node.addMember(&event.Origin)
	case Transfer:
		log.Printf("[EVENT] Transfer")
		for _, m := range event.Members {
			node.addMember(&m)
		}
	case Leave:
		log.Printf("[EVENT] Leave")
		if err = node.removeMember(event.SenderID); err != nil {
			log.Printf("Tried to remove an unknown member.")
		}

		log.Printf("Removed member with the id %v", event.SenderID.String())
		// Add leave event to eventqueue.
		node.addEvent(event)
	default:
		log.Printf("Unknown event received from peer.\n")
		err = errors.New("Unknown event type received.")

	}
	return err
}

func (node *Node) addEvent(event *Event) (err error) {
	if event == nil {
		return errors.New("Event may not be nil.")
	}

	event.Hops++
	node.mutex.Lock()
	node.eventQueue = append(node.eventQueue, event)
	node.mutex.Unlock()
	return nil
}

func (node *Node) addMember(member *Member) (err error) {
	if member == nil {
		return errors.New("Member may not be nil.")
	}

	node.mutex.Lock()
	found := false
	for _, m := range node.members {
		if m.ID == member.ID {
			found = true
		}
	}

	if found {
		return errors.New("Member already registered with node.")
	}

	node.members = append(node.members, member)
	node.mutex.Unlock()
	log.Printf("Members: %v\n", node.members)
	return nil
}

func (node *Node) removeMember(id uuid.UUID) (err error) {
	node.mutex.Lock()
	found := false
	for i, m := range node.members {
		if m.ID == id {
			node.members = append(node.members[:i], node.members[i+1:]...)
			found = true
		}
	}
	node.mutex.Unlock()

	if !found {
		return errors.New("Member has not been found in memberlist.")
	}
	return nil
}

// Members returns the members of the node.
func (node *Node) Members() []*Member {
	return node.members
}

// Stop stops execution of the node.
func (node *Node) Stop() {
	// Set TCP timeout so listener will die.
	node.SetDeadline(time.Now())
	node.stop <- true
}
