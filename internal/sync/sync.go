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

	remoteUUIDMap := make(map[string]*article.HatenaEntry)
	for _, entry := range remoteEntries {
		uuid := hatena.ExtractUUIDFromEntryID(entry.ID)
		if uuid != "" {
			remoteUUIDMap[uuid] = entry
		}
	}

	localUUIDMap := make(map[string]*article.Article)
	var articlesWithoutUUID []*article.Article

	for _, art := range localArticles {
		if art.UUID != "" {
			localUUIDMap[art.UUID] = art
		} else {
			articlesWithoutUUID = append(articlesWithoutUUID, art)
		}
	}

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
			_, err := s.client.CreateEntry(localArticle)
			if err != nil {
				if isDailyLimitExceeded(err) {
					log.Printf("はてなブログの1日の投稿制限に達しました。処理を停止します。")
					return result, fmt.Errorf("daily posting limit exceeded: %w", err)
				}
				err = fmt.Errorf("failed to create article %s: %w", localArticle.Title, err)
				result.Errors = append(result.Errors, err)
				continue
			}
			log.Printf("+ %s", localArticle.FilePath)
			result.Created++
		}
	}

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

	return result, nil
}

func (s *Syncer) DryRunSyncArticles(localArticles []*article.Article) (*SyncResult, error) {
	result := &SyncResult{}
	var actions []DryRunAction

	remoteEntries, err := s.client.GetEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote entries: %w", err)
	}

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
