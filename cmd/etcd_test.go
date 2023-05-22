package cmd

import (
	"flag"
	"testing"

	"github.com/gardener/etcd-wrapper/internal/types"
	. "github.com/onsi/gomega"
)

func TestAddEtcdFlags(t *testing.T) {
	g := NewWithT(t)
	tlsEnabled := flag.Bool("backup-restore-tls-enabled", types.DefaultBackupRestoreTLSEnabled, "Enables TLS for communicating with backup-restore container")
	hostPort := flag.String("backup-restore-host-port", types.DefaultBackupRestoreHostPort, "Host and Port to be used to connect to the backup-restore container")
	caCertBundlePath := flag.String("backup-restore-ca-cert-bundle-path", "", "File path of CA cert bundle to help establish TLS communication with backup-restore container")
	fs := flag.FlagSet{}
	AddEtcdFlags(&fs)
	g.Expect(backupRestoreConfig).ToNot(BeNil())
	g.Expect(backupRestoreConfig.TLSEnabled).To(Equal(*tlsEnabled))
	g.Expect(backupRestoreConfig.HostPort).To(Equal(*hostPort))
	g.Expect(*backupRestoreConfig.CaCertBundlePath).To(Equal(*caCertBundlePath))
}
