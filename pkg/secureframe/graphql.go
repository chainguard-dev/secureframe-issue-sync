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

type report struct {
	Key   string
	Label string
}

type variables struct {
	// Used by DashboardTests
	SearchBy             *searchBy   `json:"searchBy,omitempty"`
	SearchKick           *searchKick `json:"searchkick,omitempty"`
	CurrentCompanyUserID string      `json:"current_company_user_id"`
	Page                 int         `json:"page,omitempty"`
	Limit                int         `json:"limit,omitempty"`

	// Used by Test
	ID   *string `json:"id,omitempty"`
	Pass bool    `json:"pass"`

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

type getCompanyTestResponse struct {
	Errors []Error            `json:"errors"`
	Data   getCompanyTestData `json:"data"`
}

type getCompanyTestData struct {
	Test Test `json:"getCompanyTest"`
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
	AssertionKeys    []string         `json:"assertionKeys"`
	AssertionResults AssertionResults `json:"assertionResults"`
	Title            string           `json:"title"`
	EvidenceType     string           `json:"evidenceType"`

	V2 TestV2 `json:"testV2"`
}

type Control struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Report      report `json:"report"`
}

type TestV2 struct {
	ID            string        `json:"id"`
	Key           string        `json:"key"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	AssertionKey  string        `json:"assertion_key"`
	AssertionData AssertionData `json:"assertionData"`
	ConditionKey  string        `json:"conditionKey"`

	ReportKeys []string `json:"reportKeys"`

	Controls []Control `json:"controls"`

	TestDomain       string `json:"testDomain"`
	TestFunction     string `json:"testFunction"`
	TestType         string `json:"testType"`
	ResourceCategory string `json:"resourceCategory"`

	DetailedRemediationSteps string `json:"detailedRemediationSteps"`
	RecommendedAction        string `json:"recommendedAction"`

	AssertionResults *AssertionResults `json:"assertionResults"`

	Status string `json:"status"`
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

func getCompanyTest(ctx context.Context, companyID string, token string, id string) (Test, error) {
	in := payload{
		OperationName: "getCompanyTest",
		Variables: variables{
			ID:                   &id,
			Page:                 1,
			Limit:                50,
			Pass:                 false,
			CurrentCompanyUserID: companyID,
		},
		Query: `query getCompanyTest($id: ID!, $page: Int, $limit: Int, $pass: Boolean) {
			getCompanyTest(id: $id) {
			  ...CompanyTestType
			  attachedEvidences {
				evidence {
				  id
				  files
				  fileNode {
					id
					discardedAt
					__typename
				  }
				  __typename
				}
				__typename
			  }
			  assertionResults(page: $page, limit: $limit, pass: $pass) {
				collection {
				  ...AssertionResultType
				  __typename
				}
				metadata {
				  currentPage
				  limitValue
				  totalCount
				  totalFailingAssertions(companyTestId: $id)
				  totalAssertions(companyTestId: $id)
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

		  fragment AssertionResultType on AssertionResult {
			id
			resourceable {
			  ...FailingResourceType
			  __typename
			}
			failMessage
			successMessage
			assertionKey
			pass
			data
			createdAt
			optional
			enabled
			disabledJustification
			__typename
		  }

		  fragment FailingResourceType on Resourceable {
			__typename
			... on CompanyUser {
			  id
			  companyUserName: name
			  imageUrl
			  __typename
			}
			... on Policy {
			  id
			  policyName: name
			  __typename
			}
			... on CloudResource {
			  id
			  owner {
				name
				__typename
			  }
			  vendor {
				name
				__typename
			  }
			  cloudResourceType
			  region
			  account
			  thirdPartyId
			  description
			  name
			  __typename
			}
			... on Device {
			  id
			  deviceName
			  serialNumber
			  __typename
			}
			... on Vendor {
			  id
			  vendorName: name
			  __typename
			}
			... on Repository {
			  id
			  repositoryName: name
			  __typename
			}
			... on CompanyUserVendor {
			  id
			  email
			  username
			  companyUser {
				id
				companyUserName: name
				imageUrl
				__typename
			  }
			  __typename
			}
			... on Evidence {
			  id
			  evidenceType
			  files
			  fileNode {
				id
				name
				__typename
			  }
			  __typename
			}
			... on CompanyTestAcknowledgement {
			  id
			  createdAt
			  acknowledgedBy {
				name
				__typename
			  }
			  __typename
			}
			... on PullRequest {
			  id
			  pullRequestName: name
			  __typename
			}
			... on ProductionBranch {
			  id
			  productionBranchName: name
			  __typename
			}
			... on CompanyControl {
			  id
			  control {
				key
				description
				__typename
			  }
			  __typename
			}
			... on Ticket {
			  id
			  title
			  openedAt
			  __typename
			}
		  }`,
	}

	out := &getCompanyTestResponse{}
	if err := query(ctx, token, in, out); err != nil {
		return Test{}, fmt.Errorf("request: %w", err)
	}

	if len(out.Errors) > 0 {
		return Test{}, fmt.Errorf("API returned errors: %+v", out.Errors)
	}

	log.Printf("out.Data: %+v", out.Data.Test)
	return out.Data.Test, nil
}

func GetTests(ctx context.Context, companyID string, token string, reportKey string) ([]Test, error) {
	tests, err := getCompanyTestV2s(ctx, companyID, token, reportKey)
	if err != nil {
		return nil, fmt.Errorf("get company test v2s: %w", err)
	}

	log.Printf("got data on %d tests ... filling in", len(tests))
	// The remaining bit of this function is a hack to fill in more information for failing tests.
	// If we had a properly documented GraphQL API, we could get everything in a single query.
	nts := []Test{}
	for x, t := range tests {
		if t.Pass || !t.Enabled {
			nts = append(nts, t)
			continue
		}

		log.Printf("[%d/%d] Fetching detailed data for failing test %s ...", x, len(tests), t.ID)
		mt, err := getCompanyTest(ctx, companyID, token, t.ID)
		if err != nil {
			return nil, fmt.Errorf("get company test (%s): %w", t.ID, err)
		}
		nts = append(nts, mt)
	}
	return nts, nil
}

func getCompanyTestV2s(ctx context.Context, companyID string, token string, reportKey string) ([]Test, error) {
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

	log.Printf("API returned %d results", len(out.Data.SearchCompanyTests.Data.Collection))
	// The API no longer appears to filter out report keys 🤷
	tests := []Test{}
	for _, t := range out.Data.SearchCompanyTests.Data.Collection {
		for _, k := range t.V2.ReportKeys {
			if k != reportKey {
				continue
			}
			tests = append(tests, t)
			continue
		}
	}

	return tests, nil
}
