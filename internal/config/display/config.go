package display

type GatewayCfg struct {
	ProxyURL  string                        `toml:"proxy_url" yaml:"proxy_url"`
	ProxyKey  string                        `toml:"proxy_key" yaml:"proxy_key"`
	Platforms map[string]GatewayPlatformCfg `toml:"platforms" yaml:"platforms"`
}

type GatewayPlatformCfg struct {
	GatewayRestartNotification *bool `toml:"gateway_restart_notification" yaml:"gateway_restart_notification"`
}

type DisplayCfg struct {
	Language                 string                        `toml:"language" yaml:"language"`
	Personality              string                        `toml:"personality" yaml:"personality"`
	ToolProgress             string                        `toml:"tool_progress" yaml:"tool_progress"`
	ToolProgressCommand      bool                          `toml:"tool_progress_command" yaml:"tool_progress_command"`
	ShowReasoning            bool                          `toml:"show_reasoning" yaml:"show_reasoning"`
	Streaming                bool                          `toml:"streaming" yaml:"streaming"`
	BellOnComplete           bool                          `toml:"bell_on_complete" yaml:"bell_on_complete"`
	Compact                  bool                          `toml:"compact" yaml:"compact"`
	CleanupProgress          bool                          `toml:"cleanup_progress" yaml:"cleanup_progress"`
	InterimAssistantMessages bool                          `toml:"interim_assistant_messages" yaml:"interim_assistant_messages"`
	BackgroundProcessNotifs  string                        `toml:"background_process_notifications" yaml:"background_process_notifications"`
	BusyInputMode            string                        `toml:"busy_input_mode" yaml:"busy_input_mode"`
	Platforms                map[string]DisplayPlatformCfg `toml:"platforms" yaml:"platforms"`
}

type DisplayPlatformCfg struct {
	ToolProgress string `toml:"tool_progress" yaml:"tool_progress"`
}
