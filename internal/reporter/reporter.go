package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/skoredin/db-benchmark-suite/internal/benchmark"
)

type Reporter struct {
	format string
	w      io.Writer
}

func New(format string, w io.Writer) *Reporter {
	return &Reporter{format: format, w: w}
}

func (r *Reporter) printLine(a ...any) {
	_, _ = fmt.Fprintln(r.w, a...)
}

func (r *Reporter) PrintHeader() {
	r.printLine()
	r.printLine("  Database Benchmark Suite")
	r.printLine()
}

func (r *Reporter) PrintResults(results map[string]*benchmark.Results) {
	switch r.format {
	case "json":
		r.printJSON(results)
	case "markdown":
		r.printMarkdown(results)
	default:
		r.printTable(results)
	}
}

func (r *Reporter) newTable(title string) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(r.w)
	t.SetTitle(title)
	t.SetStyle(table.StyleRounded)

	t.Style().Title.Align = text.AlignCenter
	t.Style().Format.Header = text.FormatDefault

	return t
}

func (r *Reporter) printTable(results map[string]*benchmark.Results) {
	databases := sortedKeys(results)
	r.printInsertTable(databases, results)
	r.printQueryTables(databases, results)
	r.printStorageTable(databases, results)
}

func (r *Reporter) printInsertTable(databases []string, results map[string]*benchmark.Results) {
	t := r.newTable("INSERT BENCHMARK")
	t.AppendHeader(table.Row{"Database", "Events", "Duration", "Throughput", "Errors", "Workers", "Batch"})

	for _, db := range databases {
		result := results[db]
		if result.Error != nil {
			t.AppendRow(table.Row{db, "ERROR", result.Error, "", "", "", ""})
		} else if result.Insert != nil {
			t.AppendRow(table.Row{
				db,
				result.Insert.TotalEvents,
				result.Insert.Duration.Round(time.Millisecond),
				fmt.Sprintf("%.0f/sec", result.Insert.Throughput),
				result.Insert.ErrorCount,
				result.Insert.WorkerCount,
				result.Insert.BatchSize,
			})
		}
	}

	t.Render()
	r.printLine()
}

func (r *Reporter) printQueryTables(databases []string, results map[string]*benchmark.Results) {
	for _, queryName := range sortedQueryNames(results) {
		t := r.newTable(queryName + " QUERY")
		t.AppendHeader(table.Row{"Database", "Avg", "Min", "Max", "P50", "P95", "P99", "Errors"})

		for _, db := range databases {
			result := results[db]
			if result.Queries == nil {
				continue
			}

			if qr, exists := result.Queries[queryName]; exists {
				t.AppendRow(table.Row{
					db,
					qr.AvgDuration.Round(time.Millisecond),
					qr.MinDuration.Round(time.Millisecond),
					qr.MaxDuration.Round(time.Millisecond),
					qr.P50Duration.Round(time.Millisecond),
					qr.P95Duration.Round(time.Millisecond),
					qr.P99Duration.Round(time.Millisecond),
					qr.ErrorCount,
				})
			}
		}

		t.Render()
		r.printLine()
	}
}

func (r *Reporter) printStorageTable(databases []string, results map[string]*benchmark.Results) {
	t := r.newTable("STORAGE STATISTICS")
	t.AppendHeader(table.Row{"Database", "Total Size", "Index Size", "Compression", "Row Count"})

	for _, db := range databases {
		result := results[db]
		if result.Storage != nil {
			t.AppendRow(table.Row{
				db,
				formatBytes(result.Storage.TotalSize),
				formatBytes(result.Storage.IndexSize),
				fmt.Sprintf("%.1f%%", result.Storage.CompressionPct),
				result.Storage.RowCount,
			})
		}
	}

	t.Render()
	r.printLine()
}

func (r *Reporter) printJSON(results map[string]*benchmark.Results) {
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(results); err != nil {
		log.Println(err)
	}
}

func (r *Reporter) printMarkdown(results map[string]*benchmark.Results) {
	databases := sortedKeys(results)
	r.printMarkdownInsert(databases, results)
	r.printMarkdownQueries(databases, results)
	r.printMarkdownStorage(databases, results)
}

func (r *Reporter) printMarkdownInsert(databases []string, results map[string]*benchmark.Results) {
	t := r.newTable("")
	t.SetStyle(table.StyleDefault)

	t.Style().Options.SeparateColumns = true

	t.AppendHeader(table.Row{"Database", "Events", "Duration", "Throughput", "Errors"})

	for _, db := range databases {
		result := results[db]
		if result.Error != nil {
			t.AppendRow(table.Row{db, "ERROR", "-", "-", "-"})
		} else if result.Insert != nil {
			t.AppendRow(table.Row{
				db,
				result.Insert.TotalEvents,
				result.Insert.Duration.Round(time.Second),
				fmt.Sprintf("%.0f/sec", result.Insert.Throughput),
				result.Insert.ErrorCount,
			})
		}
	}

	r.printLine("## Insert Performance")
	t.RenderMarkdown()
	r.printLine()
}

func (r *Reporter) printMarkdownQueries(databases []string, results map[string]*benchmark.Results) {
	for _, queryName := range sortedQueryNames(results) {
		_, _ = fmt.Fprintf(r.w, "\n### %s Query\n\n", queryName)

		t := r.newTable("")
		t.AppendHeader(table.Row{"Database", "Avg", "Min", "Max", "P95", "P99"})

		for _, db := range databases {
			result := results[db]
			if result.Queries == nil {
				continue
			}

			if qr, exists := result.Queries[queryName]; exists {
				t.AppendRow(table.Row{
					db,
					qr.AvgDuration.Round(time.Millisecond),
					qr.MinDuration.Round(time.Millisecond),
					qr.MaxDuration.Round(time.Millisecond),
					qr.P95Duration.Round(time.Millisecond),
					qr.P99Duration.Round(time.Millisecond),
				})
			}
		}

		t.RenderMarkdown()
		r.printLine()
	}
}

func (r *Reporter) printMarkdownStorage(databases []string, results map[string]*benchmark.Results) {
	r.printLine("\n## Storage Statistics")

	t := r.newTable("")
	t.AppendHeader(table.Row{"Database", "Total Size", "Index Size", "Compression", "Rows"})

	for _, db := range databases {
		result := results[db]
		if result.Storage != nil {
			t.AppendRow(table.Row{
				db,
				formatBytes(result.Storage.TotalSize),
				formatBytes(result.Storage.IndexSize),
				fmt.Sprintf("%.1f%%", result.Storage.CompressionPct),
				result.Storage.RowCount,
			})
		}
	}

	t.RenderMarkdown()
	r.printLine()
}

func sortedKeys(results map[string]*benchmark.Results) []string {
	databases := make([]string, 0, len(results))

	for db := range results {
		databases = append(databases, db)
	}

	sort.Strings(databases)

	return databases
}

func sortedQueryNames(results map[string]*benchmark.Results) []string {
	queryNames := make(map[string]bool)

	for _, result := range results {
		if result.Queries != nil {
			for name := range result.Queries {
				queryNames[name] = true
			}
		}
	}

	sorted := make([]string, 0, len(queryNames))

	for name := range queryNames {
		sorted = append(sorted, name)
	}

	sort.Strings(sorted)

	return sorted
}

func formatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
