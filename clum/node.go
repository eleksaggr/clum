package clum

import (
	"encoding/gob"
	"errors"
	"log"
	"math/rand"
	"net"
	"strconv"
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
	Member
	members *MemberList

	listener *net.TCPListener

	clock LogicalClock
	stop  chan bool
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
		Member: Member{
			ID: *id,

			Addr: addr,
			Port: uint16(port),
		},
		members: NewMemberList(),
		clock:   &LamportClock{},
		stop:    make(chan bool, 1),
	}

	// Subscribe as observer to the member list.
	node.members.Subscribe(node)

	// Create a tcp listener with the host argument.
	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, err
	}
	if node.listener, err = net.ListenTCP("tcp", tcpAddr); err != nil {
		return nil, err
	}

	log.Printf("Created node with ID %v\n", node.ID.String())
	return node, nil
}

// Join makes a node join a cluster by contacting a node under the address host.
func (node *Node) Join(host string) (err error) {
	log.Printf("Trying to join cluster on %v\n", host)

	event := &Event{
		SenderID:  node.ID,
		Operation: Join,
		Origin: &Member{
			ID:   node.ID,
			Addr: node.Addr,
			Port: node.Port,
		},
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
	failCounter := 0
loop:
	for {
		select {
		case <-node.stop:
			log.Printf("Stopping handle routine.\n")
			break loop
		default:
			// If we exceed the maximum amount of connection failures, exit the loop.
			if failCounter >= maxConnFailures {
				err = errors.New("Maximum amount of connection failures exceeded.")
				break loop
			}

			conn, err := node.listener.Accept()
			if err != nil {
				failCounter++
				continue
			}

			go func(conn net.Conn) {
				defer conn.Close()

				event, err := node.receive(conn)
				if err != nil {
					log.Printf("receive: %v\n", err)
					return
				}

				if err := node.handle(event); err != nil {
					log.Printf("handle: %v\n", err)
					return
				}
				log.Printf("Members: %v\n", node.members.Length())
			}(conn)
		}
	}
	log.Printf("Trying to leave cluster...\n")
	event := Event{
		Operation: Leave,
		SenderID:  node.ID,
		Origin:    &node.Member,
	}

	if err = node.sendToRandomMember(&event); err != nil {
		log.Printf("sendToRandomMember: %v\n", err)
	}

	log.Printf("Cleaning up...\n")
	close(node.stop)
	return err
}

func (node *Node) receive(conn net.Conn) (event *Event, err error) {
	if conn == nil {
		return nil, errors.New("Connection may not be nil.")
	}

	// Receive the event and decode it.
	event = new(Event)
	if err := gob.NewDecoder(conn).Decode(event); err != nil {
		return nil, err
	}

	// Update logical clock time.
	if event.Time > node.clock.Time() {
		node.clock.Set(event.Time)
	}
	node.clock.Increment()
	return event, nil
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

	if node.members.Length() == 0 {
		return errors.New("No members registered.")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	index := rand.Intn(node.members.Length())

	if err := node.sendToMember(node.members.Members()[index], event); err != nil {
		return err
	}
	return nil
}

func (node *Node) handle(event *Event) (err error) {
	log.Printf("Time: %v\n", event.Time)
	switch event.Operation {
	case Join:
		log.Printf("Operation: Join\n")
		// Create Transfer event and prepare members for transport.
		response := &Event{
			SenderID:  node.ID,
			Operation: Transfer,
		}

		members := make([]*Member, node.members.Length())
		copy(members, node.members.Members())
		members = append(members, &node.Member)
		response.Members = members
		// Send the transfer message to the new member.
		node.sendToMember(event.Origin, response)

		// Add new member to memberlist.
		node.members.Add(event.Origin)
	case Transfer:
		log.Printf("Operation: Transfer\n")
		log.Printf("Transfering %v members...\n", len(event.Members))
		node.members.SetMembers(event.Members)
	case Leave:
		log.Printf("Operation: Leave\n")
		node.members.Remove(event.Origin)
	case NotifyJoin:
		log.Printf("Operation: NotifyJoin\n")
		if event.Origin.ID != node.ID {
			node.members.Add(event.Origin)
		}
	case NotifyLeave:
		log.Printf("Operation: NotifyLeave\n")
		node.members.Remove(event.Origin)
	default:
		return errors.New("Unknown operation requested.")
	}
	return nil
}

func (node *Node) update(origin *Member, update updateType) {
	if node.members.Length() == 0 {
		// No sense in updating, when we don't have a cluster.
		return
	}
	event := &Event{
		SenderID: node.ID,
		Origin:   origin,
	}

	switch update {
	case Add:
		event.Operation = NotifyJoin
	case Remove:
		event.Operation = NotifyLeave
	}

	// for i := 0; i < node.members.Length()+1; i++ {
	// 	node.sendToRandomMember(event)
	// 	time.Sleep(communicationTimeout)
	// }
	for _, m := range node.members.Members() {
		node.sendToMember(m, event)
	}
}

// Members returns the members of the node.
func (node *Node) Members() []*Member {
	return node.members.Members()
}

// Stop stops execution of the node.
func (node *Node) Stop() {
	// Set TCP timeout so listener will die.
	node.listener.SetDeadline(time.Now())
	node.stop <- true
}
