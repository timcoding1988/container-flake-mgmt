package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/container-flake-mgmt/internal/analyzer"
	"github.com/containers/container-flake-mgmt/internal/cirrus"
	"github.com/containers/container-flake-mgmt/internal/parser"
	"github.com/containers/container-flake-mgmt/internal/reporter"
)

func main() {
	var (
		repo     = flag.String("repo", "containers/podman", "Repository (owner/name)")
		branch   = flag.String("branch", "main", "Branch to analyze")
		days     = flag.Int("days", 30, "Number of days to analyze")
		output   = flag.String("output", "docs", "Output directory for reports")
		dataDir  = flag.String("data", "data/results", "Directory for JSON data")
		workers  = flag.Int("workers", 10, "Number of concurrent artifact fetchers")
		verbose  = flag.Bool("verbose", false, "Verbose output")
		dryRun   = flag.Bool("dry-run", false, "Don't write files")
	)
	flag.Parse()

	parts := strings.SplitN(*repo, "/", 2)
	if len(parts) != 2 {
		log.Fatalf("Invalid repo format: %s (expected owner/repo)", *repo)
	}
	owner, name := parts[0], parts[1]

	token := os.Getenv("CIRRUS_API_TOKEN")
	client := cirrus.NewClient("", token)
	fetcher := cirrus.NewFetcher(client, *workers)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	window := time.Duration(*days) * 24 * time.Hour

	log.Printf("Fetching builds for %s/%s (branch: %s, window: %d days)", owner, name, *branch, *days)

	// Fetch builds
	builds, err := client.FetchBuilds(ctx, owner, name, *branch, window)
	if err != nil {
		log.Fatalf("Failed to fetch builds: %v", err)
	}
	log.Printf("Found %d builds", len(builds))

	// Collect artifact requests
	var requests []cirrus.ArtifactRequest
	for _, build := range builds {
		tasks, err := client.FetchTasks(ctx, build.ID)
		if err != nil {
			log.Printf("Warning: failed to fetch tasks for build %s: %v", build.ID, err)
			continue
		}

		for _, task := range tasks {
			if !task.IsTestTask() || !task.IsFinished() {
				continue
			}

			requests = append(requests, cirrus.ArtifactRequest{
				TaskID:   task.ID,
				TaskName: task.Name,
				BuildID:  build.ID,
				Commit:   build.ChangeIDInRepo,
				Time:     build.Timestamp(),
			})
		}
	}

	log.Printf("Fetching %d artifacts concurrently with %d workers", len(requests), *workers)

	// Fetch all artifacts concurrently
	results := fetcher.FetchAll(ctx, requests)

	// Parse results
	var allResults []parser.TestResult
	successCount := 0
	for _, r := range results {
		if r.Error != nil {
			if *verbose {
				log.Printf("  No artifact for task %s: %v", r.Request.TaskName, r.Error)
			}
			continue
		}

		parsed, err := parser.ParseHTML(r.Data, "")
		if err != nil {
			log.Printf("  Warning: failed to parse HTML for task %s: %v", r.Request.TaskName, err)
			continue
		}

		// Add build/task context
		for i := range parsed {
			parsed[i].BuildID = r.Request.BuildID
			parsed[i].TaskID = r.Request.TaskID
			parsed[i].TaskName = r.Request.TaskName
			parsed[i].Platform = r.Request.TaskName // Use task name as platform
			parsed[i].CommitSHA = r.Request.Commit
			parsed[i].Timestamp = r.Request.Time
		}

		allResults = append(allResults, parsed...)
		successCount++

		if *verbose {
			log.Printf("  Task %s: %d test results", r.Request.TaskName, len(parsed))
		}
	}

	log.Printf("Processed %d tasks, collected %d test results", successCount, len(allResults))

	// Analyze
	report := analyzer.Analyze(allResults)
	report.Repository = *repo
	report.Branch = *branch
	report.WindowDays = *days
	report.TotalBuilds = len(builds)

	log.Printf("Analysis complete: %d unique tests, %d flaky", report.TotalTests, report.FlakyCount)

	if *dryRun {
		log.Println("Dry run - not writing files")
		printSummary(report)
		return
	}

	// Write JSON data
	jsonPath := filepath.Join(*dataDir, fmt.Sprintf("%s.json", strings.ReplaceAll(*repo, "/", "-")))
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}
	log.Printf("Wrote JSON report to %s", jsonPath)

	// Write HTML report
	htmlPath := filepath.Join(*output, "index.html")
	if err := reporter.WriteHTML(report, htmlPath); err != nil {
		log.Fatalf("Failed to write HTML: %v", err)
	}
	log.Printf("Wrote HTML report to %s", htmlPath)

	printSummary(report)
}

func printSummary(report *analyzer.Report) {
	fmt.Println()
	fmt.Println("=== Flakiness Report Summary ===")
	fmt.Printf("Repository: %s\n", report.Repository)
	fmt.Printf("Branch: %s\n", report.Branch)
	fmt.Printf("Window: %d days\n", report.WindowDays)
	fmt.Printf("Total builds analyzed: %d\n", report.TotalBuilds)
	fmt.Printf("Unique tests (per platform): %d\n", report.TotalTests)
	fmt.Printf("Flaky tests: %d\n", report.FlakyCount)
	fmt.Println()

	if report.FlakyCount > 0 {
		fmt.Println("Top Flaky Tests:")
		count := 0
		for _, t := range report.Tests {
			if t.IsFlaky() && count < 10 {
				fmt.Printf("  [%s] %s @ %s: %.1f%% (%d/%d failures)\n",
					t.Classification, t.Name, t.Platform, t.FlakinessPct, t.FailCount, t.TotalRuns)
				count++
			}
		}
	}
}
