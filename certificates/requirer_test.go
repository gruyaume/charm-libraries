package certificates_test

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/gruyaume/charm-libraries/certificates"
	"github.com/gruyaume/goops/goopstest"
)

func RequestExampleUse() error {
	certRequest := certificates.CertificateRequestAttributes{
		CommonName: "example.com",
		SansDNS:    []string{"example.com", "www.example.com"},
		SansIP:     []string{"1.2.3.4"},
	}
	integration := &certificates.IntegrationRequirer{
		RelationName:       "certificates",
		CertificateRequest: certRequest,
	}

	err := integration.Request()
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	return nil
}

type RequirerRelationData struct {
	CertificateSigningRequest string `json:"certificate_signing_request"`
	CA                        string `json:"ca"`
}

func TestRequest(t *testing.T) {
	ctx := goopstest.Context{
		Charm: RequestExampleUse,
	}

	certificatesRelation := goopstest.Relation{
		Endpoint: "certificates",
	}
	stateIn := goopstest.State{
		Relations: []goopstest.Relation{
			certificatesRelation,
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}

	if ctx.CharmErr != nil {
		t.Fatalf("charm error: %v", ctx.CharmErr)
	}

	if len(stateOut.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(stateOut.Relations))
	}

	if len(stateOut.Relations[0].LocalUnitData) == 0 {
		t.Fatal("expected local unit data to be set, got empty")
	}

	relationData, ok := stateOut.Relations[0].LocalUnitData["certificate_signing_requests"]
	if !ok {
		t.Fatal("expected 'certificate_request' key in relation data")
	}

	var csrData []*RequirerRelationData

	if err := json.Unmarshal([]byte(relationData), &csrData); err != nil {
		t.Fatalf("failed to unmarshal relation data: %v", err)
	}

	if len(csrData) != 1 {
		t.Fatalf("expected 1 certificate signing request, got %d", len(csrData))
	}
	if csrData[0].CA != "false" {
		t.Fatalf("expected CA to be 'false', got '%s'", csrData[0].CA)
	}

	block, _ := pem.Decode([]byte(csrData[0].CertificateSigningRequest))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		t.Fatalf("failed to decode PEM block containing certificate request")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate signing request: %v", err)
	}

	if err := csr.CheckSignature(); err != nil {
		t.Fatalf("CSR signature validation failed: %v", err)
	}

	if csr.Subject.CommonName != "example.com" {
		t.Fatalf("expected CommonName to be 'example.com', got '%s'", csr.Subject.CommonName)
	}

	if len(csr.DNSNames) != 2 || csr.DNSNames[0] != "example.com" || csr.DNSNames[1] != "www.example.com" {
		t.Fatalf("expected DNSNames to be ['example.com', 'www.example.com'], got %v", csr.DNSNames)
	}

	if len(csr.IPAddresses) != 1 || csr.IPAddresses[0].String() != "1.2.3.4" {
		t.Fatalf("expected IPAddresses to be ['1.2.3.4'], got %v", csr.IPAddresses)
	}
}
