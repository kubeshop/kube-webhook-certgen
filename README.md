[![Go Report Card](https://goreportcard.com/badge/github.com/kubeshop/kube-webhook-certgen)](https://goreportcard.com/report/github.com/kubeshop/kube-webhook-certgen)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubeshop/kube-webhook-certgen?sort=semver)](https://github.com/kubeshop/kube-webhook-certgen/releases/latest)
[![Docker Pulls](https://img.shields.io/docker/pulls/kubeshop/kube-webhook-certgen?color=blue)](https://hub.docker.com/r/kubeshop/kube-webhook-certgen/tags)

# Kubernetes webhook certificate generator and patcher

This is repo is a fork from [jet/kube-webhook-certgen](https://github.com/jet/kube-webhook-certgen) as the original project is no longer maintained.

This project will diverge from [jet/kube-webhook-certgen](https://github.com/jet/kube-webhook-certgen) and [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx/tree/main/images/kube-webhook-certgen)
as it has a roadmap of its own.

## Overview
Generates a CA and leaf certificate with a long (100y) expiration, then patches [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
by setting the `caBundle` field with the generated CA. 
Can optionally patch the hooks `failurePolicy` setting - useful in cases where a single Helm chart needs to provision resources
and hooks at the same time as patching.

The utility works in two parts, optimized to work better with the Helm provisioning process that leverages pre-install and post-install hooks to execute this as a Kubernetes job.

## Security Considerations
This tool may not be adequate in all security environments. If a more complete solution is required, you may want to 
seek alternatives such as [jetstack/cert-manager](https://github.com/jetstack/cert-manager)

## Command line options
```
Use this to create a ca and signed certificates and patch admission webhooks to allow for quick
                   installation and configuration of validating and admission webhooks.

Usage:
  kube-webhook-certgen [flags]
  kube-webhook-certgen [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  create      Generate a ca and server cert+key and store the results in a secret 'secret-name' in 'namespace'
  help        Help about any command
  patch       Patch a ValidatingWebhookConfiguration, MutatingWebhookConfiguration and CustomResourceDefinition
  version     Prints the CLI version information

Flags:
  -h, --help                help for kube-webhook-certgen
      --kubeconfig string   Path to kubeconfig file: e.g. ~/.kube/kind-config-kind
      --log-format string   Log format: text|json (default "json")
      --log-level string    Log level: panic|fatal|error|warn|info|debug|trace (default "info")

Use "kube-webhook-certgen [command] --help" for more information about a command.
```

### Create
```
Generate a ca and server cert+key and store the results in a secret 'secret-name' in 'namespace'

Usage:
  kube-webhook-certgen create [flags]

Flags:
      --ca-name string       Name of ca file in the secret (default "ca.crt")
      --cert-name string     Name of cert file in the secret (default "cert")
  -h, --help                 help for create
      --host string          Comma-separated hostnames and IPs to generate a certificate for
      --key-name string      Name of key file in the secret (default "key")
      --namespace string     Namespace of the secret where certificate information will be written
      --secret-name string   Name of the secret where certificate information will be written

Global Flags:
      --kubeconfig string   Path to kubeconfig file: e.g. ~/.kube/kind-config-kind
      --log-format string   Log format: text|json (default "json")
      --log-level string    Log level: panic|fatal|error|warn|info|debug|trace (default "info")
```

### Patch
```
Patch a ValidatingWebhookConfiguration and MutatingWebhookConfiguration 'webhook-name' and CustomResourceDefinitions by using the ca from 'secret-name' in 'namespace'

Usage:
  kube-webhook-certgen patch [flags]

Flags:
      --admission-registration-version string   admissionregistration.k8s.io api version (default "v1")
      --crd-api-groups string                   Comma-separated CustomResourceDefinition API Groups for which to patch the conversion webhook caBundle
      --crds string                             Comma-separated CustomResourceDefinition names for which to patch the conversion webhook caBundle
  -h, --help                                    help for patch
      --namespace string                        Namespace of the secret where certificate information will be read from
      --patch-failure-policy string             If set, patch the webhooks with this failure policy. Valid options are Ignore or Fail
      --patch-mutating                          If true, patch MutatingWebhookConfiguration (default true)
      --patch-validating                        If true, patch ValidatingWebhookConfiguration (default true)
      --secret-name string                      Name of the secret where certificate information will be read from
      --webhook-name string                     Name of ValidatingWebhookConfiguration and MutatingWebhookConfiguration that will be updated

Global Flags:
      --kubeconfig string   Path to kubeconfig file: e.g. ~/.kube/kind-config-kind
      --log-format string   Log format: text|json (default "json")
      --log-level string    Log level: panic|fatal|error|warn|info|debug|trace (default "info")
```

## Recent changes
* added support for CRDs
* Updated go version to v1.19
* Added support for `--admission-registration-version` flag which allows users to select which version of admissionregistration.k8s.io they want to use (v1 or v1beta1)
