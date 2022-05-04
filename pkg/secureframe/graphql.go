package secureframe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type getDashboardTestsPayload struct {
	OperationName string         `json:"operationName"`
	Variables     inputVariables `json:"variables"`
	Query         string         `json:"query"`
}

type searchBy struct {
	ReportKeys []string `json:"reportKeys"`
	Enabled    bool     `json:"enabled"`
	Pass       bool     `json:"pass"`
}

type inputVariables struct {
	SearchBy             searchBy `json:"searchBy"`
	CurrentCompanyUserID string   `json:"current_company_user_id"`
}

type Error struct {
	Message string
}

type getDashboardTestsResponse struct {
	Errors []Error
	Data   getDashboardTestsData `json:"data"`
}

type getDashboardTestsData struct {
	GetTests getTests `json:"getTests"`
}

type getTests struct {
	Collection []Test `json:"collection"`
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

	Title       string `json:"title"`
	FailMessage string
}

// DashboardTests returns data from SecureFrame getDashboardTests GraphQL API.
func DashboardTests(ctx context.Context, reportKeys []string, companyID string, token string) ([]Test, error) {
	data := getDashboardTestsPayload{
		OperationName: "getDashboardTests",
		Variables: inputVariables{
			SearchBy: searchBy{
				ReportKeys: reportKeys,
				Enabled:    true,
			},
			CurrentCompanyUserID: companyID,
		},
		Query: `query getDashboardTests($searchBy: TestFilterInput) {
  getTests(searchBy: $searchBy) {
    collection {
      id
      key
      description
      enabled
      pass
      disabledJustification
      passedWithUploadJustification
      reportKeys
      companyTest {
        id
        __typename
      }
      __typename
    }
    __typename
  }
}`,
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	log.Printf("payload: %s", payloadBytes)
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://app.secureframe.com/graphql", body)
	if err != nil {
		return nil, fmt.Errorf("post: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	gd := getDashboardTestsResponse{}
	if err := json.Unmarshal(rb, &gd); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if len(gd.Errors) > 0 {
		return nil, fmt.Errorf("API returned errors: %+v", gd.Errors)
	}

	return gd.Data.GetTests.Collection, nil
}
