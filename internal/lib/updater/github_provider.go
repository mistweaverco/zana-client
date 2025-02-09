package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
)

type GitHubProvider struct{}

// getPackageId returns the package ID from a given source
func (g *GitHubProvider) getPackageId(sourceId string) string {
	sourceId = strings.TrimPrefix(sourceId, "pkg:")
	parts := strings.Split(sourceId, "/")
	return strings.Join(parts[1:], "/")
}

// getWebUrl returns the GitHub web URL for a given source
func (g *GitHubProvider) getWebUrl(sourceId string) string {
	return "https://github.com/" + g.getPackageId(sourceId)
}

// getApiUrl returns the GitHub API URL for a given source
func (g *GitHubProvider) getApiUrl(sourceId string) string {
	return "https://api.github.com/repos/" + g.getPackageId(sourceId)
}

// Update updates a package via the GitHub provider
func (g *GitHubProvider) Update(sourceId string) {
	log.Info("Updating via GitHub provider", "source", sourceId)
}

func stripVersionPrefix(version string) string {
	return strings.TrimPrefix(version, "v")
}

// GetLatestReleaseVersionNumber returns the latest release version number for a GitHub repository
func (g *GitHubProvider) GetLatestReleaseVersionNumber(sourceId string) string {
	url := g.getApiUrl(sourceId) + "/releases/latest"

	// Create an HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// Check if the repository exists or if there was an error
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("GitHub API returned status code %d\n", resp.StatusCode)
		return ""
	}

	// Read and parse the JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return ""
	}

	// Structure to hold the JSON response
	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return ""
	}

	return stripVersionPrefix(release.TagName)
}
