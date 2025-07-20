package sync

import (
	"fmt"
	"log"
	"strings"

	"github.com/theoremoon/hatenablog-atompub-client/internal/article"
	"github.com/theoremoon/hatenablog-atompub-client/internal/hatena"
)

type Syncer struct {
	client       *hatena.Client
	deleteOrphan bool
}

type SyncResult struct {
	Created int
	Updated int
	Skipped int
	Deleted int
	Errors  []error
}

type DryRunAction struct {
	Type        string // "create", "update", "skip", "delete"
	Article     *article.Article
	RemoteEntry *article.HatenaEntry
	Reason      string
}

type DuplicateEntry struct {
	Title   string
	Entries []*article.HatenaEntry
}

func NewSyncer(client *hatena.Client) *Syncer {
	return &Syncer{client: client, deleteOrphan: false}
}

func NewSyncerWithDelete(client *hatena.Client, deleteOrphan bool) *Syncer {
	return &Syncer{client: client, deleteOrphan: deleteOrphan}
}

func (s *Syncer) SyncArticles(localArticles []*article.Article) (*SyncResult, error) {
	result := &SyncResult{}

	remoteEntries, err := s.client.GetEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote entries: %w", err)
	}

	// Check for duplicate entries first
	duplicates := s.FindDuplicateEntries(remoteEntries)
	s.ReportDuplicateEntries(duplicates)

	remoteUUIDMap := make(map[string]*article.HatenaEntry)
	for _, entry := range remoteEntries {
		uuid := hatena.ExtractUUIDFromEntryID(entry.ID)
		if uuid != "" {
			remoteUUIDMap[uuid] = entry
		}
	}

	localUUIDMap := make(map[string]*article.Article)
	for _, art := range localArticles {
		if art.UUID != "" {
			localUUIDMap[art.UUID] = art
		}
	}

	// Delete orphaned articles first
	if s.deleteOrphan {
		for uuid, remoteEntry := range remoteUUIDMap {
			if _, exists := localUUIDMap[uuid]; !exists {
				entryID := hatena.ExtractEntryIDFromEditURL(remoteEntry.EditURL)
				if entryID == "" {
					err := fmt.Errorf("failed to extract entry ID from edit URL: %s", remoteEntry.EditURL)
					result.Errors = append(result.Errors, err)
					continue
				}

				err := s.client.DeleteEntry(entryID)
				if err != nil {
					err = fmt.Errorf("failed to delete article %s: %w", remoteEntry.Title, err)
					result.Errors = append(result.Errors, err)
					continue
				}
				log.Printf("- %s", remoteEntry.URL)
				result.Deleted++
			}
		}
	}

	// Then create/update local articles
	for _, localArticle := range localArticles {
		if localArticle.UUID == "" {
			createdEntry, err := s.client.CreateEntry(localArticle)
			if err != nil {
				if isDailyLimitExceeded(err) {
					return result, fmt.Errorf("daily posting limit exceeded: %w", err)
				}
				err = fmt.Errorf("failed to create article %s: %w", localArticle.Title, err)
				result.Errors = append(result.Errors, err)
				continue
			}

			uuid := hatena.ExtractUUIDFromEntryID(createdEntry.ID)
			if uuid != "" {
				if err := article.UpdateArticleUUID(localArticle, uuid); err != nil {
					log.Printf("Warning: failed to update UUID in file %s: %v", localArticle.FilePath, err)
				}
			}

			log.Printf("+ %s", localArticle.FilePath)
			result.Created++
			continue
		}

		if remoteEntry, exists := remoteUUIDMap[localArticle.UUID]; exists {
			if s.needsUpdate(localArticle, remoteEntry) {
				entryID := hatena.ExtractEntryIDFromEditURL(remoteEntry.EditURL)
				if entryID == "" {
					err := fmt.Errorf("failed to extract entry ID from edit URL: %s", remoteEntry.EditURL)
					result.Errors = append(result.Errors, err)
					continue
				}

				_, err := s.client.UpdateEntry(entryID, localArticle)
				if err != nil {
					err = fmt.Errorf("failed to update article %s: %w", localArticle.Title, err)
					result.Errors = append(result.Errors, err)
					continue
				}
				log.Printf("~ %s", localArticle.FilePath)
				result.Updated++
			} else {
				log.Printf("= %s", localArticle.FilePath)
				result.Skipped++
			}
		} else {
			// UUID exists but not found in remote - should not happen in normal flow
			// This case should be handled by the UUID-less creation logic above
			log.Printf("Warning: Article %s has UUID but not found in remote", localArticle.FilePath)
			result.Skipped++
		}
	}

	return result, nil
}

