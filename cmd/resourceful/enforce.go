package main

// EnforceCmd runs a resourceful enforcement service interactively.
type EnforceCmd struct {
	Server  string `kong:"optional,name='server',short='s',help='Guardian policy server host and port.'"`
	Passive bool   `kong:"optional,name='passive',short='p',help='Run passively without killing processes.'"`
	Debug   bool   `kong:"optional,name='verbose',short='v',help='Run with extra debug logging.'"`
}

// Config returns an enforcer configuration for the command.
func (cmd *EnforceCmd) Config() EnforceConfig {
	return EnforceConfig{
		Server:  cmd.Server,
		Passive: cmd.Passive,
		Debug:   cmd.Debug,
	}
}

// EnforceConfig holds configuration for the enforce command.
type EnforceConfig struct {
	Server  string
	Passive bool
	Debug   bool
}
