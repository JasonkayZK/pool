package pool

// The pool interface
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error.
	Get() (interface{}, error)

	// Put puts the connection into the pool instead of closing it.
	Put(interface{}) error

	// CloseConn directly close the connection
	CloseConn(interface{}) error

	// ShutDown closes the pool and all its connections.
	// After ShutDown() the pool is no longer usable.
	ShutDown() error

	// Len returns the current number of connections of the pool.
	Len() int
}
