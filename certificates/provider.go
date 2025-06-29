package certificates

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/gruyaume/goops"
)

type IntegrationProvider struct {
	RelationName string
}

type CertificateSigningRequestRequirerRelationData struct {
	CertificateSigningRequest string `json:"certificate_signing_request"`
	CA                        bool   `json:"ca"`
}

type CertificateSigningRequestProviderAppRelationData struct {
	CA                        string   `json:"ca"`
	Chain                     []string `json:"chain"`
	CertificateSigningRequest string   `json:"certificate_signing_request"`
	Certificate               string   `json:"certificate"`
}

type ProviderAppRelationData struct {
	Certificates string `json:"certificates"`
}

type CertificateSigningRequest struct {
	Raw                 string
	CommonName          string
	SansDNS             []string
	SansIP              []string
	SansOID             []string
	EmailAddress        string
	Organization        string
	OrganizationalUnit  string
	CountryName         string
	StateOrProvinceName string
	LocalityName        string
}

type RequirerCertificateRequest struct {
	RelationID                string
	CertificateSigningRequest CertificateSigningRequest
	IsCA                      bool
}

type Certificate struct {
	Raw                 string
	CommonName          string
	ExpiryTime          string
	ValidityStartTime   string
	IsCA                bool
	SansDNS             []string
	SansIP              []string
	SansOID             []string
	EmailAddress        string
	Organization        string
	OrganizationalUnit  string
	CountryName         string
	StateOrProvinceName string
	LocalityName        string
}

func (p *IntegrationProvider) GetOutstandingCertificateRequests() ([]RequirerCertificateRequest, error) {
	if p.RelationName == "" {
		return nil, fmt.Errorf("relation name is empty")
	}

	relationIDs, err := goops.GetRelationIDs(p.RelationName)
	if err != nil {
		return nil, fmt.Errorf("could not get relation IDs: %w", err)
	}

	requirerCertificateRequests := make([]RequirerCertificateRequest, 0)

	for _, relationID := range relationIDs {
		relationUnits, err := goops.ListRelationUnits(relationID)
		if err != nil {
			return nil, fmt.Errorf("could not list relation units: %w", err)
		}

		for _, unitID := range relationUnits {
			relationData, err := goops.GetUnitRelationData(relationID, unitID)
			if err != nil {
				return nil, fmt.Errorf("could not get relation data: %w", err)
			}

			csrJSON, ok := relationData["certificate_signing_requests"]
			if !ok {
				continue
			}

			var certificateSigningRequestsRelationData []CertificateSigningRequestRequirerRelationData

			err = json.Unmarshal([]byte(csrJSON), &certificateSigningRequestsRelationData)
			if err != nil {
				return nil, fmt.Errorf("could not unmarshal certificate signing requests: %w", err)
			}

			for _, csrRelationData := range certificateSigningRequestsRelationData {
				csrString := csrRelationData.CertificateSigningRequest

				csr, err := loadCertificateSigningRequest(csrString)
				if err != nil {
					return nil, fmt.Errorf("could not parse certificate signing request: %w", err)
				}

				requirerCertificateRequest := RequirerCertificateRequest{
					RelationID:                relationID,
					CertificateSigningRequest: csr,
					IsCA:                      csrRelationData.CA,
				}
				requirerCertificateRequests = append(requirerCertificateRequests, requirerCertificateRequest)
			}
		}
	}

	return requirerCertificateRequests, nil
}

// alreadyProvided checks if we've already pushed this CSR into the relation.
func (p *IntegrationProvider) AlreadyProvided(relationID string, csr string) bool {
	issued, err := p.GetIssuedCertificates(relationID)
	if err != nil {
		goops.LogWarningf("Could not get issued certificates: %v", err)
		return false
	}

	for _, pc := range issued {
		if pc.CertificateSigningRequest == csr {
			return true
		}
	}

	return false
}

type SetRelationCertificateOptions struct {
	RelationID                string
	CA                        string
	Chain                     []string
	CertificateSigningRequest string
	Certificate               string
}

