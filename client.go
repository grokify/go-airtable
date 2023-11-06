package airtable

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/grokify/goauth/authutil"
	"github.com/grokify/mogo/net/urlutil"
)

const AirtableAPIBaseURL = "https://api.airtable.com/v0/"

type AirtableAPIClient struct {
	token      string
	BaseID     string
	TableName  string
	HTTPClient *http.Client
}

func NewAirtableAPIClient(token, baseID, tableName string) *AirtableAPIClient {
	client := &AirtableAPIClient{
		BaseID:    strings.TrimSpace(baseID),
		TableName: tableName}
	client.SetToken(token)
	return client
}

func (client *AirtableAPIClient) SetToken(token string) {
	client.token = strings.TrimSpace(token)
	if len(client.token) > 0 {
		client.HTTPClient = authutil.NewClientAuthzTokenSimple(
			authutil.Bearer.String(), client.token)
	}
}

type ListOpts struct {
	Fields       []string
	FilterIn     map[string]string
	FilterOut    map[string]string
	View         string
	ViewGridView bool
}

func (client *AirtableAPIClient) GetRecordID(opts *ListOpts) (string, error) {
	recs := TableGenericRecordList{}
	_, _, err := client.ListRecords(opts, &recs)
	if err != nil {
		return "", err
	}
	if len(recs.Records) != 1 {
		return "", fmt.Errorf("airtable.AirtableAPIClient `GetRecordID()` non-1 record returned, records: [%d]", len(recs.Records))
	}
	return recs.Records[0].ID, nil
}

func (client *AirtableAPIClient) ListRecords(opts *ListOpts, res any) ([]byte, *http.Response, error) {
	resp, err := client.listRecordsRaw(opts)
	if err != nil {
		return []byte(""), resp, err
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return bytes, resp, err
	}
	if res != nil {
		return bytes, resp, json.Unmarshal(bytes, res)
	}
	return bytes, resp, nil
}

func (client *AirtableAPIClient) listRecordsRaw(opts *ListOpts) (*http.Response, error) {
	apiURL := urlutil.JoinAbsolute(
		AirtableAPIBaseURL,
		client.BaseID,
		strings.Join(strings.Split(client.TableName, " "), "%20"))
	if opts != nil {
		qry := url.Values{}
		for _, field := range opts.Fields {
			qry.Add("field[]", field)
		}
		for k, v := range opts.FilterIn {
			val := fmt.Sprintf("{%s} = '%s'", k, v)
			qry.Add("filterByFormula", val)
		}
		for k, v := range opts.FilterOut {
			val := fmt.Sprintf("{%s} = '%s'", k, v)
			qry.Add("filterByFormula", val)
		}
		if opts.ViewGridView {
			qry.Add("view", "Grid view")
		}
		if len(qry) > 0 {
			apiURL += "?" + qry.Encode()
		}
	}
	if client.HTTPClient == nil {
		return nil, errors.New("airtable API client error: no HTTP client set")
	}
	return client.HTTPClient.Get(apiURL)
}

type TableGenericRecordList struct {
	Records []TableGenericRecord `json:"records"`
	Offset  string               `json:"offset"`
}

type TableGenericRecord struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}
