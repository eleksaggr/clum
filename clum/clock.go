package clum

// LogicalClock represents an interface a logical clock must implement to be usable for the system.
type LogicalClock interface {
	Increment()
	Time() uint64
	Set(uint64)
}

// LamportClock is an implementation of the LogicalClock interface based on the idea of a lamport time.
type LamportClock struct {
	time uint64
}

// Increment increments the lamport time by 1 unit.
func (clock *LamportClock) Increment() {
	clock.time++
}

// Time returns the current lamport time.
func (clock *LamportClock) Time() uint64 {
	return clock.time
}

// Set sets the time of the clock to the specified newTime.
func (clock *LamportClock) Set(newTime uint64) {
	clock.time = newTime
}
