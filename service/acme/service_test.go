//go:build with_acme

package acme

import (
	"testing"

	"github.com/sagernet/sing-box/option"

	"github.com/caddyserver/certmagic"
	"github.com/stretchr/testify/require"
)

func TestConfigureCertificateSubjects(t *testing.T) {
	config := new(certmagic.Config)
	domains, err := configureCertificateSubjects(config, []string{" EXAMPLE.COM ", "www.example.com", "*.example.com", "example.com"})
	require.NoError(t, err)
	require.Equal(t, []string{"example.com", "www.example.com", "*.example.com"}, domains)
	require.Equal(t, map[string][]string{
		"example.com": {"www.example.com", "*.example.com"},
	}, config.SubjectToSANs)

	singleDomainConfig := new(certmagic.Config)
	domains, err = configureCertificateSubjects(singleDomainConfig, []string{"example.com", "EXAMPLE.COM"})
	require.NoError(t, err)
	require.Equal(t, []string{"example.com"}, domains)
	require.Nil(t, singleDomainConfig.SubjectToSANs)
	_, err = configureCertificateSubjects(config, []string{"example.com", " "})
	require.ErrorContains(t, err, "domain must not be empty")
}

func TestConfigurePreferredChains(t *testing.T) {
	t.Run("explicit", func(t *testing.T) {
		smallest := true
		issuer := new(certmagic.ACMEIssuer)
		configurePreferredChains(issuer, certmagic.ZeroSSLProductionCA, option.ACMECertificateProviderOptions{
			PreferredChain: &option.ACMEPreferredChainOptions{
				Smallest:       &smallest,
				RootCommonName: []string{"Preferred Root"},
				AnyCommonName:  []string{"Preferred Issuer"},
			},
		})
		require.Equal(t, certmagic.ChainPreference{
			Smallest:       &smallest,
			RootCommonName: []string{"Preferred Root"},
			AnyCommonName:  []string{"Preferred Issuer"},
		}, issuer.PreferredChains)
	})

	t.Run("letsencrypt ECC default", func(t *testing.T) {
		issuer := new(certmagic.ACMEIssuer)
		configurePreferredChains(issuer, certmagic.LetsEncryptProductionCA, option.ACMECertificateProviderOptions{})
		require.Equal(t, []string{"ISRG Root X2"}, issuer.PreferredChains.RootCommonName)
	})

	t.Run("letsencrypt RSA", func(t *testing.T) {
		issuer := new(certmagic.ACMEIssuer)
		configurePreferredChains(issuer, certmagic.LetsEncryptProductionCA, option.ACMECertificateProviderOptions{
			KeyType: option.ACMEKeyTypeRSA2048,
		})
		require.Empty(t, issuer.PreferredChains)
	})
}
