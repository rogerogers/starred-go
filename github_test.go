package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type mockTransport struct {
	roundTripFn func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFn(req)
}

func TestGitHubClient_FetchStarredRepos(t *testing.T) {
	client := NewGitHubClient("test-token")

	page := 0
	client.httpClient.Transport = &mockTransport{
		roundTripFn: func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "Bearer test-token" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewBufferString("Unauthorized")),
					Header:     make(http.Header),
				}, nil
			}

			var payload gqlPayload
			_ = json.NewDecoder(req.Body).Decode(&payload)
			after, _ := payload.Variables["after"].(string)

			var resp map[string]interface{}
			if after == "" && page == 0 {
				page++
				resp = map[string]interface{}{
					"data": map[string]interface{}{
						"user": map[string]interface{}{
							"starredRepositories": map[string]interface{}{
								"totalCount": 2,
								"nodes": []map[string]interface{}{
									{
										"name":           "repo-one",
										"nameWithOwner":  "user/repo-one",
										"description":    "First repo",
										"url":            "https://github.com/user/repo-one",
										"stargazerCount": 100,
										"forkCount":      10,
										"isPrivate":      false,
										"languages": map[string]interface{}{
											"edges": []map[string]interface{}{
												{"node": map[string]interface{}{"name": "Go"}},
											},
										},
										"repositoryTopics": map[string]interface{}{
											"nodes": []map[string]interface{}{
												{"topic": map[string]interface{}{"name": "tools", "stargazerCount": 600}},
												{"topic": map[string]interface{}{"name": "low-stars", "stargazerCount": 10}},
											},
										},
									},
								},
								"pageInfo": map[string]interface{}{
									"endCursor":   "cursor-page-2",
									"hasNextPage": true,
								},
							},
						},
					},
				}
			} else {
				resp = map[string]interface{}{
					"data": map[string]interface{}{
						"user": map[string]interface{}{
							"starredRepositories": map[string]interface{}{
								"totalCount": 2,
								"nodes": []map[string]interface{}{
									{
										"name":           "repo-two",
										"nameWithOwner":  "user/repo-two",
										"description":    "Second repo",
										"url":            "https://github.com/user/repo-two",
										"stargazerCount": 50,
										"forkCount":      5,
										"isPrivate":      true,
										"languages": map[string]interface{}{
											"edges": []map[string]interface{}{
												{"node": map[string]interface{}{"name": "Python"}},
											},
										},
										"repositoryTopics": map[string]interface{}{
											"nodes": []map[string]interface{}{},
										},
									},
								},
								"pageInfo": map[string]interface{}{
									"endCursor":   "",
									"hasNextPage": false,
								},
							},
						},
					},
				}
			}

			bodyBytes, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}

	repos, err := client.FetchStarredRepos(context.Background(), "user", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}

	if repos[0].NameWithOwner != "user/repo-one" || repos[0].Language != "Go" {
		t.Errorf("repo 1 mismatch: %+v", repos[0])
	}
	if len(repos[0].Topics) != 1 || repos[0].Topics[0] != "tools" {
		t.Errorf("topics threshold filter mismatch: %+v", repos[0].Topics)
	}

	if repos[1].NameWithOwner != "user/repo-two" || !repos[1].IsPrivate {
		t.Errorf("repo 2 mismatch: %+v", repos[1])
	}
}
