package reporter

import (
	"fmt"
	"html/template"

	"github.com/containers/container-flake-mgmt/internal/analyzer"
)

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"classColor": classColor,
	"formatPct":  formatPct,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Container Flakiness Report - {{ .Repository }}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
            background-color: #0d1117;
            color: #c9d1d9;
            line-height: 1.5;
            padding: 20px;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
        }

        header {
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px solid #30363d;
        }

        h1 {
            font-size: 32px;
            font-weight: 600;
            margin-bottom: 10px;
            color: #f0f6fc;
        }

        .meta {
            color: #8b949e;
            font-size: 14px;
        }

        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .card {
            background-color: #161b22;
            border: 1px solid #30363d;
            border-radius: 6px;
            padding: 20px;
        }

        .card-title {
            font-size: 14px;
            color: #8b949e;
            margin-bottom: 8px;
        }

        .card-value {
            font-size: 32px;
            font-weight: 600;
            color: #f0f6fc;
        }

        .filters {
            margin-bottom: 20px;
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }

        .filter-btn {
            padding: 8px 16px;
            background-color: #21262d;
            border: 1px solid #30363d;
            border-radius: 6px;
            color: #c9d1d9;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.2s;
        }

        .filter-btn:hover {
            background-color: #30363d;
        }

        .filter-btn.active {
            background-color: #1f6feb;
            border-color: #1f6feb;
            color: #ffffff;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            background-color: #161b22;
            border: 1px solid #30363d;
            border-radius: 6px;
            overflow: hidden;
        }

        thead {
            background-color: #21262d;
        }

        th {
            padding: 12px 16px;
            text-align: left;
            font-weight: 600;
            font-size: 12px;
            text-transform: uppercase;
            color: #8b949e;
            border-bottom: 1px solid #30363d;
        }

        td {
            padding: 12px 16px;
            border-bottom: 1px solid #30363d;
            font-size: 14px;
        }

        tbody tr {
            transition: background-color 0.2s;
        }

        tbody tr:hover {
            background-color: #21262d;
        }

        tbody tr:last-child td {
            border-bottom: none;
        }

        .badge {
            display: inline-block;
            padding: 4px 10px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 500;
            text-transform: uppercase;
        }

        .badge-critical {
            background-color: rgba(248, 81, 73, 0.15);
            color: #f85149;
        }

        .badge-high {
            background-color: rgba(242, 130, 53, 0.15);
            color: #f28235;
        }

        .badge-moderate {
            background-color: rgba(187, 128, 9, 0.15);
            color: #bb8009;
        }

        .badge-low {
            background-color: rgba(56, 139, 253, 0.15);
            color: #388bfd;
        }

        .test-name {
            font-weight: 500;
            color: #58a6ff;
        }

        .platform {
            font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
            font-size: 12px;
            color: #8b949e;
        }

        .stat {
            font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
            color: #c9d1d9;
        }

        .no-results {
            text-align: center;
            padding: 40px;
            color: #8b949e;
            font-size: 14px;
        }

        @media (max-width: 768px) {
            body {
                padding: 10px;
            }

            h1 {
                font-size: 24px;
            }

            .summary {
                grid-template-columns: 1fr;
            }

            table {
                font-size: 12px;
            }

            th, td {
                padding: 8px;
            }

            .card-value {
                font-size: 24px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Container Flakiness Report</h1>
            <div class="meta">
                <strong>Repository:</strong> {{ .Repository }} |
                <strong>Branch:</strong> {{ .Branch }} |
                <strong>Window:</strong> {{ .WindowDays }} days |
                <strong>Generated:</strong> {{ .GeneratedAt.Format "2006-01-02 15:04:05 MST" }}
            </div>
        </header>

        <div class="summary">
            <div class="card">
                <div class="card-title">Total Builds</div>
                <div class="card-value">{{ .TotalBuilds }}</div>
            </div>
            <div class="card">
                <div class="card-title">Total Tests</div>
                <div class="card-value">{{ .TotalTests }}</div>
            </div>
            <div class="card">
                <div class="card-title">Flaky Tests</div>
                <div class="card-value">{{ .FlakyCount }}</div>
            </div>
        </div>

        <div class="filters">
            <button class="filter-btn active" data-filter="all">All</button>
            <button class="filter-btn" data-filter="broken">Broken</button>
            <button class="filter-btn" data-filter="high">High</button>
            <button class="filter-btn" data-filter="medium">Medium</button>
            <button class="filter-btn" data-filter="low">Low</button>
        </div>

        <table>
            <thead>
                <tr>
                    <th>Test Name</th>
                    <th>Platform</th>
                    <th>Classification</th>
                    <th>Flakiness %</th>
                    <th>Runs</th>
                    <th>Pass/Fail</th>
                    <th>Days Since Fail</th>
                </tr>
            </thead>
            <tbody>
                {{ range .Tests }}
                <tr data-classification="{{ .Classification }}">
                    <td class="test-name">{{ .Name }}</td>
                    <td class="platform">{{ .Platform }}</td>
                    <td><span class="badge badge-{{ classColor .Classification }}">{{ .Classification }}</span></td>
                    <td class="stat">{{ formatPct .FlakinessPct }}</td>
                    <td class="stat">{{ .TotalRuns }}</td>
                    <td class="stat">{{ .PassCount }}/{{ .FailCount }}</td>
                    <td class="stat">{{ .DaysSinceFail }}</td>
                </tr>
                {{ end }}
            </tbody>
        </table>

        {{ if eq (len .Tests) 0 }}
        <div class="no-results">
            No flaky tests found in the analysis period.
        </div>
        {{ end }}
    </div>

    <script>
        // Filter functionality
        const filterBtns = document.querySelectorAll('.filter-btn');
        const rows = document.querySelectorAll('tbody tr');

        filterBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                // Update active state
                filterBtns.forEach(b => b.classList.remove('active'));
                btn.classList.add('active');

                const filter = btn.dataset.filter;

                // Filter rows
                rows.forEach(row => {
                    if (filter === 'all') {
                        row.style.display = '';
                    } else {
                        const classification = row.dataset.classification.toLowerCase();
                        row.style.display = classification === filter ? '' : 'none';
                    }
                });
            });
        });
    </script>
</body>
</html>`))

// classColor returns the CSS class color for a classification level
func classColor(classification analyzer.Classification) string {
	switch classification {
	case analyzer.ClassificationBroken:
		return "critical"
	case analyzer.ClassificationHigh:
		return "high"
	case analyzer.ClassificationMedium:
		return "moderate"
	case analyzer.ClassificationLow:
		return "low"
	case analyzer.ClassificationStable:
		return "low"
	default:
		return "low"
	}
}

// formatPct formats a percentage with one decimal place
func formatPct(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}