func (s *Syncer) DryRunSyncArticles(localArticles []*article.Article) (*SyncResult, error) {
	result := &SyncResult{}
	var actions []DryRunAction

	remoteEntries, err := s.client.GetEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote entries: %w", err)
	}

	// Check for duplicate entries first
	duplicates := s.FindDuplicateEntries(remoteEntries)
	s.ReportDuplicateEntries(duplicates)

	remoteUUIDMap := make(map[string]*article.HatenaEntry)
	for _, entry := range remoteEntries {
		uuid := hatena.ExtractUUIDFromEntryID(entry.ID)
		if uuid != "" {
			remoteUUIDMap[uuid] = entry
		}
	}

	localUUIDMap := make(map[string]*article.Article)
	for _, art := range localArticles {
		if art.UUID != "" {
			localUUIDMap[art.UUID] = art
		}
	}

	// Check for orphaned articles first
	if s.deleteOrphan {
		for uuid, remoteEntry := range remoteUUIDMap {
			if _, exists := localUUIDMap[uuid]; !exists {
				actions = append(actions, DryRunAction{
					Type:        "delete",
					RemoteEntry: remoteEntry,
					Reason:      "Article no longer exists locally",
				})
				result.Deleted++
			}
		}
	}

	// Then check local articles for create/update
	for _, localArticle := range localArticles {
		if localArticle.UUID == "" {
			actions = append(actions, DryRunAction{
				Type:    "create",
				Article: localArticle,
				Reason:  "New article (no UUID assigned yet)",
			})
			result.Created++
			continue
		}
		if remoteEntry, exists := remoteUUIDMap[localArticle.UUID]; exists {
			if s.needsUpdate(localArticle, remoteEntry) {
				var changes []string
				if localArticle.Title != remoteEntry.Title {
					changes = append(changes, fmt.Sprintf("title: '%s' → '%s'", remoteEntry.Title, localArticle.Title))
				}
				if localArticle.Content != remoteEntry.Content {
					changes = append(changes, "content: modified")
				}

				actions = append(actions, DryRunAction{
					Type:        "update",
					Article:     localArticle,
					RemoteEntry: remoteEntry,
					Reason:      fmt.Sprintf("Changes: %v", changes),
				})
				result.Updated++
			} else {
				actions = append(actions, DryRunAction{
					Type:        "skip",
					Article:     localArticle,
					RemoteEntry: remoteEntry,
					Reason:      "No changes detected",
				})
				result.Skipped++
			}
		} else {
			actions = append(actions, DryRunAction{
				Type:    "create",
				Article: localArticle,
				Reason:  "New article (UUID not found in remote)",
			})
			result.Created++
		}
	}

	s.printDryRunReport(actions)
	return result, nil
}

func (s *Syncer) printDryRunReport(actions []DryRunAction) {
	for _, action := range actions {
		switch action.Type {
		case "create":
			fmt.Printf("+ %s\n", action.Article.FilePath)
		case "update":
			fmt.Printf("~ %s\n", action.Article.FilePath)
		case "skip":
			fmt.Printf("= %s\n", action.Article.FilePath)
		case "delete":
			if action.RemoteEntry.URL != "" {
				fmt.Printf("- %s\n", action.RemoteEntry.URL)
			}
		}
	}
}

func (s *Syncer) needsUpdate(local *article.Article, remote *article.HatenaEntry) bool {
	if local.Title != remote.Title {
		return true
	}

	if local.Content != remote.Content {
		return true
	}

	return false
}

func isDailyLimitExceeded(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "Entry limit was exceeded")
}

func (s *Syncer) FindDuplicateEntries(remoteEntries []*article.HatenaEntry) []DuplicateEntry {
	titleMap := make(map[string][]*article.HatenaEntry)
	
	// Group entries by title
	for _, entry := range remoteEntries {
		titleMap[entry.Title] = append(titleMap[entry.Title], entry)
	}
	
	// Find duplicates
	var duplicates []DuplicateEntry
	for title, entries := range titleMap {
		if len(entries) > 1 {
			duplicates = append(duplicates, DuplicateEntry{
				Title:   title,
				Entries: entries,
			})
		}
	}
	
	return duplicates
}

func (s *Syncer) ReportDuplicateEntries(duplicates []DuplicateEntry) {
	if len(duplicates) == 0 {
		fmt.Println("No duplicate entries found.")
		return
	}
	
	fmt.Printf("\n=== DUPLICATE ENTRIES DETECTED ===\n")
	fmt.Printf("Found %d titles with multiple entries:\n\n", len(duplicates))
	
	for i, dup := range duplicates {
		fmt.Printf("%d. Title: \"%s\" (%d entries)\n", i+1, dup.Title, len(dup.Entries))
		for j, entry := range dup.Entries {
			uuid := hatena.ExtractUUIDFromEntryID(entry.ID)
			fmt.Printf("   Entry %d: UUID=%s, Updated=%s\n", j+1, uuid, entry.Updated)
			if entry.URL != "" {
				fmt.Printf("             URL=%s\n", entry.URL)
			}
		}
		fmt.Println()
	}
	
	fmt.Println("=== END DUPLICATE REPORT ===\n")
}
