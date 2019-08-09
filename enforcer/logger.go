// +build windows

package enforcer

// A Logger is capable of logging enforcement messages.
type Logger interface {
	Printf(string, ...interface{})
}

/*
type EventLogger interface {
	Log(event)
}
*/
