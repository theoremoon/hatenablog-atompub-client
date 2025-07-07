package article

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)

func ParseFile(filePath string) (*Article, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return ParseContent(string(content), filePath)
}

func ParseContent(content, filePath string) (*Article, error) {
	matches := frontmatterRegex.FindStringSubmatch(strings.TrimSpace(content))
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid frontmatter format in %s", filePath)
	}

	frontmatter := matches[1]
	body := strings.TrimSpace(matches[2])

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

	contentStr := string(content)
	matches := frontmatterRegex.FindStringSubmatch(strings.TrimSpace(contentStr))
	if len(matches) != 3 {
		return fmt.Errorf("invalid frontmatter format in %s", article.FilePath)
	}

	frontmatter := matches[1]
	body := matches[2]

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