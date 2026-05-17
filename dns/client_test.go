package dns

import (
	"net"
	"testing"

	mDNS "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestFixedResponseRecordTypes(t *testing.T) {
	txtQuestion := mDNS.Question{
		Name:   "example.com.",
		Qtype:  mDNS.TypeTXT,
		Qclass: mDNS.ClassINET,
	}
	txtResponse := FixedResponseTXT(1, txtQuestion, []string{"ok"}, 60)
	require.Len(t, txtResponse.Answer, 1)
	require.Equal(t, mDNS.TypeTXT, txtResponse.Answer[0].Header().Rrtype)

	mxQuestion := mDNS.Question{
		Name:   "example.com.",
		Qtype:  mDNS.TypeMX,
		Qclass: mDNS.ClassINET,
	}
	mxResponse := FixedResponseMX(1, mxQuestion, []*net.MX{{Host: "mail.example.com.", Pref: 10}}, 60)
	require.Len(t, mxResponse.Answer, 1)
	require.Equal(t, mDNS.TypeMX, mxResponse.Answer[0].Header().Rrtype)
}
