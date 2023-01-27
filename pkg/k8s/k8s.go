package k8s

import (
	"context"
	"strings"

	"github.com/kubeshop/kube-webhook-certgen/pkg/util"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"

	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type K8s struct {
	clientset           kubernetes.Interface
	aggregatorClientset clientset.Interface
	apiserverClientset  apiextensions.Interface
}

type AdmissionRegistrationVersion string

const (
	admissionRegistrationV1      AdmissionRegistrationVersion = "v1"
	admissionRegistrationV1beta1 AdmissionRegistrationVersion = "v1beta1"
)

func New(cs kubernetes.Interface, aggregatorCS clientset.Interface, apiextensionsCS apiextensions.Interface) (*K8s, error) {
	if cs == nil {
		return nil, errors.New("no kubernetes client given")
	}

	if aggregatorCS == nil {
		return nil, errors.New("no kubernetes aggregator client given")
	}

	if apiextensionsCS == nil {
		return nil, errors.New("no kubernetes apiextensions client given")
	}

	return &K8s{
		clientset:           cs,
		aggregatorClientset: aggregatorCS,
		apiserverClientset:  apiextensionsCS,
	}, nil
}

func (k8s *K8s) PatchCustomResourceDefinitions(
	ctx context.Context,
	crds, crdAPIGroup string,
	ca []byte,
) error {
	if crds != "" {
		if err := k8s.patchCRDs(ctx, crds, ca); err != nil {
			return err
		}
	}

	if crdAPIGroup != "" {
		if err := k8s.patchCRDAPIGroup(ctx, crdAPIGroup, ca); err != nil {
			return err
		}
	}

	log.Info("successfully patched CRD(s)")

	return nil
}

func (k8s *K8s) patchCRDs(ctx context.Context, crds string, ca []byte) error {
	log.Infof("patching CustomResourceDefinition objects '%s'", crds)

	splitted := strings.Split(crds, ",")
	for _, crd := range splitted {
		obj, err := k8s.apiserverClientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "error getting CustomResourceDefinition %s", crd)
		}
		if err := setCABundle(obj, ca); err != nil {
			log.Warnf("skip patching CustomResourceDefinition %s: %v", crd, err)
		}
		if _, err := k8s.apiserverClientset.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
			return errors.Wrapf(err, "error updating CustomResourceDefinition %s", obj.Name)
		}
		log.Infof("patched caBundle for CustomResourceDefinition %s", crd)
	}
	return nil
}

func (k8s *K8s) patchCRDAPIGroup(ctx context.Context, crdAPIGroups string, ca []byte) error {
	log.Infof("patching CustomResourceDefinition objects from API Groups '%s'", crdAPIGroups)

	list, err := k8s.apiserverClientset.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "error listing CustomResourceDefinition objects")
	}

	splitted := strings.Split(crdAPIGroups, ",")

	for i := range list.Items {
		crd := list.Items[i]
		if util.In(splitted, crd.Spec.Group) {
			if err := setCABundle(&crd, ca); err != nil {
				log.Warnf("skip patching CustomResourceDefinition %s: %v", crd.Name, err)
			}
			if _, err := k8s.apiserverClientset.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, &crd, metav1.UpdateOptions{}); err != nil {
				return errors.Wrapf(err, "error updating CustomResourceDefinition %s", crd.Name)
			}
			log.Infof("patched caBundle for CustomResourceDefinition %s", crd.Name)
		}
	}
	return nil
}

func setCABundle(crd *apiextensionsv1.CustomResourceDefinition, ca []byte) error {
	if crd.Spec.Conversion == nil {
		return errors.New("spec.conversion is not defined")
	}
	if crd.Spec.Conversion.Webhook == nil {
		return errors.New("spec.conversion.webhook is not defined")
	}
	if crd.Spec.Conversion.Webhook.ClientConfig == nil {
		return errors.New("spec.conversion.webhook.clientConfig is not defined")
	}
	crd.Spec.Conversion.Webhook.ClientConfig.CABundle = ca
	return nil
}

