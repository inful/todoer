package main

// Constants for the application
const (
	FilePermissions  = 0o644
	ConfigDirName    = "todoer"
	ConfigFileName   = "config.toml"
	TemplateFileName = "template.md"

	// cmdProcess is the kong command string for the process subcommand.
	// Defined here so a rename of the positional args has one place to update.
	cmdProcess = "process <source-file> <target-file>"
)
