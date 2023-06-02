package cmd

import (
	"flag"
	"testing"

	. "github.com/onsi/gomega"
)

func TestAddEtcdFlags(t *testing.T) {
	g := NewWithT(t)
	expectedBRHostPort := "etcd-main-local:8080"
	expectedBRCACertPath := "/var/etcd/ssl/ca/bundle.crt"
	expectedETCDServerName := "etcd-main-local"
	expectedETCDClientCertPath := "/var/etcd/ssl/client/tls.crt"
	expectedETCDClientKeyPath := "/var/etcd/ssl/client/tls.key"
	expectedETCDReadyTimeout := "2m0s"
	args := []string{
		"-backup-restore-tls-enabled=true",
		"-backup-restore-host-port", expectedBRHostPort,
		"-backup-restore-ca-cert-bundle-path", expectedBRCACertPath,
		"-etcd-server-name", expectedETCDServerName,
		"-etcd-client-cert-path", expectedETCDClientCertPath,
		"-etcd-client-key-path", expectedETCDClientKeyPath,
		"-etcd-ready-timeout", expectedETCDReadyTimeout,
	}
	fs := flag.NewFlagSet("testutil", flag.ContinueOnError)
	AddEtcdFlags(fs)
	g.Expect(fs.Parse(args)).To(Succeed())
	g.Expect(config).ToNot(BeNil())
	g.Expect(config.BackupRestore.TLSEnabled).To(BeTrue())
	g.Expect(config.BackupRestore.HostPort).To(Equal(expectedBRHostPort))
	g.Expect(config.BackupRestore.CaCertBundlePath).To(Equal(expectedBRCACertPath))
	g.Expect(config.EtcdClientTLS.ServerName).To(Equal(expectedETCDServerName))
	g.Expect(config.EtcdClientTLS.CertPath).To(Equal(expectedETCDClientCertPath))
	g.Expect(config.EtcdClientTLS.KeyPath).To(Equal(expectedETCDClientKeyPath))
	g.Expect(etcdReadyTimeout.String()).To(Equal(expectedETCDReadyTimeout))
}
