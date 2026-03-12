package parser

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Ginkgo phases to capture (not just [It])
var ginkgoPhases = []string{"It", "BeforeEach", "AfterEach", "BeforeSuite", "AfterSuite", "JustBeforeEach", "JustAfterEach"}

// ParseHTML parses logformatter HTML output and extracts test results
func ParseHTML(data []byte, framework string) ([]TestResult, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var results []TestResult

	switch framework {
	case "bats":
		results = parseBATS(doc)
	case "ginkgo":
		results = parseGinkgo(doc)
	default:
		// Auto-detect based on content
		results = parseBATS(doc)
		if len(results) == 0 {
			results = parseGinkgo(doc)
		}
	}

	return results, nil
}

// parseBATS extracts test results from BATS logformatter output
func parseBATS(doc *html.Node) []TestResult {
	var results []TestResult

	// Pattern: "ok N [NNN] test name" or "not ok N [NNN] test name"
	batsLineRe := regexp.MustCompile(`^(not )?ok \d+ \[?\d*\]?\s*(.+?)(\s*#\s*skip.*)?$`)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			class := getAttr(n, "class")
			var status Status

			switch {
			case strings.Contains(class, "bats-passed"):
				status = StatusPassed
			case strings.Contains(class, "bats-failed"):
				status = StatusFailed
			case strings.Contains(class, "bats-skipped"):
				status = StatusSkipped
			default:
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
				return
			}

			text := extractText(n)
			text = strings.TrimSpace(text)

			matches := batsLineRe.FindStringSubmatch(text)
			if matches != nil {
				testName := strings.TrimSpace(matches[2])
				if strings.Contains(class, "bats-skipped") {
					testName = strings.TrimSuffix(testName, " # skip")
					testName = strings.Split(testName, " # skip")[0]
				}

				results = append(results, TestResult{
					Name:      testName,
					Framework: "bats",
					Phase:     "It",
					Status:    status,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return results
}

// parseGinkgo extracts test results from Ginkgo logformatter output
// Captures [It], [BeforeEach], [AfterEach], etc.
func parseGinkgo(doc *html.Node) []TestResult {
	var results []TestResult
	var currentSuite string
	var currentStatus Status

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h2" {
			class := getAttr(n, "class")

			// Determine status from class
			switch {
			case strings.Contains(class, "log-passed"):
				currentStatus = StatusPassed
			case strings.Contains(class, "log-failed"):
				currentStatus = StatusFailed
			case strings.Contains(class, "log-skipped"):
				currentStatus = StatusSkipped
			default:
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
				return
			}

			text := strings.TrimSpace(extractText(n))

			// Check for any Ginkgo phase block
			var phase string
			var testName string

			for _, p := range ginkgoPhases {
				prefix := "[" + p + "] "
				if strings.HasPrefix(text, prefix) {
					phase = p
					testName = strings.TrimPrefix(text, prefix)
					break
				}
			}

			if phase != "" {
				// Remove any trailing status markers
				testName = strings.TrimSuffix(testName, " [SKIPPED]")
				testName = strings.TrimSuffix(testName, " [FAILED]")

				results = append(results, TestResult{
					Name:      testName,
					Suite:     currentSuite,
					Framework: "ginkgo",
					Phase:     phase,
					Status:    currentStatus,
				})
			} else {
				// This is a suite/describe name
				currentSuite = text
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return results
}

// getAttr returns the value of an HTML attribute
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// extractText extracts all text content from an HTML node
func extractText(n *html.Node) string {
	var buf bytes.Buffer
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return buf.String()
}
