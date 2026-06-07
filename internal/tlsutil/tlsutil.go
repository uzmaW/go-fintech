package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

// Config holds TLS configuration parameters.
type Config struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	MinVersion string
	MutualTLS  bool
}

func minVersionFromString(v string) uint16 {
	switch strings.TrimSpace(v) {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3", "":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS13
	}
}

// LoadTLSConfig creates a tls.Config from the provided Config.
func LoadTLSConfig(cfg Config) (*tls.Config, error) {
	minVer := minVersionFromString(cfg.MinVersion)

	tlsCfg := &tls.Config{
		MinVersion: minVer,
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate/key pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file %s: %w", cfg.CAFile, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from %s", cfg.CAFile)
		}
		tlsCfg.RootCAs = caCertPool

		if cfg.MutualTLS {
			tlsCfg.ClientCAs = caCertPool
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		}
	} else if cfg.MutualTLS {
		return nil, fmt.Errorf("CA file is required when MutualTLS is enabled")
	}

	return tlsCfg, nil
}

// KafkaTLSConfig creates a tls.Config suitable for Kafka mTLS connections.
func KafkaTLSConfig(cfg Config) (*tls.Config, error) {
	if cfg.CAFile == "" {
		return nil, fmt.Errorf("CA file is required for Kafka TLS config")
	}

	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file %s: %w", cfg.CAFile, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", cfg.CAFile)
	}

	tlsCfg := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS13,
	}

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate/key pair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

// ServerTLSConfig creates a tls.Config for server-side connections with optional client certificate verification.
func ServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cfg := Config{
		CertFile:   certFile,
		KeyFile:    keyFile,
		CAFile:     caFile,
		MinVersion: "1.3",
		MutualTLS:  caFile != "",
	}
	return LoadTLSConfig(cfg)
}

// ClientTLSConfig creates a tls.Config for client-side mTLS connections.
func ClientTLSConfig(caFile string) (*tls.Config, error) {
	if caFile == "" {
		return nil, fmt.Errorf("CA file is required for client TLS config")
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file %s: %w", caFile, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
	}

	return &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS13,
	}, nil
}

// NetworkPolicy defines allowed ingress/gress traffic for a Kubernetes NetworkPolicy.
type NetworkPolicy struct {
	Name              string
	Namespace         string
	IngressPorts      []int
	EgressPorts       []int
	AllowedNamespaces []string
}

// GenerateNetworkPolicy produces a Kubernetes NetworkPolicy YAML string.
func GenerateNetworkPolicy(name, namespace string, ingressPorts, egressPorts []int, allowedNamespaces []string) string {
	var sb strings.Builder

	sb.WriteString("apiVersion: networking.k8s.io/v1\n")
	sb.WriteString("kind: NetworkPolicy\n")
	sb.WriteString("metadata:\n")
	sb.WriteString("  name: " + name + "\n")
	sb.WriteString("  namespace: " + namespace + "\n")
	sb.WriteString("spec:\n")
	sb.WriteString("  podSelector: {}\n")
	sb.WriteString("  policyTypes:\n")
	sb.WriteString("    - Ingress\n")
	sb.WriteString("    - Egress\n")

	if len(ingressPorts) > 0 {
		sb.WriteString("  ingress:\n")
		sb.WriteString("    - from:\n")
		if len(allowedNamespaces) > 0 {
			for _, ns := range allowedNamespaces {
				sb.WriteString("        - namespaceSelector:\n")
				sb.WriteString("            matchLabels:\n")
				sb.WriteString("              kubernetes.io/metadata.name: " + ns + "\n")
			}
		}
		sb.WriteString("      ports:\n")
		for _, port := range ingressPorts {
			sb.WriteString("        - protocol: TCP\n")
			sb.WriteString(fmt.Sprintf("          port: %d\n", port))
		}
	}

	if len(egressPorts) > 0 {
		sb.WriteString("  egress:\n")
		sb.WriteString("    - to:\n")
		if len(allowedNamespaces) > 0 {
			for _, ns := range allowedNamespaces {
				sb.WriteString("        - namespaceSelector:\n")
				sb.WriteString("            matchLabels:\n")
				sb.WriteString("              kubernetes.io/metadata.name: " + ns + "\n")
			}
		}
		sb.WriteString("      ports:\n")
		for _, port := range egressPorts {
			sb.WriteString("        - protocol: TCP\n")
			sb.WriteString(fmt.Sprintf("          port: %d\n", port))
		}
	}

	return sb.String()
}
