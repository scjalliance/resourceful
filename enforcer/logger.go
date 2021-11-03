package enforcer

// A Logger is capable of logging events from an enforcement service.
type Logger interface {
	Log(Event)
}
