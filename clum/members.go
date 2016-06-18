package clum

import (
	"errors"
	"net"

	"github.com/nu7hatch/gouuid"
)

type Member struct {
	ID uuid.UUID

	host string
	Hops uint
}

type MemberList []Member

func (list *MemberList) Add(member *Member) (err error) {
	if member == nil {
		return errors.New("Member may not be nil.")
	}

	*list = append(*list, *member)
	return nil
}

func (list *MemberList) Remove(member *Member) (err error) {
	if member == nil {
		return errors.New("Member may not be nil.")
	}

	for i, m := range *list {
		if m.ID == member.ID {
			*list = append(*list[:i], *list[i+1:]...)
			break
		}
	}
	return nil
}

func (member *Member) Host() string {
	return member.host
}

func (member *Member) Addr() (host string, err error) {
	host, _, err = net.SplitHostPort(member.host)
	if err != nil {
		return "", err
	}
	return host, nil
}
