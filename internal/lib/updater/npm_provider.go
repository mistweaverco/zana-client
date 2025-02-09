package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
)

type NPMProvider struct{}

// getPackageId returns the package ID from a given source
func (p *NPMProvider) getPackageId(sourceId string) string {
	sourceId = strings.TrimPrefix(sourceId, "pkg:")
	parts := strings.Split(sourceId, "/")
	return strings.Join(parts[1:], "/")
}

// getWebUrl returns the NPM web URL for a given source
func (p *NPMProvider) getWebUrl(sourceId string) string {
	return "https://github.com/" + p.getPackageId(sourceId)
}

// getApiUrl returns the NPM API URL for a given source
func (p *NPMProvider) getApiUrl(sourceId string) string {
	return "https://registry.npmjs.org/" + p.getPackageId(sourceId)
}

func (p *NPMProvider) stripVersionPrefix(version string) string {
	return strings.TrimPrefix(version, "v")
}

func (p *NPMProvider) Update(source string) {
	log.Info("Updating via NPM provider", "source", source)
}

// GetLatestReleaseVersionNumber returns the latest release version number for a NPM package
func (p *NPMProvider) GetLatestReleaseVersionNumber(sourceId string) string {
	url := p.getApiUrl(sourceId) + "/latest"

	// Create an HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// Check if the repository exists or if there was an error
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API returned status code %d\n", resp.StatusCode)
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
		Version string `json:"version"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return ""
	}

	return p.stripVersionPrefix(release.Version)
}
