package types

const (
	// DefaultTLSEnabled defines the default TLS state of the application
	DefaultTLSEnabled = false
	// DefaultSideCarHostPort defines the default sidecar host and port
	DefaultSideCarHostPort = ":8080"
	// DefaultExitCodeFilePath defines the default file path for the file that stores the exit code of the previous run
	DefaultExitCodeFilePath = "/var/etcd/data/exit_code"
)
