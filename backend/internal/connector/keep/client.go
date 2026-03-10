package keep

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const keepUserAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:78.0) Gecko/20100101 Firefox/78.0"

type KeepClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewKeepClient(baseURL string, httpClient *http.Client) *KeepClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &KeepClient{baseURL: baseURL, httpClient: httpClient}
}

func (c *KeepClient) Login(ctx context.Context, phone, password string) (string, error) {
	if phone == "" || password == "" {
		return "", errors.New("keep credential is empty")
	}
	form := url.Values{}
	form.Set("mobile", phone)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1.1/users/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", keepUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("keep login failed")
	}

	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Data.Token == "" {
		return "", errors.New("keep token empty")
	}
	return body.Data.Token, nil
}

func (c *KeepClient) FetchRunIDs(ctx context.Context, token, sportType string, lastDate int64) ([]string, int64, error) {
	q := url.Values{}
	q.Set("dateUnit", "all")
	q.Set("type", sportType)
	q.Set("lastDate", strconv.FormatInt(lastDate, 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/pd/v3/stats/detail?"+q.Encode(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", keepUserAgent)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, 0, errors.New("keep fetch list failed")
	}

	var body struct {
		Data struct {
			Records []struct {
				Logs []struct {
					Stats struct {
						ID         string `json:"id"`
						IsDoubtful bool   `json:"isDoubtful"`
					} `json:"stats"`
				} `json:"logs"`
			} `json:"records"`
			LastTimestamp int64 `json:"lastTimestamp"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, 0, err
	}

	ids := make([]string, 0)
	for _, record := range body.Data.Records {
		for _, log := range record.Logs {
			if !log.Stats.IsDoubtful {
				ids = append(ids, log.Stats.ID)
			}
		}
	}

	return ids, body.Data.LastTimestamp, nil
}

func (c *KeepClient) FetchRunDetail(ctx context.Context, token, sportType, runID string) (keepRunDetail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/pd/v3/"+sportType+"log/"+runID, nil)
	if err != nil {
		return keepRunDetail{}, err
	}
	req.Header.Set("User-Agent", keepUserAgent)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return keepRunDetail{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return keepRunDetail{}, errors.New("keep fetch detail failed")
	}

	var body keepRunDetail
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return keepRunDetail{}, err
	}
	return body, nil
}
