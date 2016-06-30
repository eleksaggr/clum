package clum

import (
	"net"

	"github.com/nu7hatch/gouuid"
)

// Member is the information a node knowns about any other node in the cluster.
type Member struct {
	ID uuid.UUID

	Addr net.IP
	Port uint16
}
