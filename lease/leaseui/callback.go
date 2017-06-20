package leaseui

// Callback is a lease user interface callback function that processes the
// result of a user interaction.
type Callback func(t Type, result Result, err error)
