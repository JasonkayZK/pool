package pool

import (
	"time"
)

// generalConn is a wrapper around the connection
type generalConn struct {
	conn     interface{}
	t        time.Time
	unusable bool
}
