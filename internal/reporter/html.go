package reporter

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/containers/container-flake-mgmt/internal/analyzer"
)

// GenerateHTML generates an HTML report from the analysis
func GenerateHTML(report *analyzer.Report) (string, error) {
	var buf bytes.Buffer
	if err := htmlTemplate.Execute(&buf, report); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteHTML generates and writes the HTML report to a file
func WriteHTML(report *analyzer.Report, outputPath string) error {
	html, err := GenerateHTML(report)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outputPath, []byte(html), 0644)
}
