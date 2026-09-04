//go:build with_acme

package acme

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type serviceTestIssuer struct {
	reverse  bool
	requests [][]string
}

func (*serviceTestIssuer) IssuerKey() string { return "offline-audit" }

func (i *serviceTestIssuer) Issue(_ context.Context, csr *x509.CertificateRequest) (*certmagic.IssuedCertificate, error) {
	i.requests = append(i.requests, append([]string(nil), csr.DNSNames...))
	names := append([]string(nil), csr.DNSNames...)
	if i.reverse {
		slices.Reverse(names)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(int64(len(i.requests))),
		DNSNames:     names,
		IPAddresses:  csr.IPAddresses,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, leaf, leaf, csr.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &certmagic.IssuedCertificate{Certificate: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}, nil
}

// Exercise the service's subject mapping together with the selected certmagic
// dependency, reusing storage across two independent certificate caches.
func TestMultiSANServiceRestart(t *testing.T) {
	storage := &certmagic.FileStorage{Path: t.TempDir()}
	issuer := &serviceTestIssuer{reverse: true}
	for _, domains := range [][]string{{"EXAMPLE.COM", "example.com"}, {"EXAMPLE.COM", "www.example.com", "example.com"}} {
		var cfg *certmagic.Config
		cache := certmagic.NewCache(certmagic.CacheOptions{
			GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) { return cfg, nil },
			Logger:           zap.NewNop(),
		})
		base := certmagic.Config{Storage: storage, Issuers: []certmagic.Issuer{issuer}, DisableARI: true, Logger: zap.NewNop()}
		normalized, err := configureCertificateSubjects(&base, domains)
		require.NoError(t, err)
		cfg = certmagic.New(cache, base)
		t.Cleanup(cache.Stop)
		require.NoError(t, cfg.ManageSync(context.Background(), normalized))
		cert, err := cfg.CacheManagedCertificate(context.Background(), normalized[0])
		require.NoError(t, err)
		for _, domain := range normalized {
			require.NoError(t, cert.Leaf.VerifyHostname(domain))
		}
	}
	require.Len(t, issuer.requests, 2)
	require.ElementsMatch(t, []string{"example.com", "www.example.com"}, issuer.requests[1])
}
