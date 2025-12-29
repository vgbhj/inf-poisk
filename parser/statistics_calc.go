package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CorpusStats struct {
	Source             string
	RawDocCount        int
	RawTotalSize       int64
	RawAvgDocSize      int64
	ParsedDocCount     int
	ParsedTotalSize    int64
	ParsedAvgDocSize   int64
	ExtractedTextRatio float64
}

func CalculateCorpusStatistics(corpusDir string, source string) (*CorpusStats, error) {
	stats := &CorpusStats{
		Source: source,
	}

	rawDir := filepath.Join(corpusDir, source, "raw")
	parsedDir := filepath.Join(corpusDir, source, "parsed")

	rawEntries, err := os.ReadDir(rawDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw directory: %w", err)
	}

	for _, entry := range rawEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		rawPath := filepath.Join(rawDir, entry.Name())
		fileInfo, err := os.Stat(rawPath)
		if err != nil {
			continue
		}

		stats.RawDocCount++
		stats.RawTotalSize += fileInfo.Size()
	}

	if stats.RawDocCount > 0 {
		stats.RawAvgDocSize = stats.RawTotalSize / int64(stats.RawDocCount)
	}

	parsedEntries, err := os.ReadDir(parsedDir)
	if err == nil {
		for _, entry := range parsedEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
				continue
			}

			parsedPath := filepath.Join(parsedDir, entry.Name())
			fileInfo, err := os.Stat(parsedPath)
			if err != nil {
				continue
			}

			stats.ParsedDocCount++
			stats.ParsedTotalSize += fileInfo.Size()
		}
	}

	if stats.ParsedDocCount > 0 {
		stats.ParsedAvgDocSize = stats.ParsedTotalSize / int64(stats.ParsedDocCount)
	}

	if stats.RawTotalSize > 0 {
		stats.ExtractedTextRatio = float64(stats.ParsedTotalSize) / float64(stats.RawTotalSize) * 100
	}

	return stats, nil
}

func PrintCorpusStatistics(stats *CorpusStats) {
	fmt.Printf("\nCorpus Statistics: %s\n", strings.ToUpper(stats.Source))
	fmt.Printf("=====================================\n")
	fmt.Printf("Raw Documents:\n")
	fmt.Printf("  Count:           %d\n", stats.RawDocCount)
	fmt.Printf("  Total size:      %s\n", formatBytes(stats.RawTotalSize))
	fmt.Printf("  Average size:    %s\n", formatBytes(stats.RawAvgDocSize))
	fmt.Printf("\nParsed Documents:\n")
	fmt.Printf("  Count:           %d\n", stats.ParsedDocCount)
	fmt.Printf("  Total size:      %s\n", formatBytes(stats.ParsedTotalSize))
	fmt.Printf("  Average size:    %s\n", formatBytes(stats.ParsedAvgDocSize))
	fmt.Printf("\nExtraction Ratio: %.2f%%\n", stats.ExtractedTextRatio)
	fmt.Printf("=====================================\n\n")
}
