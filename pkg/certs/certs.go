package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// GenerateCerts venerates a ca with a leaf certificate and key and returns the ca, cert and key as PEM encoded slices.
func GenerateCerts(host string) (ca []byte, cert []byte, key []byte, err error) {
	notBefore := time.Now().Add(time.Minute * -5)
	notAfter := notBefore.Add(100 * 365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to generate serial number for CA certificate")
	}
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error creating root key for CA certificate")
	}

	rootTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		Subject:               pkix.Name{Organization: []string{"nil1"}},
	}
	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			rootTemplate.IPAddresses = append(rootTemplate.IPAddresses, ip)
		} else {
			rootTemplate.DNSNames = append(rootTemplate.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error creating CA certificate")
	}

	ca = encodeCert(derBytes)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error creating key for leaf certificate")
	}

	key, err = encodeKey(leafKey)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error encoding leaf key")
	}

	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error creating serial number for leaf certificate")
	}
	leafTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		Subject:               pkix.Name{Organization: []string{"nil2"}},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			leafTemplate.IPAddresses = append(leafTemplate.IPAddresses, ip)
		} else {
			leafTemplate.DNSNames = append(leafTemplate.DNSNames, h)
		}
	}

	derBytes, err = x509.CreateCertificate(rand.Reader, &leafTemplate, &rootTemplate, &leafKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error creating leaf certificate")
	}

	cert = encodeCert(derBytes)
	return ca, cert, key, nil
}

func encodeKey(key *ecdsa.PrivateKey) ([]byte, error) {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal ECDSA private key")
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b}), nil
}

func encodeCert(derBytes []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
}
