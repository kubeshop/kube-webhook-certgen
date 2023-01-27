package k8s

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/admissionregistration/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testWebhookName = "c7c95710-d8c3-4cc3-a2a8-8d2b46909c76"
	testSecretName  = "15906410-af2a-4f9b-8a2d-c08ffdd5e129"
	testNamespace   = "7cad5f92-c0d5-4bc9-87a3-6f44d5a5619d"
)

func genSecretData() (ca, cert, key []byte) {
	ca = make([]byte, 4)
	cert = make([]byte, 4)
	key = make([]byte, 4)
	_, _ = rand.Read(cert)
	_, _ = rand.Read(key)
	return
}

func newTestSimpleK8s() *K8s {
	return &K8s{
		clientset: fake.NewSimpleClientset(),
	}
}

func TestGetCaFromCertificate(t *testing.T) {
	t.Parallel()

	k := newTestSimpleK8s()

	ca, cert, key := genSecretData()

	caName := "ca.crt"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: testSecretName,
		},
		Data: map[string][]byte{caName: ca, "cert": cert, "key": key},
	}

	_, err := k.clientset.CoreV1().Secrets(testNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
	assert.NoError(t, err)

	retrievedCa, err := k.GetCaFromSecret(testSecretName, testNamespace, caName)
	assert.NoError(t, err)
	if !bytes.Equal(retrievedCa, ca) {
		t.Error("Was not able to retrieve CA information that was saved")
	}
}

func TestSaveCertsToSecret(t *testing.T) {
	t.Parallel()

	k := newTestSimpleK8s()

	ca, cert, key := genSecretData()

	ctx := context.Background()

	caName := "ca.crt"
	certName := "cert.tls"
	keyName := "key.tls"

	err := k.SaveCertsToSecret(ctx, testSecretName, testNamespace, caName, certName, keyName, ca, cert, key)
	assert.NoError(t, err)

	secret, _ := k.clientset.CoreV1().Secrets(testNamespace).Get(context.Background(), testSecretName, metav1.GetOptions{})

	if !bytes.Equal(secret.Data[certName], cert) {
		t.Error("'cert' saved data does not match retrieved")
	}

	if !bytes.Equal(secret.Data[keyName], key) {
		t.Error("'key' saved data does not match retrieved")
	}
}

func TestSaveThenLoadSecret(t *testing.T) {
	t.Parallel()

	k := newTestSimpleK8s()
	ca, cert, key := genSecretData()

	ctx := context.Background()

	caName := "ca.crt"
	certName := "cert.tls"
	keyName := "key.tls"

	err := k.SaveCertsToSecret(ctx, testSecretName, testNamespace, caName, certName, keyName, ca, cert, key)
	assert.NoError(t, err)
	retrievedCert, err := k.GetCaFromSecret(testSecretName, testNamespace, caName)
	assert.NoError(t, err)
	if !bytes.Equal(retrievedCert, ca) {
		t.Error("Was not able to retrieve CA information that was saved")
	}
}

func TestPatchWebhookConfigurations(t *testing.T) {
	t.Parallel()

	k := newTestSimpleK8s()

	ca, _, _ := genSecretData()

	ctx := context.Background()

	_, err := k.clientset.
		AdmissionregistrationV1beta1().
		MutatingWebhookConfigurations().
		Create(context.Background(), &v1beta1.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: testWebhookName,
			},
			Webhooks: []v1beta1.MutatingWebhook{{Name: "m1"}, {Name: "m2"}},
		}, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = k.clientset.
		AdmissionregistrationV1beta1().
		ValidatingWebhookConfigurations().
		Create(context.Background(), &v1beta1.ValidatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: testWebhookName,
			},
			Webhooks: []v1beta1.ValidatingWebhook{{Name: "v1"}, {Name: "v2"}},
		}, metav1.CreateOptions{})
	assert.NoError(t, err)

	err = k.PatchWebhookConfigurations(ctx, testWebhookName, ca, "fail", true, true, "v1beta1")
	assert.NoError(t, err)

	whmut, err := k.clientset.
		AdmissionregistrationV1beta1().
		MutatingWebhookConfigurations().
		Get(context.Background(), testWebhookName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	whval, err := k.clientset.
		AdmissionregistrationV1beta1().
		MutatingWebhookConfigurations().
		Get(context.Background(), testWebhookName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(whmut.Webhooks[0].ClientConfig.CABundle, ca) {
		t.Error("Ca retrieved from first mutating webhook configuration does not match")
	}
	if !bytes.Equal(whmut.Webhooks[1].ClientConfig.CABundle, ca) {
		t.Error("Ca retrieved from second mutating webhook configuration does not match")
	}
	if !bytes.Equal(whval.Webhooks[0].ClientConfig.CABundle, ca) {
		t.Error("Ca retrieved from first validating webhook configuration does not match")
	}
	if !bytes.Equal(whval.Webhooks[1].ClientConfig.CABundle, ca) {
		t.Error("Ca retrieved from second validating webhook configuration does not match")
	}
	if whmut.Webhooks[0].FailurePolicy == nil {
		t.Errorf("Expected first mutating webhook failure policy to be set to %s", "fail")
	}
	if whmut.Webhooks[1].FailurePolicy == nil {
		t.Errorf("Expected second mutating webhook failure policy to be set to %s", "fail")
	}
	if whval.Webhooks[0].FailurePolicy == nil {
		t.Errorf("Expected first validating webhook failure policy to be set to %s", "fail")
	}
	if whval.Webhooks[1].FailurePolicy == nil {
		t.Errorf("Expected second validating webhook failure policy to be set to %s", "fail")
	}
}
