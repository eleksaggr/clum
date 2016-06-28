package clum

import (
	"net"

	"github.com/nu7hatch/gouuid"
)

type Member struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
}
