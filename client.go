package airtable

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/grokify/oauth2more"
	"github.com/grokify/simplego/net/urlutil"
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
		client.HTTPClient = oauth2more.NewClientAuthzTokenSimple(
			oauth2more.Bearer.String(), client.token)
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
		return "", fmt.Errorf("Airtable API Client: Non-1 record returned [%d]", len(recs.Records))
	}
	return recs.Records[0].ID, nil
}

func (client *AirtableAPIClient) ListRecords(opts *ListOpts, res interface{}) ([]byte, *http.Response, error) {
	resp, err := client.listRecordsRaw(opts)
	if err != nil {
		return []byte(""), resp, err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
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
		return nil, errors.New("Airtable API Client Error: No HTTP Client Set")
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
