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

func GetIssuedCertificatesExampleUse() error {
	ip := &certificates.IntegrationProvider{
		RelationName: "certificates",
	}

	issuedCerts, err := ip.GetIssuedCertificates("certificates:0")
	if err != nil {
		return err
	}

	if len(issuedCerts) != 1 {
		return fmt.Errorf("expected 1 issued certificate, got %d", len(issuedCerts))
	}

	if string(issuedCerts[0].Certificate) != "example-cert" {
		return fmt.Errorf("expected certificate to be 'example-cert', got '%s'", issuedCerts[0].Certificate)
	}

	if issuedCerts[0].CertificateSigningRequest != "example-csr" {
		return fmt.Errorf("expected certificate signing request to be 'example-csr', got '%s'", issuedCerts[0].CertificateSigningRequest)
	}
	if issuedCerts[0].CA != "test-ca" {
		return fmt.Errorf("expected CA to be 'test-ca', got '%s'", issuedCerts[0].CA)
	}

	return nil
}

type ProviderCertificateRelationData struct {
	CA                        string `json:"ca"`
	Chain                     string `json:"chain"`
	CertificateSigningRequest string `json:"certificate_signing_request"`
	Certificate               string `json:"certificate"`
}

func TestGetIssuedCertificates(t *testing.T) {
	ctx := goopstest.Context{
		Charm:   GetIssuedCertificatesExampleUse,
		AppName: "test-charm",
		UnitID:  0,
	}

	providedCertificates := make([]ProviderCertificateRelationData, 0)

	providedCertificates = append(providedCertificates, ProviderCertificateRelationData{
		CA:                        "test-ca",
		Chain:                     `["example-cert","test-ca"]`,
		CertificateSigningRequest: "example-csr",
		Certificate:               "example-cert",
	})

	relationData, err := json.Marshal(providedCertificates)
	if err != nil {
		t.Fatalf("Failed to marshal provided certificates: %v", err)
	}

	certificatesRelation := &goopstest.Relation{
		Endpoint: "certificates",
		LocalAppData: goopstest.DataBag{
			"certificates": string(relationData),
		},
	}

	stateIn := &goopstest.State{
		Relations: []*goopstest.Relation{
			certificatesRelation,
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}

	if len(stateOut.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(stateOut.Relations))
	}
}

func SetRelationCertificateExampleUse() error {
	ip := &certificates.IntegrationProvider{
		RelationName: "certificates",
	}

	opts := &certificates.SetRelationCertificateOptions{
		RelationID:                "certificates:0",
		CA:                        "test-ca",
		Chain:                     []string{"example-cert", "test-ca"},
		CertificateSigningRequest: "example-csr",
		Certificate:               "example-cert",
	}

	err := ip.SetRelationCertificate(opts)
	if err != nil {
		return fmt.Errorf("failed to set relation certificate: %w", err)
	}

	return nil
}

func TestSetRelationCertificate(t *testing.T) {
	ctx := goopstest.Context{
		Charm: SetRelationCertificateExampleUse,
	}

	certificatesRelation := &goopstest.Relation{
		Endpoint: "certificates",
		LocalAppData: goopstest.DataBag{
			"certificates": `[]`,
		},
	}

	stateIn := &goopstest.State{
		Relations: []*goopstest.Relation{
			certificatesRelation,
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}

	if len(stateOut.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(stateOut.Relations))
	}

	relationData, ok := stateOut.Relations[0].LocalAppData["certificates"]
	if !ok {
		t.Fatal("expected 'certificates' key in relation data")
	}

	var certs []certificates.CertificateSigningRequestProviderAppRelationData
	err = json.Unmarshal([]byte(relationData), &certs)
	if err != nil {
		t.Fatalf("failed to unmarshal relation data: %v", err)
	}

	if len(certs) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(certs))
	}

	if certs[0].CA != "test-ca" || certs[0].CertificateSigningRequest != "example-csr" || certs[0].Certificate != "example-cert" {
		t.Fatalf("certificate data does not match expected values")
	}
}
