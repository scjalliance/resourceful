package main

// InstallCmd performs installation of the program and user service
// registration.
type InstallCmd struct {
	Server  string `kong:"optional,name='server',short='s',help='Guardian policy server host and port.'"`
	Passive bool   `kong:"optional,name='passive',short='p',help='Run passively without killing processes.'"`
	Debug   bool   `kong:"optional,name='verbose',short='v',help='Run with extra debug logging.'"`
}
