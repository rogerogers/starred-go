package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	githubGraphQLEndpoint = "https://api.github.com/graphql"
	githubRESTEndpoint    = "https://api.github.com"
	userAgent             = "starred-go/1.0"
)

const gqlStarredQuery = `
query ($username: String!, $after: String) {
  user(login: $username) {
    starredRepositories(first: 100, after: $after, orderBy: {direction: DESC, field: STARRED_AT}) {
      totalCount
      nodes {
        name
        nameWithOwner
        description
        url
        stargazerCount
        forkCount
        isPrivate
        pushedAt
        updatedAt
        languages(first: 1, orderBy: {field: SIZE, direction: DESC}) {
          edges {
            node {
              id
              name
            }
          }
        }
        repositoryTopics(first: 100) {
          nodes {
            topic {
              name
              stargazerCount
            }
          }
        }
      }
      pageInfo {
        endCursor
        hasNextPage
      }
    }
  }
}
`

type gqlPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type gqlResponse struct {
	Data struct {
		User *struct {
			StarredRepositories struct {
				TotalCount int `json:"totalCount"`
				Nodes      []struct {
					Name           string `json:"name"`
					NameWithOwner  string `json:"nameWithOwner"`
					Description    string `json:"description"`
					URL            string `json:"url"`
					StargazerCount int    `json:"stargazerCount"`
					ForkCount      int    `json:"forkCount"`
					IsPrivate      bool   `json:"isPrivate"`
					Languages      struct {
						Edges []struct {
							Node struct {
								ID   string `json:"id"`
								Name string `json:"name"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"languages"`
					RepositoryTopics struct {
						Nodes []struct {
							Topic struct {
								Name           string `json:"name"`
								StargazerCount int    `json:"stargazerCount"`
							} `json:"topic"`
						} `json:"nodes"`
					} `json:"repositoryTopics"`
				} `json:"nodes"`
				PageInfo struct {
					EndCursor   string `json:"endCursor"`
					HasNextPage bool   `json:"hasNextPage"`
				} `json:"pageInfo"`
			} `json:"starredRepositories"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// GitHubClient handles interaction with GitHub GraphQL and REST APIs.
type GitHubClient struct {
	token      string
	httpClient *http.Client
}

// NewGitHubClient creates a GitHub client with authorization token.
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token: token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FetchStarredRepos fetches all starred repositories of a user using GitHub GraphQL API.
func (c *GitHubClient) FetchStarredRepos(ctx context.Context, username string, topicLimit int) ([]Repository, error) {
	var allRepos []Repository
	var cursor *string

	for {
		variables := map[string]interface{}{
			"username": username,
		}
		if cursor != nil && *cursor != "" {
			variables["after"] = *cursor
		}

		payloadBytes, err := json.Marshal(gqlPayload{
			Query:     gqlStarredQuery,
			Variables: variables,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubGraphQLEndpoint, bytes.NewReader(payloadBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GraphQL request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read GraphQL response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub GraphQL API error (status %d): %s", resp.StatusCode, string(body))
		}

		var gqlResp gqlResponse
		if err := json.Unmarshal(body, &gqlResp); err != nil {
			return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
		}

		if len(gqlResp.Errors) > 0 {
			var errMsg []string
			for _, e := range gqlResp.Errors {
				errMsg = append(errMsg, e.Message)
			}
			return nil, fmt.Errorf("GitHub GraphQL errors: %s", strings.Join(errMsg, "; "))
		}

		if gqlResp.Data.User == nil {
			return nil, fmt.Errorf("user '%s' not found on GitHub", username)
		}

		starred := gqlResp.Data.User.StarredRepositories
		for _, node := range starred.Nodes {
			var lang string
			if len(node.Languages.Edges) > 0 {
				lang = node.Languages.Edges[0].Node.Name
			}

			var topics []string
			for _, tNode := range node.RepositoryTopics.Nodes {
				if tNode.Topic.StargazerCount > topicLimit {
					topics = append(topics, tNode.Topic.Name)
				}
			}

			allRepos = append(allRepos, Repository{
				NameWithOwner:  node.NameWithOwner,
				Description:    node.Description,
				Language:       lang,
				URL:            node.URL,
				StargazerCount: node.StargazerCount,
				IsPrivate:      node.IsPrivate,
				Topics:         topics,
			})
		}

		if !starred.PageInfo.HasNextPage || starred.PageInfo.EndCursor == "" {
			break
		}
		c := starred.PageInfo.EndCursor
		cursor = &c
	}

	return allRepos, nil
}

// SyncToRepository commits markdown file to a GitHub repository, creating the repository if needed.
func (c *GitHubClient) SyncToRepository(ctx context.Context, username, repoName, filename, commitMsg, content string) (string, error) {
	// 1. Ensure repository exists
	repoURL, err := c.ensureRepository(ctx, username, repoName)
	if err != nil {
		return "", err
	}

	// 2. Read existing file content (if any)
	fileURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s", githubRESTEndpoint, username, repoName, filename)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query file '%s': %w", filename, err)
	}
	defer resp.Body.Close()

	var sha string
	var existingContent string

	if resp.StatusCode == http.StatusOK {
		var fileInfo struct {
			SHA      string `json:"sha"`
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err == nil {
			sha = fileInfo.SHA
			if fileInfo.Encoding == "base64" {
				cleanB64 := strings.ReplaceAll(fileInfo.Content, "\n", "")
				if decoded, err := base64.StdEncoding.DecodeString(cleanB64); err == nil {
					existingContent = string(decoded)
				}
			}
		}
	}

	// If content is identical, no commit needed
	if sha != "" && existingContent == content {
		return repoURL, nil
	}

	// 3. Create or update file
	putBody := map[string]interface{}{
		"message": commitMsg,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
	}
	if sha != "" {
		putBody["sha"] = sha
	}

	bodyBytes, err := json.Marshal(putBody)
	if err != nil {
		return "", err
	}

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, fileURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	putReq.Header.Set("Authorization", "Bearer "+c.token)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("User-Agent", userAgent)

	putResp, err := c.httpClient.Do(putReq)
	if err != nil {
		return "", fmt.Errorf("failed to commit file: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK && putResp.StatusCode != http.StatusCreated {
		errBytes, _ := io.ReadAll(putResp.Body)
		return "", fmt.Errorf("commit failed (status %d): %s", putResp.StatusCode, string(errBytes))
	}

	return repoURL, nil
}

func (c *GitHubClient) ensureRepository(ctx context.Context, username, repoName string) (string, error) {
	getURL := fmt.Sprintf("%s/repos/%s/%s", githubRESTEndpoint, username, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to check repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var repoInfo struct {
			HTMLURL string `json:"html_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err == nil && repoInfo.HTMLURL != "" {
			return repoInfo.HTMLURL, nil
		}
		return fmt.Sprintf("https://github.com/%s/%s", username, repoName), nil
	}

	if resp.StatusCode == http.StatusNotFound {
		// Create repo
		createURL := fmt.Sprintf("%s/user/repos", githubRESTEndpoint)
		createPayload := map[string]interface{}{
			"name":        repoName,
			"description": "A curated list of my GitHub stars!",
			"auto_init":   false,
		}
		pBytes, _ := json.Marshal(createPayload)
		createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(pBytes))
		if err != nil {
			return "", err
		}
		createReq.Header.Set("Authorization", "Bearer "+c.token)
		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("User-Agent", userAgent)

		createResp, err := c.httpClient.Do(createReq)
		if err != nil {
			return "", fmt.Errorf("failed to create repository: %w", err)
		}
		defer createResp.Body.Close()

		if createResp.StatusCode != http.StatusCreated && createResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(createResp.Body)
			return "", fmt.Errorf("failed to create repo %s (status %d): %s", repoName, createResp.StatusCode, string(body))
		}

		var createdInfo struct {
			HTMLURL string `json:"html_url"`
		}
		if err := json.NewDecoder(createResp.Body).Decode(&createdInfo); err == nil && createdInfo.HTMLURL != "" {
			return createdInfo.HTMLURL, nil
		}
		return fmt.Sprintf("https://github.com/%s/%s", username, repoName), nil
	}

	body, _ := io.ReadAll(resp.Body)
	return "", fmt.Errorf("repository check returned unexpected status %d: %s", resp.StatusCode, string(body))
}

// OpenInBrowser attempts to open the given URL in the default web browser.
func OpenInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return errors.New("unsupported platform")
	}
	return cmd.Start()
}
