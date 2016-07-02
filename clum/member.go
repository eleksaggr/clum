package clum

import (
	"errors"
	"net"
	"sync"

	"github.com/nu7hatch/gouuid"
)

// Member is the information a node knowns about any other node in the cluster.
type Member struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
}

// MemberList is a thread-safe structure that manages a list of members.
type MemberList struct {
	members []*Member
	mutex   *sync.Mutex
}

// Add adds a member to the list, if he's not already in there. Should he already exist, nothing happens.
func (m *MemberList) Add(member *Member) (err error) {
	if member == nil {
		return errors.New("Member may not be nil.")
	}

	m.mutex.Lock()
	found := false
	for _, mem := range m.members {
		if mem.ID == member.ID {
			found = true
		}
	}

	if !found {
		m.members = append(m.members, member)
	}
	m.mutex.Unlock()
	return nil // Adding a member that's already in the list, is not an error.
}

// Remove removes a member from the list, if he's in there. Should he not exist, nothing happens.
func (m *MemberList) Remove(id uuid.UUID) {
	m.mutex.Lock()
	for i, member := range m.members {
		if member.ID == id {
			m.members = append(m.members[:i], m.members[i+1:]...)
		}
	}
	m.mutex.Unlock()
}

// Members returns a list of members currently in the list.
func (m *MemberList) Members() []*Member {
	m.mutex.Lock()
	members := m.members
	m.mutex.Unlock()

	return members
}
