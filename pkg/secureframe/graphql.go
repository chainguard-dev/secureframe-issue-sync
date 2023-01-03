package secureframe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

var (
	defaultEndpoint = "https://app.secureframe.com/graphql"
)

type payload struct {
	OperationName string    `json:"operationName"`
	Variables     variables `json:"variables"`
	Query         string    `json:"query"`
}

type searchBy struct {
	ReportKeys []string `json:"reportKeys"`
	Enabled    bool     `json:"enabled"`
	Pass       bool     `json:"pass"`
}

type searchKick struct {
	Page    int    `json:"page"`
	PerPage int    `json:"perPage"`
	Query   string `json:"query"`
}

type variables struct {
	// Used by DashboardTests
	SearchBy             *searchBy   `json:"searchBy,omitempty"`
	SearchKick           *searchKick `json:"searchkick,omitempty"`
	CurrentCompanyUserID string      `json:"current_company_user_id"`
	Page                 int         `json:"page,omitempty"`
	Limit                int         `json:"limit,omitempty"`

	// Used by Test
	TestID *string `json:"testId,omitempty"`
	Pass   bool    `json:"pass"`

	Key string `json:"key"`
}

type Error struct {
	Message string
}

type getCompanyTestsData struct {
	SearchCompanyTests dataCollection `json:"searchCompanyTests"`
}

type dataCollection struct {
	Data getTests `json:"data"`
}

type getCompanyTestsResponse struct {
	Errors []Error             `json:"errors"`
	Data   getCompanyTestsData `json:"data"`
}

type getTests struct {
	Collection []Test `json:"collection"`
}

type AssertionData struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

type Resourceable struct {
	ID                string `json:"id"`
	DeviceName        string `json:"deviceName"`
	Email             string `json:"email"`
	RepositoryName    string `json:"repositoryName"`
	CompanyUserName   string `json:"companyUserName"`
	VendorName        string `json:"vendorName"`
	Account           string `json:"account"`
	CloudResourceType string `json:"cloudResourceType"`
	Region            string `json:"region"`
}

func ResourceID(r Resourceable) string {
	switch {
	case r.DeviceName != "":
		return r.DeviceName
	case r.Email != "":
		return r.Email
	case r.RepositoryName != "":
		return r.RepositoryName
	case r.CompanyUserName != "":
		return r.CompanyUserName
	case r.CompanyUserName != "":
		return r.CompanyUserName
	case r.VendorName != "":
		return r.VendorName
	case r.Region != "" && r.Account != "":
		return r.Account + "/" + r.Region
	case r.Account != "":
		return r.Account
	case r.Region != "":
		return r.Region
	default:
		return r.ID
	}
}

func AssertWork(a AssertionResult) string {
	work := strings.TrimSpace(a.FailMessage)
	// TODO: Don't hardcode this
	url := "https://app.secureframe.com/dashboard/incomplete-tests/soc2-beta"

	if work == "" {
		work = fmt.Sprintf("Upload evidence for %s", a.Data.Type)
	}

	if a.Resourceable != nil {
		return fmt.Sprintf("%s: %s", ResourceID(*a.Resourceable), work)
	}

	if strings.HasPrefix(work, "Upload") {
		work := strings.TrimRight(work, ".")
		return work + " to " + url
	}

	if strings.HasPrefix(work, "Select") && strings.HasSuffix(work, "Policy") {
		return work + " at " + url
	}

	return work
}

type AssertionResult struct {
	AssertionKey          string        `json:"assertionKey"`
	CreatedAt             string        `json:"createdAt"`
	Data                  AssertionData `json:"data"`
	DisabledJustification string        `json:"disabledJustification"`
	Enabled               bool          `json:"enabled"`
	Pass                  bool          `json:"pass"`
	Optional              bool          `json:"optional"`
	Resourceable          *Resourceable `json:"resourceable"`

	FailMessage string `json:"failMessage"`
}

type AssertionResults struct {
	Collection []AssertionResult `json:"collection"`
}

