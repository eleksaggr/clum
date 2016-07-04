package clum

import (
	"errors"
	"net"
	"strconv"
	"sync"

	"github.com/nu7hatch/gouuid"
)

type updateType uint

const (
	Add updateType = iota
	Remove
)

// Member is the information a node knowns about any other node in the cluster.
type Member struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
}

func (m *Member) Host() string {
	portStr := strconv.Itoa(int(m.Port))
	return net.JoinHostPort(m.Addr.String(), portStr)
}

// MemberList is a thread-safe structure that manages a list of members.
type MemberList struct {
	members []*Member
	mutex   *sync.Mutex

	observer *Node
}

func NewMemberList() (m *MemberList) {
	m = &MemberList{
		members: make([]*Member, 0),
		mutex:   &sync.Mutex{},
	}
	return m
}

func (m *MemberList) Subscribe(observer *Node) (err error) {
	if observer == nil {
		return errors.New("Empty node may not subscribe to the memberlist.")
	}
	m.observer = observer
	return nil
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
		m.notify(member, Add)
	}
	m.mutex.Unlock()

	return nil // Adding a member that's already in the list, is not an error.
}

func (m *MemberList) SetMembers(members []*Member) {
	m.mutex.Lock()
	m.members = members
	m.mutex.Unlock()
}

// Remove removes a member from the list, if he's in there. Should he not exist, nothing happens.
func (m *MemberList) Remove(member *Member) (err error) {
	if member == nil {
		return errors.New("Member may not be nil.")
	}
	m.mutex.Lock()
	for i, mem := range m.members {
		if mem.ID == member.ID {
			m.members = append(m.members[:i], m.members[i+1:]...)
			m.notify(member, Remove)
		}
	}
	m.mutex.Unlock()
	return nil
}

// Members returns a list of members currently in the list.
func (m *MemberList) Members() []*Member {
	m.mutex.Lock()
	members := m.members
	m.mutex.Unlock()

	return members
}

func (m *MemberList) Length() int {
	return len(m.members)
}

func (m *MemberList) notify(origin *Member, update updateType) {
	go m.observer.update(origin, update)
}
