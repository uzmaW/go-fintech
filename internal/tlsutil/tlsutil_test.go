package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

func generateSelfSignedCert(t *testing.T) (certFile, keyFile string, cleanup func()) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certOut, err := os.CreateTemp("", "cert-*.pem")
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		certOut.Close()
		t.Fatalf("failed to encode cert: %v", err)
	}
	certOut.Close()

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}
	keyOut, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		keyOut.Close()
		t.Fatalf("failed to encode key: %v", err)
	}
	keyOut.Close()

	cleanup = func() {
		os.Remove(certOut.Name())
		os.Remove(keyOut.Name())
	}

	return certOut.Name(), keyOut.Name(), cleanup
}

func TestLoadTLSConfigEmpty(t *testing.T) {
	cfg := Config{}
	tlsCfg, err := LoadTLSConfig(cfg)
	if err != nil {
		t.Fatalf("LoadTLSConfig with empty config failed: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if tlsCfg.MinVersion != 0x0304 {
		t.Errorf("expected default min version TLS 1.3, got %v", tlsCfg.MinVersion)
	}
}

func TestServerTLSConfig(t *testing.T) {
	certFile, keyFile, cleanup := generateSelfSignedCert(t)
	defer cleanup()

	tlsCfg, err := ServerTLSConfig(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("ServerTLSConfig failed: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(tlsCfg.Certificates))
	}
	if tlsCfg.MinVersion != 0x0304 {
		t.Errorf("expected min version TLS 1.3, got %v", tlsCfg.MinVersion)
	}
}

func TestKafkaTLSConfig(t *testing.T) {
	certFile, keyFile, cleanup := generateSelfSignedCert(t)
	defer cleanup()

	// Use the cert itself as the CA for testing.
	tlsCfg, err := KafkaTLSConfig(Config{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   certFile,
	})
	if err != nil {
		t.Fatalf("KafkaTLSConfig failed: %v", err)
	}
	if tlsCfg.InsecureSkipVerify != false {
		t.Error("expected InsecureSkipVerify=false")
	}
	if tlsCfg.MinVersion != 0x0304 {
		t.Errorf("expected min version TLS 1.3, got %v", tlsCfg.MinVersion)
	}
}

func TestClientTLSConfig(t *testing.T) {
	certFile, _, cleanup := generateSelfSignedCert(t)
	defer cleanup()

	tlsCfg, err := ClientTLSConfig(certFile)
	if err != nil {
		t.Fatalf("ClientTLSConfig failed: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if tlsCfg.RootCAs == nil {
		t.Fatal("expected RootCAs pool to be set")
	}
}

func TestGenerateNetworkPolicy(t *testing.T) {
	policy := GenerateNetworkPolicy("test-policy", "default", []int{80, 443}, []int{5432}, []string{"frontend"})

	if policy == "" {
		t.Fatal("expected non-empty policy")
	}
	for _, want := range []string{
		"apiVersion: networking.k8s.io/v1",
		"kind: NetworkPolicy",
		"name: test-policy",
		"namespace: default",
		"port: 80",
		"port: 443",
		"port: 5432",
		"kubernetes.io/metadata.name: frontend",
	} {
		if !containsString(policy, want) {
			t.Errorf("policy missing %q", want)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
