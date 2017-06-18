package leaseui

// Callback is a lease user interface callback function that processes the
// result of a user interaction.
type Callback func(result Result, err error)
