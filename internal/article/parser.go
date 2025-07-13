package article

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseFile(filePath string) (*Article, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return ParseContent(string(content), filePath)
}

func ParseContent(content, filePath string) (*Article, error) {
	lines := strings.Split(content, "\n")
	
	// Check if first line is opening frontmatter delimiter
	if len(lines) == 0 || lines[0] != "---" {
		return nil, fmt.Errorf("invalid frontmatter format in %s: missing opening ---", filePath)
	}
	
	// Find the second occurrence of "---" at the beginning of a line
	var frontmatterLines []string
	var bodyStartIndex int
	found := false
	
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			bodyStartIndex = i + 1
			found = true
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}
	
	if !found {
		return nil, fmt.Errorf("invalid frontmatter format in %s: missing closing ---", filePath)
	}
	
	frontmatter := strings.Join(frontmatterLines, "\n")
	var body string
	if bodyStartIndex < len(lines) {
		body = strings.Join(lines[bodyStartIndex:], "\n")
	}
	body = strings.TrimSpace(body)

	var article Article
	if err := yaml.Unmarshal([]byte(frontmatter), &article); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter in %s: %w", filePath, err)
	}

	article.Content = body
	article.FilePath = filePath

	return &article, nil
}

func LoadArticlesFromDir(dir string) ([]*Article, error) {
	var articles []*Article

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(path), ".md") || strings.HasSuffix(strings.ToLower(path), ".markdown") {
			article, err := ParseFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse %s: %w", path, err)
			}
			articles = append(articles, article)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return articles, nil
}

func UpdateArticleUUID(article *Article, uuid string) error {
	if article.UUID != "" {
		return fmt.Errorf("article already has UUID: %s", article.UUID)
	}

	content, err := os.ReadFile(article.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", article.FilePath, err)
	}

	lines := strings.Split(string(content), "\n")
	
	// Check if first line is opening frontmatter delimiter
	if len(lines) == 0 || lines[0] != "---" {
		return fmt.Errorf("invalid frontmatter format in %s", article.FilePath)
	}
	
	// Find the second occurrence of "---" at the beginning of a line
	var frontmatterLines []string
	var bodyStartIndex int
	found := false
	
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			bodyStartIndex = i + 1
			found = true
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}
	
	if !found {
		return fmt.Errorf("invalid frontmatter format in %s", article.FilePath)
	}
	
	frontmatter := strings.Join(frontmatterLines, "\n")
	var body string
	if bodyStartIndex < len(lines) {
		body = strings.Join(lines[bodyStartIndex:], "\n")
	}

	var frontMatterMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &frontMatterMap); err != nil {
		return fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	frontMatterMap["uuid"] = uuid

	updatedFrontmatter, err := yaml.Marshal(frontMatterMap)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	newContent := fmt.Sprintf("---\n%s---\n%s", string(updatedFrontmatter), body)

	if err := os.WriteFile(article.FilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", article.FilePath, err)
	}

	article.UUID = uuid
	return nil
}