// PatchWebhookConfigurations will patch validatingWebhook and mutatingWebhook clientConfig configurations with
// the provided ca data. If failurePolicy is provided, patch all webhooks with this value.
func (k8s *K8s) PatchWebhookConfigurations(
	ctx context.Context,
	configurationNames string,
	ca []byte,
	failurePolicy string,
	patchMutating bool,
	patchValidating bool,
	version AdmissionRegistrationVersion,
) error {
	log.Infof(
		"patching webhook configurations '%s' mutating=%t, validating=%t, failurePolicy=%s",
		configurationNames, patchMutating, patchValidating, failurePolicy,
	)

	if patchValidating {
		if err := k8s.patchValidatingWebhookConfiguration(ctx, version, configurationNames, ca, failurePolicy); err != nil {
			return err
		}
	} else {
		log.Debug("validating hook patching not required")
	}

	if patchMutating {
		if err := k8s.patchMutatingWebhookConfiguration(ctx, version, configurationNames, ca, failurePolicy); err != nil {
			return err
		}
	} else {
		log.Debug("mutating hook patching not required")
	}

	log.Info("successfully patched hook(s)")

	return nil
}

func (k8s *K8s) patchValidatingWebhookConfiguration(
	ctx context.Context,
	version AdmissionRegistrationVersion,
	configurationNames string,
	ca []byte,
	failurePolicy string,
) error {
	switch version {
	case admissionRegistrationV1beta1:
		failurePolicyV1beta1 := admissionv1beta1.FailurePolicyType(failurePolicy)
		return k8s.patchValidatingWebhookConfigurationV1beta1(ctx, configurationNames, ca, &failurePolicyV1beta1)
	case admissionRegistrationV1:
		failurePolicyV1 := admissionv1.FailurePolicyType(failurePolicy)
		return k8s.patchValidatingWebhookConfigurationV1(ctx, configurationNames, ca, &failurePolicyV1)
	default:
		return errors.Errorf("invalid admissionregistration.k8s.io version: %s", version)
	}
}

func (k8s *K8s) patchValidatingWebhookConfigurationV1beta1(
	ctx context.Context,
	configurationNames string,
	ca []byte,
	failurePolicy *admissionv1beta1.FailurePolicyType,
) error {
	valHook, err := k8s.clientset.
		AdmissionregistrationV1beta1().
		ValidatingWebhookConfigurations().
		Get(ctx, configurationNames, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed getting admissionregistration.k8s.io/v1beta1 validating webhook")
	}

	for i := range valHook.Webhooks {
		h := &valHook.Webhooks[i]
		h.ClientConfig.CABundle = ca
		if *failurePolicy != "" {
			h.FailurePolicy = failurePolicy
		}
	}

	if _, err = k8s.clientset.AdmissionregistrationV1beta1().
		ValidatingWebhookConfigurations().
		Update(ctx, valHook, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed patching admissionregistration.k8s.io/v1beta1 validating webhook")
	}
	log.Info("patched admissionregistration.k8s.io/v1beta1 validating hook")

	return nil
}

func (k8s *K8s) patchValidatingWebhookConfigurationV1(
	ctx context.Context,
	configurationNames string,
	ca []byte,
	failurePolicy *admissionv1.FailurePolicyType,
) error {
	valHook, err := k8s.clientset.
		AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Get(ctx, configurationNames, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed getting admissionregistration.k8s.io/v1 validating webhook")
	}

	for i := range valHook.Webhooks {
		h := &valHook.Webhooks[i]
		h.ClientConfig.CABundle = ca
		if *failurePolicy != "" {
			h.FailurePolicy = failurePolicy
		}
	}

	if _, err = k8s.clientset.AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Update(ctx, valHook, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed patching admissionregistration.k8s.io/v1 validating webhook")
	}
	log.Info("patched admissionregistration.k8s.io/v1 validating hook")

	return nil
}

