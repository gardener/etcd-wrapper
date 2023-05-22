package types

const (
	// DefaultBackupRestoreTLSEnabled defines the default TLS state of the application
	DefaultBackupRestoreTLSEnabled = false
	// DefaultBackupRestoreHostPort defines the default sidecar host and port
	DefaultBackupRestoreHostPort = ":8080"
	// DefaultExitCodeFilePath defines the default file path for the file that stores the exit code of the previous run
	DefaultExitCodeFilePath = "/var/etcd/data/exit_code"
)
