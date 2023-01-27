package cmd

import (
	"context"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"github.com/kubeshop/kube-webhook-certgen/pkg/k8s"
)

var patch = &cobra.Command{
	Use:    "patch",
	Short:  "Patch a ValidatingWebhookConfiguration, MutatingWebhookConfiguration and CustomResourceDefinition",
	Long:   "Patch a ValidatingWebhookConfiguration and MutatingWebhookConfiguration 'webhook-name' and CustomResourceDefinitions by using the ca from 'secret-name' in 'namespace'",
	PreRun: prePatchCommand,
	RunE:   patchCommand,
}

func prePatchCommand(cmd *cobra.Command, args []string) {
	configureLogging(cmd, args)
	if cfg.patchMutating == false && cfg.patchValidating == false {
		log.Fatal("patch-validating=false, patch-mutating=false. You must patch at least one kind of webhook, otherwise this command is a no-op")
		os.Exit(1)
	}
	switch cfg.patchFailurePolicy {
	case "":
		break
	case "Ignore":
	case "Fail":
		failurePolicy = cfg.patchFailurePolicy
	default:
		log.Fatalf("patch-failure-policy %s is not valid", cfg.patchFailurePolicy)
		os.Exit(1)
	}
}

func patchCommand(_ *cobra.Command, _ []string) error {
	k, err := k8s.New(newKubernetesClients(cfg.kubeconfig))
	if err != nil {
		return err
	}
	ca, err := k.GetCaFromSecret(cfg.secretName, cfg.namespace, cfg.caName)
	if err != nil {
		return err
	}

	if ca == nil {
		return errors.Errorf("no secret with '%s' in '%s'", cfg.secretName, cfg.namespace)
	}

	ctx := context.Background()

	if err := k.PatchWebhookConfigurations(
		ctx,
		cfg.webhookName,
		ca,
		failurePolicy,
		cfg.patchMutating,
		cfg.patchValidating,
		k8s.AdmissionRegistrationVersion(cfg.admissionRegistrationVersion),
	); err != nil {
		return err
	}

	if cfg.crds != "" || cfg.crdAPIGroups != "" {
		if err := k.PatchCustomResourceDefinitions(ctx, cfg.crds, cfg.crdAPIGroups, ca); err != nil {
			return err
		}
	}

	return nil
}

func newKubernetesClients(kubeconfig string) (kubernetes.Interface, aggregator.Interface, apiextensions.Interface) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.WithError(err).Fatal("error building kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).Fatal("error creating kubernetes client")
	}

	aggregatorClientset, err := aggregator.NewForConfig(config)
	if err != nil {
		log.WithError(err).Fatal("error creating kubernetes aggregator client")
	}

	apiextensionClientset, err := apiextensions.NewForConfig(config)

	return clientset, aggregatorClientset, apiextensionClientset
}

func init() {
	rootCmd.AddCommand(patch)
	patch.Flags().StringVar(&cfg.secretName, "secret-name", "", "Name of the secret where certificate information will be read from")
	patch.Flags().StringVar(&cfg.namespace, "namespace", "", "Namespace of the secret where certificate information will be read from")
	patch.Flags().StringVar(&cfg.webhookName, "webhook-name", "", "Name of ValidatingWebhookConfiguration and MutatingWebhookConfiguration that will be updated")
	patch.Flags().BoolVar(&cfg.patchValidating, "patch-validating", true, "If true, patch ValidatingWebhookConfiguration")
	patch.Flags().BoolVar(&cfg.patchMutating, "patch-mutating", true, "If true, patch MutatingWebhookConfiguration")
	patch.Flags().StringVar(&cfg.patchFailurePolicy, "patch-failure-policy", "", "If set, patch the webhooks with this failure policy. Valid options are Ignore or Fail")
	patch.Flags().StringVar(&cfg.admissionRegistrationVersion, "admission-registration-version", "v1", "admissionregistration.k8s.io api version")
	patch.Flags().StringVar(&cfg.crds, "crds", "", "Comma-separated CustomResourceDefinition names for which to patch the conversion webhook caBundle")
	patch.Flags().StringVar(&cfg.crdAPIGroups, "crd-api-groups", "", "Comma-separated CustomResourceDefinition API Groups for which to patch the conversion webhook caBundle")
	_ = patch.MarkFlagRequired("secret-name")
	_ = patch.MarkFlagRequired("namespace")
	_ = patch.MarkFlagRequired("webhook-name")
}
