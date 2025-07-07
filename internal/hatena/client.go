package hatena

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/theoremoon/hatenablog-atompub-client/internal/article"
	"github.com/theoremoon/hatenablog-atompub-client/internal/config"
)

type Client struct {
	config     *config.Config
	httpClient *http.Client
}

type AtomEntry struct {
	XMLName     xml.Name `xml:"entry"`
	Xmlns       string   `xml:"xmlns,attr"`
	XmlnsApp    string   `xml:"xmlns:app,attr"`
	ID          string   `xml:"id,omitempty"`
	Title       string   `xml:"title"`
	Content     Content  `xml:"content"`
	Updated     string   `xml:"updated,omitempty"`
	Published   string   `xml:"published,omitempty"`
	Link        []Link   `xml:"link,omitempty"`
	Control     *Control `xml:"app:control,omitempty"`
	CustomURL   string   `xml:"hatenablog:custom-url,omitempty"`
	XmlnsHatena string   `xml:"xmlns:hatenablog,attr,omitempty"`
}

type Content struct {
	Type string `xml:"type,attr"`
	Text string `xml:",chardata"`
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Control struct {
	Draft string `xml:"app:draft"`
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Entry   []AtomEntry `xml:"entry"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}


func (c *Client) getCollectionURL() string {
	return fmt.Sprintf("https://blog.hatena.ne.jp/%s/%s/atom/entry", c.config.HatenaID, c.config.BlogID)
}

func (c *Client) getMemberURL(entryID string) string {
	return fmt.Sprintf("https://blog.hatena.ne.jp/%s/%s/atom/entry/%s", c.config.HatenaID, c.config.BlogID, entryID)
}

func (c *Client) createRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", BasicAuth(c.config.HatenaID, c.config.APIKey))
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("User-Agent", "hatenablog-atompub-client/1.0")

	return req, nil
}

func (c *Client) GetEntries() ([]*article.HatenaEntry, error) {
	req, err := c.createRequest("GET", c.getCollectionURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var feed AtomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	var entries []*article.HatenaEntry
	for _, atomEntry := range feed.Entry {
		entry := &article.HatenaEntry{
			ID:      atomEntry.ID,
			Title:   atomEntry.Title,
			Content: atomEntry.Content.Text,
			Updated: atomEntry.Updated,
			IsDraft: atomEntry.Control != nil && atomEntry.Control.Draft == "yes",
		}

		for _, link := range atomEntry.Link {
			if link.Rel == "alternate" {
				entry.URL = link.Href
			} else if link.Rel == "edit" {
				entry.EditURL = link.Href
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (c *Client) CreateEntry(art *article.Article) (*article.HatenaEntry, error) {
	entry := &AtomEntry{
		Xmlns:       "http://www.w3.org/2005/Atom",
		XmlnsApp:    "http://www.w3.org/2007/app",
		XmlnsHatena: "http://www.hatena.ne.jp/info/xmlns#hatenablog",
		Title:       art.Title,
		Content: Content{
			Text: art.Content,
		},
		CustomURL: art.Path,
	}

	xmlData, err := xml.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}


	req, err := c.createRequest("POST", c.getCollectionURL(), bytes.NewReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}


	var createdEntry AtomEntry
	if err := xml.Unmarshal(responseBody, &createdEntry); err != nil {
		return nil, fmt.Errorf("failed to decode response XML: %w", err)
	}

	result := &article.HatenaEntry{
		ID:      createdEntry.ID,
		Title:   createdEntry.Title,
		Content: createdEntry.Content.Text,
		Updated: createdEntry.Updated,
		IsDraft: createdEntry.Control != nil && createdEntry.Control.Draft == "yes",
	}

	for _, link := range createdEntry.Link {
		if link.Rel == "alternate" {
			result.URL = link.Href
		} else if link.Rel == "edit" {
			result.EditURL = link.Href
		}
	}

	return result, nil
}

func (c *Client) UpdateEntry(entryID string, art *article.Article) (*article.HatenaEntry, error) {
	entry := &AtomEntry{
		Xmlns:       "http://www.w3.org/2005/Atom",
		XmlnsApp:    "http://www.w3.org/2007/app",
		XmlnsHatena: "http://www.hatena.ne.jp/info/xmlns#hatenablog",
		Title:       art.Title,
		Content: Content{
			Text: art.Content,
		},
		CustomURL: art.Path,
	}

	xmlData, err := xml.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}


	req, err := c.createRequest("PUT", c.getMemberURL(entryID), bytes.NewReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}


	var updatedEntry AtomEntry
	if err := xml.Unmarshal(responseBody, &updatedEntry); err != nil {
		return nil, fmt.Errorf("failed to decode response XML: %w", err)
	}

	result := &article.HatenaEntry{
		ID:      updatedEntry.ID,
		Title:   updatedEntry.Title,
		Content: updatedEntry.Content.Text,
		Updated: updatedEntry.Updated,
		IsDraft: updatedEntry.Control != nil && updatedEntry.Control.Draft == "yes",
	}

	for _, link := range updatedEntry.Link {
		if link.Rel == "alternate" {
			result.URL = link.Href
		} else if link.Rel == "edit" {
			result.EditURL = link.Href
		}
	}

	return result, nil
}

func (c *Client) DeleteEntry(entryID string) error {
	req, err := c.createRequest("DELETE", c.getMemberURL(entryID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func ExtractEntryIDFromEditURL(editURL string) string {
	re := regexp.MustCompile(`/atom/entry/(.+)$`)
	matches := re.FindStringSubmatch(editURL)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func ExtractUUIDFromEntryID(entryID string) string {
	parts := strings.Split(entryID, "-")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}


