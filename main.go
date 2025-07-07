package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/theoremoon/hatenablog-atompub-client/internal/article"
	"github.com/theoremoon/hatenablog-atompub-client/internal/config"
	"github.com/theoremoon/hatenablog-atompub-client/internal/hatena"
	"github.com/theoremoon/hatenablog-atompub-client/internal/sync"
)

func main() {
	var articlesDir string
	var dryRun bool
	var deleteOrphan bool
	flag.StringVar(&articlesDir, "dir", ".", "Directory containing article files")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be done without making any changes")
	flag.BoolVar(&deleteOrphan, "delete-orphan", false, "Delete remote articles that no longer exist locally (DANGEROUS)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	articles, err := article.LoadArticlesFromDir(articlesDir)
	if err != nil {
		log.Fatalf("Failed to load articles: %v", err)
	}

	if len(articles) == 0 {
		fmt.Println("No articles found")
		return
	}

	if deleteOrphan && !dryRun {
		fmt.Print("WARNING: --delete-orphan is enabled. This will permanently delete remote articles that don't exist locally.\nAre you sure you want to continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Operation cancelled.")
			os.Exit(0)
		}
	}

	client := hatena.NewClient(cfg)
	syncer := sync.NewSyncerWithDelete(client, deleteOrphan)

	var result *sync.SyncResult

	if dryRun {
		result, err = syncer.DryRunSyncArticles(articles)
	} else {
		result, err = syncer.SyncArticles(articles)
	}
	if err != nil {
		log.Fatalf("Synchronization failed: %v", err)
	}

	fmt.Printf("Created: %d, Updated: %d, Skipped: %d, Deleted: %d, Errors: %d\n",
		result.Created, result.Updated, result.Skipped, result.Deleted, len(result.Errors))

	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}
}