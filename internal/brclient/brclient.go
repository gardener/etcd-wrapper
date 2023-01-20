package brclient

type InitStatus int

const (
	InProgress InitStatus = iota
	Success
)

//go:generate stringer -type=InitStatus

type ValidationType string

const (
	SanityValidation ValidationType = "sanity" // validation_sanity
	FullValidation   ValidationType = "full"   // validation_full
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
