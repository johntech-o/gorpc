// client and server settings
// follow the rules below
// DefaultMaxOpenConns > defaultIdleConns
// DefaultServerTimeout > 2*DefaultPingInterval
package gorpc

import (
	"time"
)

// client  setting
const (
	DefaultMaxOpenConns    = 100 // max conns
	DefaultMaxIdleConns    = 50  // max idle conns
	DefaultReadTimeout     = 30 * time.Second
	DefaultWriteTimeout    = 30 * time.Second
	DefaultConnectTimeout  = 30 * time.Second // default connect timeout
	DefaultPingInterval    = 50 * time.Second // conn idle beyond DefaultPingInterval  send a ping packet to server
	DefaultTimerGCInterval = time.Second
)

// server setting
const (
	DefaultServerIdleTimeout = time.Second * 300
	// client wait server to close the connection
	DefaultClientWaitResponseTimeout = DefaultServerIdleTimeout + time.Second*10

	DefaultServerTimerGCInterval = DefaultServerIdleTimeout / 2
)
