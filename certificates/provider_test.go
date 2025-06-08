package certificates_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/gruyaume/charm-libraries/certificates"
	"github.com/gruyaume/goops/goopstest"
)

func GetOutstandingCertificateRequestsExampleUse() error {
	ip := &certificates.IntegrationProvider{
		RelationName: "certificates",
	}

	requirerRequests, err := ip.GetOutstandingCertificateRequests()
	if err != nil {
		return err
	}

	if len(requirerRequests) != 1 {
		return fmt.Errorf("expected 1 outstanding certificate requests, got %d", len(requirerRequests))
	}

	request0 := requirerRequests[0]
	if request0.RelationID != "certificates:0" {
		return fmt.Errorf("expected relation ID 'certificates:0', got '%s'", request0.RelationID)
	}

	if request0.CertificateSigningRequest.CommonName != "example.com" {
		return fmt.Errorf("expected CommonName 'example.com', got '%s'", request0.CertificateSigningRequest.CommonName)
	}

	if len(request0.CertificateSigningRequest.SansDNS) != 2 ||
		request0.CertificateSigningRequest.SansDNS[0] != "example.com" ||
		request0.CertificateSigningRequest.SansDNS[1] != "www.example.com" {
		return fmt.Errorf("expected SansDNS to contain 'example.com' and 'www.example.com', got %v", request0.CertificateSigningRequest.SansDNS)
	}

	if request0.IsCA {
		return fmt.Errorf("expected IsCA to be false, got true")
	}

	return nil
}

func generateCSR() (string, error) {
	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "example.com",
			Organization: []string{"Example Corp"},
		},
		DNSNames: []string{"example.com", "www.example.com"},
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("failed to generate private key: %v", err)
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create certificate request: %v", err)
	}
	csr := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	}))

	return csr, nil
}

func TestGetOutstandingCertificateRequests(t *testing.T) {
	ctx := goopstest.Context{
		Charm: GetOutstandingCertificateRequestsExampleUse,
	}

	csr, err := generateCSR()
	if err != nil {
		t.Fatalf("Failed to generate CSR: %v", err)
	}

	requestMap := map[string]interface{}{
		"certificate_signing_request": csr,
	}

	requestList := []map[string]interface{}{
		requestMap,
	}

	requestData, err := json.Marshal(requestList)
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}

	certificatesRelation := &goopstest.Relation{
		Endpoint: "certificates",
		RemoteUnitsData: map[goopstest.UnitID]goopstest.DataBag{
			"requirer/0": {
				"certificate_signing_requests": string(requestData),
			},
		},
	}

	stateIn := &goopstest.State{
		Relations: []*goopstest.Relation{
			certificatesRelation,
		},
	}

	_, err = ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}
}
