package clum

import (
	"errors"
	"math/rand"
	"net"
	"strconv"
	"testing"
)

const (
	maxPortTries = 10
	basePort     = 10000
)

func TestHandleJoin(t *testing.T) {
	node, err := tryCreateNode()
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	event := Event{
		Event:    Join,
		SenderId: [16]byte{},

		Addr: net.ParseIP("localhost"),
		Port: 0,

		LamportTime: 1,
	}

	if err = node.handle(event); err != nil {
		t.Error("Error during handling of Join event.")
	}

	members := node.Members()
	if len(members) != 1 {
		t.Error("Member has not been added by event.")
	}
}

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
		SenderId: [16]byte{},

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

func tryCreateNode() (node *Node, err error) {
	err = errors.New("Satisfy first loop condition.")
	for i := 0; i <= maxPortTries && err != nil; i++ {
		if i == maxPortTries {
			return nil, errors.New("Exceeded maximum amount of tries for node creation.")
		}
		port := basePort + rand.Intn(65536-basePort)

		portStr := strconv.Itoa(port)
		hostPort := net.JoinHostPort("localhost", portStr)

		node, err = New(hostPort)
	}
	return node, nil
}