type Test struct {
	ID                            string   `json:"id"`
	Key                           string   `json:"key"`
	Description                   string   `json:"description"`
	Enabled                       bool     `json:"enabled"`
	Pass                          bool     `json:"pass"`
	DisabledJustification         string   `json:"disabledJustification"`
	PassedWithUploadJustification string   `json:"passedWithUploadJustification"`
	ReportKeys                    []string `json:"reportKeys"`
	Optional                      bool     `json:"optional"`

	// The following fields are only returned if getTest is called?
	AssertionKeys    []string          `json:"assertionKeys"`
	AssertionResults *AssertionResults `json:"assertionResults"`
	Title            string            `json:"title"`
	EvidenceType     string            `json:"evidenceType"`

	TestV2 TestV2 `json:"testV2"`
}

type TestV2 struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Title       string `json:"title"`

	ReportKeys []string `json:"reportKeys"`

	DetailedRemediationSteps string `json:"detailedRemediationSteps"`
}

func query(ctx context.Context, token string, in interface{}, out interface{}) error {
	payloadBytes, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	//	log.Printf("payload: %s", payloadBytes)
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequestWithContext(ctx, "POST", defaultEndpoint, body)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	log.Printf("POST'ing to %s with: \n%s", defaultEndpoint, payloadBytes)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Printf("response: %s", rb)

	if err := json.Unmarshal(rb, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	//	log.Printf("parsed response: %+v", out)
	return nil
}

func GetTests(ctx context.Context, companyID string, token string, reportKeys []string) ([]Test, error) {
	in := payload{
		OperationName: "getCompanyTestV2s",
		Variables: variables{
			SearchKick: &searchKick{
				Page:    1,
				PerPage: 1000,
				Query:   "*",
			},
			CurrentCompanyUserID: companyID,
		},
		Query: `query getCompanyTestV2s($searchkick: CompanyTestSearchkickInput) {
			searchCompanyTests(searchkick: $searchkick) {
			  data {
				collection {
				  ...CompanyTestType
				  __typename
				}
				metadata {
				  currentPage
				  limitValue
				  totalCount
				  totalPages
				  __typename
				}
				__typename
			  }
			  __typename
			}
		  }

		  fragment CompanyTestType on CompanyTest {
			id
			pass
			enabled
			exportable
			disabledJustification
			discardedAt
			passedWithUploadJustification
			updatedAt
			lastEvaluated
			lastPassedAt
			enabledFieldUpdatedById
			enabledFieldUpdatedByUser
			firstFailedAt
			nextDueDate
			owner {
			  id
			  name
			  imageUrl
			  __typename
			}
			testV2 {
			  ...TestV2Type
			  __typename
			}
			resourceableType
			status
			toleranceWindowSeconds
			testIntervalSeconds
			__typename
		  }

		  fragment TestV2Type on TestV2 {
			id
			key
			title
			description
			assertionKey
			assertionData
			conditionKey
			conditionData
			reportKeys
			testDomain
			testFunction
			resourceCategory
			recommendedAction
			detailedRemediationSteps
			additionalInfo
			global
			vendor {
			  id
			  name
			  __typename
			}
			author {
			  id
			  name
			  imageUrl
			  __typename
			}
			testType
			controls {
			  id
			  key
			  name
			  description
			  report {
				key
				label
				__typename
			  }
			  __typename
			}
			__typename
		  }
	`,
	}

	out := &getCompanyTestsResponse{}
	if err := query(ctx, token, in, out); err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}

	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("API returned errors: %+v", out.Errors)
	}

	needsK := map[string]bool{}
	for _, k := range reportKeys {
		needsK[k] = true
	}

	log.Printf("API returned %d results", len(out.Data.SearchCompanyTests.Data.Collection))
	// The API no longer appears to filter out report keys 🤷
	tests := []Test{}
	for _, t := range out.Data.SearchCompanyTests.Data.Collection {
		for _, k := range t.TestV2.ReportKeys {
			if !needsK[k] {
				continue
			}
			tests = append(tests, t)
			continue
		}
	}

	return tests, nil
}
