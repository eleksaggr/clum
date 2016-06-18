package clum

import "net"

// BroadcastWriter implements io.Writer. It sends a UDP datagram to a UDP multicast address.
type BroadcastWriter struct {
	addr string
}

// NewBroadcastWriter creates a new BroadcastWriter and set its Addr to addr. Should the addr not be a UDP multicast address, an error will be returned.
func NewBroadcastWriter(addr string) (w *BroadcastWriter, err error) {
	w = &BroadcastWriter{}
	if err := w.SetAddr(addr); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *BroadcastWriter) Write(b []byte) (n int, err error) {
	conn, err := net.Dial("udp", w.addr)
	if err != nil {
		return 0, err
	}

	return conn.Write(b)
}

// SetAddr sets the address of the BroadcastWriter. Should the address not be a UDP multicast address, an error will be returned.
func (w *BroadcastWriter) SetAddr(addr string) (err error) {
	if !net.ParseIP(addr).IsMulticast() {
		return err
	}
	w.addr = addr
	return nil
}

// Addr returns the address of the BroadcastWriter.
func (w *BroadcastWriter) Addr() string {
	return w.addr
}

// BroadcastReader implements io.Reader. It sends a UDP datagram to a UDP multicast address.
type BroadcastReader struct {
	addr string
}

// NewBroadcastReader creates a new BroadcastReader and set its Addr to addr. Should the addr not be a UDP multicast address, an error will be returned.
func NewBroadcastReader(addr string) (r *BroadcastReader, err error) {
	r = &BroadcastReader{}
	if err := r.SetAddr(addr); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *BroadcastReader) Read(b []byte) (n int, err error) {
	listener, err := net.Listen("udp", r.addr)
	if err != nil {
		return 0, err
	}

	conn, err := listener.Accept()
	if err != nil {
		return 0, err
	}

	return conn.Read(b)
}

// SetAddr sets the address of the BroadcastReader. Should the address not be a UDP multicast address, an error will be returned.
func (r *BroadcastReader) SetAddr(addr string) (err error) {
	if !net.ParseIP(addr).IsMulticast() {
		return err
	}
	r.addr = addr
	return nil
}

// Addr returns the address of the BroadcastReader.
func (r *BroadcastReader) Addr() string {
	return r.addr
}
