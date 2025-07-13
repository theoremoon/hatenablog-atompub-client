# Hatena Blog AtomPub API Client - Development Memory

## Project Overview
ofj���nAtomPub API�(Wf����n������ա��Y�CLI���

## Technical Stack
- **Language**: Go
- **API**: Hatena Blog AtomPub API
- **Authentication**: Basic�< (HATENA_ID + API_KEY)
- **File Format**: Markdown + YAML frontmatter
- **Article Identification**: UUID-based matching

## Project Structure
```
cmd/main.go                 - CLI entry point
internal/
  config/config.go          - Environment variable handling
  article/
    model.go               - Article and HatenaEntry structs
    parser.go              - YAML frontmatter parsing
  hatena/
    client.go              - AtomPub API client
  sync/
    sync.go                - Synchronization logic
```

## Key Features Implemented
1. **Environment Configuration**: HATENA_ID, BLOG_ID, API_KEY
2. **YAML Frontmatter**: title, path, uuid fields
3. **UUID Management**: Auto-generation for new articles, written back to files
4. **Sync Operations**: Create, Update, Skip, Delete (with --delete-orphan)
5. **Dry Run Mode**: Preview changes without executing
6. **Diff-style Output**: +, ~, =, - symbols for action types
7. **Custom URL Paths**: Via hatenablog:custom-url element

## CLI Options
- `-dir`: Article directory (default: current directory)
- `-dry-run`: Preview mode
- `-delete-orphan`: Delete orphaned remote articles (DANGEROUS)

## Article File Format
```yaml
---
title: "Article Title"
path: "custom-path"
uuid: "auto-generated-uuid"
---

Article content here...
```

## API Integration Details
- **Base URL**: https://blog.hatena.ne.jp/{HATENA_ID}/{BLOG_ID}/atom/entry
- **Authentication**: Basic Auth with HATENA_ID:API_KEY
- **Content Type**: Default blog syntax (removed text/x-markdown specification)
- **Custom URLs**: Uses hatenablog:custom-url XML element
- **Namespaces**: 
  - xmlns="http://www.w3.org/2005/Atom"
  - xmlns:app="http://www.w3.org/2007/app" 
  - xmlns:hatenablog="http://www.hatena.ne.jp/info/xmlns#hatenablog"

## Sync Logic
1. Load local articles from directory
2. Fetch remote entries via AtomPub API
3. Match articles by UUID extracted from entry IDs
4. Determine actions: create/update/skip/delete
5. Execute operations (or show preview in dry-run)
6. Write back generated UUIDs to new article files

## Recent Changes
- **Content Type**: Removed `text/x-markdown` type specification to use blog's default syntax
- **XML Debug**: Removed all XML debugging functionality for cleaner codebase
- **Simplified**: Focus on core sync functionality only
- **Daily Limit Protection**: Added 400 error detection for Hatena Blog's daily posting limit with automatic sync termination

## Build & Usage
```bash
# Build
go build -o hatenablog-atompub-client ./cmd

# Set environment variables
export HATENA_ID="your-hatena-id"
export BLOG_ID="your-blog.hatenablog.com" 
export API_KEY="your-api-key"

# Sync articles
./hatenablog-atompub-client -dir /path/to/articles -dry-run
./hatenablog-atompub-client -dir /path/to/articles
```

## Development Status
 Complete and functional CLI tool
 All core requirements implemented
 Cleaned up debugging code
 Ready for production use

## Communication Guidelines
- 会話は日本語で行います