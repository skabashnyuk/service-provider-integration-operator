package config

type ZapConfiguration struct {
	ZapDevel           bool   `arg:"--zap-devel, env" default:"false" help:"Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn) Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)"`
	ZapEncoder         string `arg:"--zap-encoder, env" default:"" help:"Zap log encoding (‘json’ or ‘console’)"`
	ZapLogLevel        string `arg:"--zap-log-level, env" default:"" help:"Zap Level to configure the verbosity of logging"`
	ZapStackTraceLevel string `arg:"--zap-stacktrace-level, env" default:"" help:"Zap Level at and above which stacktraces are captured"`
	ZapTimeEncoding    string `arg:"--zap-time-encoding, env" default:"rfc3339" help:"one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'"`
}

type VaultConfiguration struct {
	VaultInsecureTLS bool `arg:"--vault-insecure-tls, env" default:"false" help:"Whether is allowed or not insecure vault tls connection."`
}

type ServiceProviderConfiguration struct {
	ConfigFile string `arg:"--config-file, env" default:"/etc/spi/config.yaml" help:"The location of the configuration file."`
}
type OperatorConfiguration struct {
	MetricsAddr          string `arg:"--metrics-bind-address, env" default:":8080" help:"The address the metric endpoint binds to."`
	ProbeAddr            string `arg:"--health-probe-bind-address, env" default:":8081" help:"The address the probe endpoint binds to."`
	EnableLeaderElection bool   `arg:"--leader-elect, env" default:"false" help:"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager."`
}
type SpiOperatorConfiguration struct {
	OperatorConfiguration
	ServiceProviderConfiguration
	VaultConfiguration
	ZapConfiguration
}
