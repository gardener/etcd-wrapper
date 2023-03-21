package brclient

// InitStatus is the status of initialisation as returned from backup-restore.
type InitStatus int

const (
	// InProgress indicates that the initialisation by backup-restore is in-progress.
	InProgress InitStatus = iota
	// Success indicates that the initialisation by backup-restore is successful.
	Success
)

//go:generate stringer -type=InitStatus

// ValidationType represents the type of validation that should be done of etcd DB during initialisation.
type ValidationType string

const (
	// SanityValidation only does sanity validation of the etcd DB.
	SanityValidation ValidationType = "sanity" // validation_sanity
	// FullValidation does a complete validation of the etcd DB.
	FullValidation ValidationType = "full" // validation_full
)

// BackupRestoreClient is a client to connect to the backup-restore HTTPs server.
type BackupRestoreClient interface {
	// GetInitializationStatus gets the latest state of initialization from the backup-restore.
	GetInitializationStatus() InitStatus
	// TriggerInitialization triggers the initialization on the backup-restore passing in the ValidationType.
	TriggerInitialization(validationType ValidationType) error
	// GetEtcdConfig gets the etcd configuration from the backup-restore, stores it into a file and returns the path to the file.
	GetEtcdConfig() string
}

type brClient struct {
}

// NewClient is a constructor which creates a new BackupRestoreClient.
func NewClient(caCertPath string) BackupRestoreClient {
	//TODO (Aaron): introduce a new command line flag cacert (there is already a similar flag passed to backup-restore). Intent is not to hard code the CA cert path in the code.
	panic("implement me")
}

func (c *brClient) GetInitializationStatus() InitStatus {
	panic("implement me")
}

func (c *brClient) TriggerInitialization(validationType ValidationType) error {
	panic("implement me")
}

func (c *brClient) GetEtcdConfig() string {
	panic("implement me")
}
