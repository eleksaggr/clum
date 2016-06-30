package clum

import (
	"errors"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
)

const (
	maxPortTries = 10
	basePort     = 10000
)

func TestJoin(t *testing.T) {
	node, err := tryCreateNode()
	if err != nil {
		t.Fatalf("Node 0: %v\n", err)
	}

	node1, err := tryCreateNode()
	if err != nil {
		t.Fatalf("Node 1: %v\n", err)
	}

	stopChan := make(chan bool)
	go func(stop chan bool) {
		if err := node1.Run(); err != nil {
			t.Errorf("%v\n", err)
		}
		stop <- true
	}(stopChan)

	portStr := strconv.Itoa(int(node1.Port))
	hostPort := net.JoinHostPort(node1.Addr.String(), portStr)

	if err := node.Join(hostPort); err != nil {
		t.Errorf("Error during joining of other node.\nError: %v\n", err)
	}
	node1.Stop()

	<-stopChan
	close(stopChan)

	if len(node1.Members()) != 1 {
		t.Error("Node did not join cluster.")
	}
}

// func TestHandleJoin(t *testing.T) {
// 	node, err := tryCreateNode()
// 	if err != nil {
// 		t.Fatalf("%v\n", err)
// 	}
//
// 	event := Event{
// 		Event:    Join,
// 		SenderID: [16]byte{},
//
// 		Addr: net.ParseIP("127.0.0.1"),
// 		Port: 0,
//
// 		LamportTime: 1,
// 	}
//
// 	if err = node.handle(event); err != nil {
// 		t.Error("Error during handling of Join event.")
// 	}
//
// 	members := node.Members()
// 	if len(members) != 1 {
// 		t.Error("Member has not been added by event.")
// 	}
// }

func TestHandleLeave(t *testing.T) {
	node, err := tryCreateNode()
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	node.members = append(node.members, &Member{
		ID: [16]byte{},
	})

	event := Event{
		Event:    Leave,
		SenderID: [16]byte{},

		Addr: net.ParseIP("localhost"),
		Port: 0,

		LamportTime: 1,
	}

	if err = node.handle(event); err != nil {
		t.Error("Error during handling of Leave event.")
	}

	members := node.Members()
	if len(members) != 0 {
		t.Error("Member has not been removed by event.")
	}
}

func TestHandleWrongEvent(t *testing.T) {
	node, err := tryCreateNode()
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	event := Event{
		Event: 255,
	}
	if err = node.handle(event); err == nil {
		t.Error("Error during handling of unrecognized event.")
	}
}

func TestHandleLeaveUnknownMember(t *testing.T) {
	node, err := tryCreateNode()
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	node.members = append(node.members, &Member{
		ID: [16]byte{},
	})

	event := Event{
		Event:    Leave,
		SenderID: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},

		Addr: net.ParseIP("localhost"),
		Port: 0,

		LamportTime: 1,
	}

	if err = node.handle(event); err != nil {
		t.Error("Error during handling of Leave event for unknown member.")
	}

	members := node.Members()
	if len(members) != 1 {
		t.Error("Member with different id has been removed by event.")
	}
}

func TestHandleLeaveNoMembers(t *testing.T) {
	node, err := tryCreateNode()
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	event := Event{
		Event:    Leave,
		SenderID: [16]byte{},

		Addr: net.ParseIP("127.0.0.1"),
		Port: 0,

		LamportTime: 1,
	}

	if err = node.handle(event); err != nil {
		t.Error("Error during handling of Leave event.")
	}

	members := node.Members()
	if len(members) != 0 {
		t.Error("Members are not unmodified.")
	}
}

func tryCreateNode() (node *Node, err error) {
	rand.Seed(time.Now().UTC().UnixNano())
	err = errors.New("Satisfy first loop condition.")
	for i := 0; i <= maxPortTries && err != nil; i++ {
		if i == maxPortTries {
			return nil, errors.New("Exceeded maximum amount of tries for node creation.")
		}
		port := basePort + rand.Intn(65536-basePort)

		portStr := strconv.Itoa(port)
		hostPort := net.JoinHostPort("127.0.0.1", portStr)

		node, err = New(hostPort)
	}
	return node, nil
}
