// colltest is a CLI smoke-test harness: it runs the same collect + analyze
// pipeline as the GUI without WebView2 or UAC, printing the report as JSON.
// Usage: colltest [-duration 15] [-interval 1]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"diagnostic-studio/internal/analyzer"
	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
)

func main() {
	duration := flag.Int("duration", 15, "sampling duration in seconds")
	interval := flag.Int("interval", 1, "sampling interval in seconds")
	flag.Parse()

	progress := func(done, total int) {
		fmt.Fprintf(os.Stderr, "sample %d/%d\n", done, total)
	}

	report := collector.CollectFor(context.Background(), model.ScanQuick, *duration, *interval, progress)
	analyzer.Analyze(&report)

	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
