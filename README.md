# はてなブログ AtomPub API クライアント

ローカルのマークダウンファイルをはてなブログに同期するCLIツールです。

## 機能

- YAML frontmatterを含むマークダウンファイルの読み込み
- UUID基準での記事の同一性判定
- 新規記事の投稿と既存記事の更新
- Basic認証を使用したはてなブログAtomPub APIとの通信

## 必要な環境変数

以下の環境変数を設定してください：

- `HATENA_ID`: はてなID
- `BLOG_ID`: ブログID（例：example.hatenablog.com）
- `API_KEY`: APIキー

## 記事ファイル形式

記事ファイルは以下の形式で作成してください：

### 新規記事（UUIDなし）
```markdown
---
title: "記事のタイトル"
path: "custom-article-path"
---

ここに記事の本文をマークダウンで書きます。

## セクション見出し

記事の内容...
```

### 既存記事（UUID付き）
```markdown
---
title: "記事のタイトル"
path: "custom-article-path"
uuid: "550e8400-e29b-41d4-a716-446655440000"
---

記事の内容...
```

**重要な仕様**:
- **UUID**: 手動設定不要。新規記事同期時に自動生成・書き戻し
- **Markdown記法**: 記事内容はMarkdown記法で記述
- **自動変換**: はてなブログ側でHTML変換されます

## 使用方法

1. プロジェクトをビルド：
   ```bash
   go build -o hatenablog-atompub-client ./cmd
   ```

2. 環境変数を設定：
   ```bash
   export HATENA_ID="your-hatena-id"              # はてなID
   export BLOG_ID="your-blog.hatenablog.com"      # ブログドメイン
   export API_KEY="your-api-key"                  # AtomPub APIキー
   ```
   
   **APIキーの取得方法**:
   1. はてなブログの設定ページにアクセス
   2. 「詳細設定」→「AtomPub」
   3. APIキーを確認・生成

3. 記事を同期：
   ```bash
   # Dry run（変更内容の確認のみ）
   ./hatenablog-atompub-client -dir /path/to/articles -dry-run
   
   # 実際に同期を実行
   ./hatenablog-atompub-client -dir /path/to/articles
   
   # 削除も含めた完全同期（DANGEROUS）
   ./hatenablog-atompub-client -dir /path/to/articles -delete-orphan -dry-run  # 最初は必ずdry-runで確認
   ./hatenablog-atompub-client -dir /path/to/articles -delete-orphan
   ```

## オプション

- `-dir`: 記事ファイルが格納されているディレクトリ（デフォルト：カレントディレクトリ）
- `-dry-run`: 実際の変更を行わず、何が実行されるかのみを表示
- `-delete-orphan`: ローカルに存在しないリモート記事を削除（⚠️ **危険**）

## 同期動作

- **UUIDなしの記事**: 新規記事として作成し、生成されたUUIDをファイルに書き戻し
- **UUIDが一致する記事が既に存在する場合**: タイトルまたは本文に変更があれば更新
- **UUIDが一致する記事が存在しない場合**: 新規作成
- **変更がない場合**: スキップ
- **`-delete-orphan` 使用時**: ローカルに存在しないリモート記事を削除

## 出力形式

実行結果はdiffスタイルで表示されます：

- `+` **作成**: 新規作成される記事（ローカルファイルパス）
- `~` **更新**: 更新される記事（ローカルファイルパス）
- `=` **スキップ**: 変更なしでスキップされる記事（ローカルファイルパス）
- `-` **削除**: 削除される記事（リモートURL、`-delete-orphan` 使用時のみ）

### 実行例
```
+ articles/new-article.md
~ articles/updated-article.md
= articles/unchanged-article.md
- https://example.hatenablog.com/entry/deleted-article
Created: 1, Updated: 1, Skipped: 1, Deleted: 1, Errors: 0
```

dry runモードでも同じ形式で表示されます（実際の変更は行われません）。

## 注意事項

- UUIDは記事の一意識別に使用されるため、重複しないように管理してください
- APIキーは適切に管理し、外部に漏洩しないよう注意してください
- 大量の記事を一度に同期する場合は、API制限に注意してください

### ⚠️ 削除機能について

`-delete-orphan` オプションは**非常に危険**です：

- ローカルに存在しないすべてのリモート記事が**永久に削除**されます
- 削除された記事は復元できません
- 必ず**事前に `-dry-run` で確認**してから実行してください
- 実行時には確認プロンプトが表示されます

**推奨される手順**：
1. まず `--dry-run --delete-orphan` で削除予定記事を確認
2. 削除されても良い記事かどうか慎重に検討
3. 問題ないことを確認してから実際の削除を実行