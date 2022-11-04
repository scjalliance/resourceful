package main

// ListCmd lists running processes that match current policies.
type ListCmd struct {
	Server string `kong:"optional,name='server',short='s',help='Guardian policy server host and port.'"`
}
