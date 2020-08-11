package main

import (
	"io/ioutil"
	certsv1a2 "kubernetes-go-cert-manager/crds/certificates/v1alpha2"
	issuerv1a2 "kubernetes-go-cert-manager/crds/issuers/v1alpha2"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Get local k8s config path.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		// Load config.
		kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
		b, err := ioutil.ReadFile(kubeconfigPath)
		if err != nil {
			return err
		}

		k8sProvider, err := kubernetes.NewProvider(ctx, "k8s", &kubernetes.ProviderArgs{
			Kubeconfig:                  pulumi.String(b),
			SuppressDeprecationWarnings: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Install cert-manager CRDs.
		_, err = yaml.NewConfigGroup(ctx, "cert-manager-crds", &yaml.ConfigGroupArgs{
			Files: []string{
				filepath.Join("manifests", "crds", "*.yaml"),
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// Install cert-manager.
		_, err = yaml.NewConfigFile(ctx, "cert-manager", &yaml.ConfigFileArgs{
			File: filepath.Join("manifests", "cert-manager.yaml"),
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// Create a SelfSigned Issuer.
		// https://cert-manager.io/docs/configuration/selfsigned/
		_, err = issuerv1a2.NewIssuer(ctx, "selfsigned-issuer", &issuerv1a2.IssuerArgs{
			Spec: &issuerv1a2.IssuerSpecArgs{
				SelfSigned: &issuerv1a2.IssuerSpecSelfSignedArgs{},
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// Issue a Certificate.
		// https://cert-manager.io/docs/usage/certificate/#creating-certificate-resources
		_, err = certsv1a2.NewCertificate(ctx, "example-com", &certsv1a2.CertificateArgs{
			Spec: certsv1a2.CertificateSpecArgs{
				// Secret names are always required.
				SecretName:  pulumi.String("example-com-tls"),
				Duration:    pulumi.String("2160h"), // 90d
				RenewBefore: pulumi.String("360h"),  // 15d
				Organization: pulumi.StringArray{
					pulumi.String("jetstack"),
				},
				CommonName:   pulumi.String("example.com"),
				IsCA:         pulumi.Bool(false),
				KeySize:      pulumi.Int(2048),
				KeyAlgorithm: pulumi.String("rsa"),
				KeyEncoding:  pulumi.String("pkcs1"),
				Usages: pulumi.StringArray{
					pulumi.String("server auth"),
					pulumi.String("client auth"),
				},
				DnsNames: pulumi.StringArray{
					pulumi.String("example.com"),
					pulumi.String("www.example.com"),
				},
				UriSANs: pulumi.StringArray{
					pulumi.String("spiffe://cluster.local/ns/default/sa/example"),
				},
				IpAddresses: pulumi.StringArray{
					pulumi.String("192.168.0.5"),
				},
				// Issuer references are always required.
				IssuerRef: certsv1a2.CertificateSpecIssuerRefArgs{
					Name:  pulumi.String("ca-issuer"),
					Kind:  pulumi.String("Issuer"),
					Group: pulumi.String("cert-manager.io"),
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
}
