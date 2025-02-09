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
func (g *GitHubProvider) getPackageId(source string) string {
	source = strings.TrimPrefix(source, "pkg:")
	parts := strings.Split(source, "/")
	return parts[1]
}

// getWebUrl returns the GitHub web URL for a given source
func (g *GitHubProvider) getWebUrl(source string) string {
	return "https://github.com/" + g.getPackageId(source)
}

// getApiUrl returns the GitHub API URL for a given source
func (g *GitHubProvider) getApiUrl(source string) string {
	return "https://api.github.com/repos/" + g.getPackageId(source)
}

// Update updates a package via the GitHub provider
func (g *GitHubProvider) Update(source string) {
	log.Info("Updating via GitHub provider", "source", source)
}

// GetLatestReleaseVersionNumber returns the latest release version number for a GitHub repository
func (g *GitHubProvider) GetLatestReleaseVersionNumber(source string) string {
	url := g.getApiUrl(source) + "/releases/latest"

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

	return release.TagName
}