func (k8s *K8s) patchMutatingWebhookConfiguration(
	ctx context.Context,
	version AdmissionRegistrationVersion,
	configurationNames string,
	ca []byte,
	failurePolicy string,
) error {
	switch version {
	case admissionRegistrationV1beta1:
		failurePolicyV1beta1 := admissionv1beta1.FailurePolicyType(failurePolicy)
		return k8s.patchMutatingWebhookConfigurationV1beta1(ctx, configurationNames, ca, &failurePolicyV1beta1)
	case admissionRegistrationV1:
		failurePolicyV1 := admissionv1.FailurePolicyType(failurePolicy)
		return k8s.patchMutatingWebhookConfigurationV1(ctx, configurationNames, ca, &failurePolicyV1)
	default:
		return errors.Errorf("invalid admissionregistration.k8s.io version: %s", version)
	}
}

func (k8s *K8s) patchMutatingWebhookConfigurationV1beta1(
	ctx context.Context,
	configurationNames string,
	ca []byte,
	failurePolicy *admissionv1beta1.FailurePolicyType,
) error {
	mutHook, err := k8s.clientset.
		AdmissionregistrationV1beta1().
		MutatingWebhookConfigurations().
		Get(ctx, configurationNames, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed getting admissionregistration.k8s.io/v1beta1 mutating webhook")
	}

	for i := range mutHook.Webhooks {
		h := &mutHook.Webhooks[i]
		h.ClientConfig.CABundle = ca
		if *failurePolicy != "" {
			h.FailurePolicy = failurePolicy
		}
	}

	if _, err = k8s.clientset.AdmissionregistrationV1beta1().
		MutatingWebhookConfigurations().
		Update(ctx, mutHook, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed patching admissionregistration.k8s.io/v1beta1 mutating webhook")
	}
	log.Info("patched admissionregistration.k8s.io/v1beta1 mutating hook")

	return nil
}

func (k8s *K8s) patchMutatingWebhookConfigurationV1(
	ctx context.Context,
	configurationNames string,
	ca []byte,
	failurePolicy *admissionv1.FailurePolicyType,
) error {
	mutHook, err := k8s.clientset.
		AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Get(ctx, configurationNames, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed getting admissionregistration.k8s.io/v1 mutating webhook")
	}

	for i := range mutHook.Webhooks {
		h := &mutHook.Webhooks[i]
		h.ClientConfig.CABundle = ca
		if *failurePolicy != "" {
			h.FailurePolicy = failurePolicy
		}
	}

	if _, err = k8s.clientset.AdmissionregistrationV1().
		MutatingWebhookConfigurations().
		Update(ctx, mutHook, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(err, "failed patching admissionregistration.k8s.io/v1 mutating webhook")
	}
	log.Info("patched admissionregistration.k8s.io/v1 mutating hook")

	return nil
}

// GetCaFromSecret will check for the presence of a secret. If it exists, will return the content of the
// "ca" from the secret, otherwise will return nil.
func (k8s *K8s) GetCaFromSecret(secretName string, namespace string, caName string) ([]byte, error) {
	log.Debugf("getting secret '%s' in namespace '%s'", secretName, namespace)
	secret, err := k8s.clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.WithField("err", err).Infof("secret %s/%s does not exist", namespace, secretName)
			return nil, nil
		}
		log.WithField("err", err).Fatal("error getting secret")
	}

	data := secret.Data[caName]
	if data == nil {
		return nil, errors.Errorf("secret %s/%s does not contain '%s' key", namespace, secretName, caName)
	}
	log.Debug("got secret")
	return data, nil
}

// SaveCertsToSecret saves the provided ca, cert and key into a secret in the specified namespace.
func (k8s *K8s) SaveCertsToSecret(ctx context.Context, secretName, namespace, caName, certName, keyName string, ca, cert, key []byte) error {
	log.Debugf("saving to secret '%s' in namespace '%s'", secretName, namespace)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Data: map[string][]byte{caName: ca, certName: cert, keyName: key},
	}

	log.Debug("saving secret")
	_, err := k8s.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed creating secret %s/%s", namespace, secretName)
	}
	log.Debug("saved secret")

	return nil
}
