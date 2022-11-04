package main

// ServiceCmd captures arguments provided by the Windows Service Control
// Manager.
type ServiceCmd struct {
	Server  string `kong:"optional,name='server',short='s',help='Guardian policy server host and port.'"`
	Passive bool   `kong:"optional,name='passive',short='p',help='Run passively without killing processes.'"`
	Debug   bool   `kong:"optional,name='verbose',short='v',help='Run with extra debug logging.'"`
}

// Config returns a custodian configuration for the command.
func (cmd *ServiceCmd) Config() EnforceConfig {
	return EnforceConfig{
		Server:  cmd.Server,
		Passive: cmd.Passive,
		Debug:   cmd.Debug,
	}
}
