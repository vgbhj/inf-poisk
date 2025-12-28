package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"corpus_parser/parser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	var configPath string
	var outputDir string
	var limit int

	flag.StringVar(&configPath, "config", "config.yaml", "Path to YAML config file")
	flag.StringVar(&outputDir, "output", "data", "Output directory for text files")
	flag.IntVar(&limit, "limit", 0, "Limit number of documents (0 = all)")
	flag.Parse()

	cfg, err := parser.LoadYAMLConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	db, err := parser.NewDatabase(cfg.DB.URI, cfg.DB.Database, cfg.DB.Collection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	os.MkdirAll(outputDir, 0755)

	ctx := context.Background()
	collection := db.GetCollection()

	filter := bson.M{}
	opts := options.Find()
	if limit > 0 {
		limit64 := int64(limit)
		opts = opts.SetLimit(limit64)
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query database: %v\n", err)
		os.Exit(1)
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		var doc parser.Document
		if err := cursor.Decode(&doc); err != nil {
			fmt.Printf("Error decoding document: %v\n", err)
			continue
		}

		var text string
		if doc.Source == "hltv" {
			article, err := parser.ParseHLTVArticleFromHTML(doc.RawHTML, doc.URL)
			if err != nil {
				fmt.Printf("Error parsing HLTV article %s: %v\n", doc.URL, err)
				continue
			}
			text = article.Title + "\n\n" + article.Content
		} else if doc.Source == "cybersport" {
			article, err := parser.ParseCybersportArticleFromHTML(doc.RawHTML, doc.URL)
			if err != nil {
				fmt.Printf("Error parsing Cybersport article %s: %v\n", doc.URL, err)
				continue
			}
			text = article.Title + "\n\n" + article.Content
		} else {
			continue
		}

		if strings.TrimSpace(text) == "" {
			continue
		}

		safeURL := strings.ReplaceAll(doc.URL, "https://", "")
		safeURL = strings.ReplaceAll(safeURL, "http://", "")
		safeURL = strings.ReplaceAll(safeURL, "/", "_")
		safeURL = strings.ReplaceAll(safeURL, "?", "_")
		safeURL = strings.ReplaceAll(safeURL, "&", "_")
		if len(safeURL) > 200 {
			safeURL = safeURL[:200]
		}

		filename := fmt.Sprintf("%s_%s.txt", doc.Source, safeURL)
		filepath := filepath.Join(outputDir, filename)

		if err := os.WriteFile(filepath, []byte(text), 0644); err != nil {
			fmt.Printf("Error writing file %s: %v\n", filepath, err)
			continue
		}

		count++
		if count%100 == 0 {
			fmt.Printf("Exported %d documents...\n", count)
		}
	}

	fmt.Printf("Total exported: %d documents\n", count)
}

