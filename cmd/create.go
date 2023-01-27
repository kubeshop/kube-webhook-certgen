package cmd

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/kubeshop/kube-webhook-certgen/pkg/certs"
	"github.com/kubeshop/kube-webhook-certgen/pkg/k8s"
)

var create = &cobra.Command{
	Use:    "create",
	Short:  "Generate a ca and server cert+key and store the results in a secret 'secret-name' in 'namespace'",
	Long:   "Generate a ca and server cert+key and store the results in a secret 'secret-name' in 'namespace'",
	PreRun: configureLogging,
	RunE:   createCommand,
}

func createCommand(_ *cobra.Command, _ []string) error {
	k, err := k8s.New(newKubernetesClients(cfg.kubeconfig))
	if err != nil {
		return err
	}
	ca, err := k.GetCaFromSecret(cfg.secretName, cfg.namespace, cfg.caName)
	if err != nil {
		return err
	}
	if ca == nil {
		log.Infof("creating new secret %s/%s", cfg.namespace, cfg.secretName)
		newCa, newCert, newKey, err := certs.GenerateCerts(cfg.host)
		if err != nil {
			return err
		}
		ca = newCa
		if err := k.SaveCertsToSecret(
			context.Background(),
			cfg.secretName,
			cfg.namespace,
			cfg.caName,
			cfg.certName,
			cfg.keyName,
			ca,
			newCert,
			newKey,
		); err != nil {
			return err
		}
	} else {
		log.Infof("secret %s/%s already exists", cfg.namespace, cfg.secretName)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(create)
	create.Flags().StringVar(&cfg.host, "host", "", "Comma-separated hostnames and IPs to generate a certificate for")
	create.Flags().StringVar(&cfg.secretName, "secret-name", "", "Name of the secret where certificate information will be written")
	create.Flags().StringVar(&cfg.namespace, "namespace", "", "Namespace of the secret where certificate information will be written")
	create.Flags().StringVar(&cfg.certName, "cert-name", "cert", "Name of cert file in the secret")
	create.Flags().StringVar(&cfg.keyName, "key-name", "key", "Name of key file in the secret")
	create.MarkFlagRequired("host")
	create.MarkFlagRequired("secret-name")
	create.MarkFlagRequired("namespace")
}
