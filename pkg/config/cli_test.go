package config

import (
	"bytes"
	"github.com/alexflint/go-arg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_cliArgs_MarshalLogObject(t *testing.T) {
	expectedHelp := `
Usage: operator [--metrics-bind-address METRICS-BIND-ADDRESS] [--health-probe-bind-address HEALTH-PROBE-BIND-ADDRESS] [--leader-elect] [--config-file CONFIG-FILE] [--vault-insecure-tls] [--zap-devel] [--zap-encoder ZAP-ENCODER] [--zap-log-level ZAP-LOG-LEVEL] [--zap-stacktrace-level ZAP-STACKTRACE-LEVEL] [--zap-time-encoding ZAP-TIME-ENCODING]

Options:
  --metrics-bind-address METRICS-BIND-ADDRESS
                         The address the metric endpoint binds to. [default: :8080, env: METRICSADDR]
  --health-probe-bind-address HEALTH-PROBE-BIND-ADDRESS
                         The address the probe endpoint binds to. [default: :8081, env: PROBEADDR]
  --leader-elect         Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager. [default: false, env: ENABLELEADERELECTION]
  --config-file CONFIG-FILE
                         The location of the configuration file. [default: /etc/spi/config.yaml, env: CONFIGFILE]
  --vault-insecure-tls   Whether is allowed or not insecure vault tls connection. [default: false, env: VAULTINSECURETLS]
  --zap-devel            Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn) Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error) [default: false, env: ZAPDEVEL]
  --zap-encoder ZAP-ENCODER
                         Zap log encoding (‘json’ or ‘console’) [env: ZAPENCODER]
  --zap-log-level ZAP-LOG-LEVEL
                         Zap Level to configure the verbosity of logging [env: ZAPLOGLEVEL]
  --zap-stacktrace-level ZAP-STACKTRACE-LEVEL
                         Zap Level at and above which stacktraces are captured [env: ZAPSTACKTRACELEVEL]
  --zap-time-encoding ZAP-TIME-ENCODING
                         one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano' [default: rfc3339, env: ZAPTIMEENCODING]
  --help, -h             display this help and exit
`
	args := SpiOperatorConfiguration{}
	p, err := arg.NewParser(arg.Config{Program: "operator"}, &args)
	require.NoError(t, err)
	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

}
