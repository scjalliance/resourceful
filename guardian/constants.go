package guardian

import "time"

// DefaultPort is the default server port used for communication.
const DefaultPort = 5877

// DefaultHealthTimeout is the default amount of time a client will wait for
// an endpoint to respond to health queries.
const DefaultHealthTimeout = 200 * time.Millisecond