func (p *IntegrationProvider) SetRelationCertificate(opts *SetRelationCertificateOptions) error {
	isLeader, err := goops.IsLeader()
	if err != nil {
		return fmt.Errorf("could not determine if unit is leader: %w", err)
	}

	if !isLeader {
		return fmt.Errorf("unit is not the leader and cannot set app relation data")
	}

	appData := []CertificateSigningRequestProviderAppRelationData{
		{
			CA:                        opts.CA,
			Chain:                     []string{},
			CertificateSigningRequest: opts.CertificateSigningRequest,
			Certificate:               opts.Certificate,
		},
	}

	appData[0].Chain = append(appData[0].Chain, opts.Chain...)

	appDataJSON, err := json.Marshal(appData)
	if err != nil {
		return fmt.Errorf("could not marshal app data: %w", err)
	}

	relationData := map[string]string{
		"certificates": string(appDataJSON),
	}

	err = goops.SetAppRelationData(opts.RelationID, relationData)
	if err != nil {
		return fmt.Errorf("could not set relation data: %w", err)
	}

	return nil
}

func loadCertificateSigningRequest(pemString string) (CertificateSigningRequest, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return CertificateSigningRequest{}, fmt.Errorf("failed to decode PEM block containing the certificate signing request")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return CertificateSigningRequest{}, fmt.Errorf("failed to parse certificate signing request: %w", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return CertificateSigningRequest{}, fmt.Errorf("CSR signature validation failed: %w", err)
	}

	var email string
	if len(csr.EmailAddresses) > 0 {
		email = csr.EmailAddresses[0]
	}

	var organization string
	if len(csr.Subject.Organization) > 0 {
		organization = csr.Subject.Organization[0]
	}

	var organizationalUnit string
	if len(csr.Subject.OrganizationalUnit) > 0 {
		organizationalUnit = csr.Subject.OrganizationalUnit[0]
	}

	var countryName string
	if len(csr.Subject.Country) > 0 {
		countryName = csr.Subject.Country[0]
	}

	var stateOrProvinceName string
	if len(csr.Subject.Province) > 0 {
		stateOrProvinceName = csr.Subject.Province[0]
	}

	var localityName string
	if len(csr.Subject.Locality) > 0 {
		localityName = csr.Subject.Locality[0]
	}

	var sansIP []string
	for _, ip := range csr.IPAddresses {
		sansIP = append(sansIP, ip.String())
	}

	return CertificateSigningRequest{
		Raw:                 pemString,
		CommonName:          csr.Subject.CommonName,
		SansDNS:             csr.DNSNames,
		SansIP:              sansIP,
		SansOID:             []string{}, // Not populated from the CSR directly.
		EmailAddress:        email,
		Organization:        organization,
		OrganizationalUnit:  organizationalUnit,
		CountryName:         countryName,
		StateOrProvinceName: stateOrProvinceName,
		LocalityName:        localityName,
	}, nil
}

func (p *IntegrationProvider) GetIssuedCertificates(relationID string) ([]*ProviderCertificate, error) {
	env := goops.ReadEnv()

	relationData, err := goops.GetAppRelationData(relationID, env.UnitName)
	if err != nil {
		return nil, fmt.Errorf("could not get relation data: %w", err)
	}

	if relationData == nil {
		return nil, fmt.Errorf("relation data is empty")
	}

	certificatesStr, ok := relationData["certificates"]
	if !ok {
		return nil, fmt.Errorf("relation data does not contain certificates")
	}

	var certificates []map[string]string

	err = json.Unmarshal([]byte(certificatesStr), &certificates)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal certificates: %w", err)
	}

	providerCertificates := make([]*ProviderCertificate, 0)
	for _, certData := range certificates {
		certificate := &ProviderCertificate{
			Certificate:               certData["certificate"],
			CA:                        certData["ca"],
			CertificateSigningRequest: certData["certificate_signing_request"],
		}
		var chain []string
		chainJSON, ok := certData["chain"]
		if !ok {
			return nil, fmt.Errorf("relation data does not contain chain")
		}
		if err := json.Unmarshal([]byte(chainJSON), &chain); err != nil {
			return nil, fmt.Errorf("could not unmarshal chain array: %w", err)
		}
		certificate.Chain = chain

		providerCertificates = append(providerCertificates, certificate)
	}

	return providerCertificates, nil
}
