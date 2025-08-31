# Repository Guidelines

## コミュニケーションポリシー
- 本リポジトリでは日本語でコミュニケーションを行います（Issue、PR、コメント、コミット、ドキュメント）。
- 公開上の都合で英語が必要な場合は、日本語を優先し、必要に応じて英語を併記してください。

## プロジェクト構成とモジュール
- `cmd/`: エントリポイント（`http-server`, `debug`）
- `core/`: 低レベル I/O（io_uring、ソケット、`engine/`、`event/`、`errors/`）
- `server/`: サーバ統括（イベントループ、`network_server.go`）
- `transport/`: プロトコル実装（HTTP: `parser.go`, `router.go`, `handlers.go`）
- `middleware/`: ミドルウェアパイプライン
- `application/`: 機能層（`live/`, `room/`）
- `bin/`: ビルド成果物、`scripts/`, `sample/`: 補助
- 備考: Linux 専用（直接 syscalls と io_uring を使用）

## ビルド・実行・開発コマンド
- ビルド（HTTP サーバ）: `go build ./cmd/http-server`
- 実行（例）: `go run ./cmd/http-server/main.go -host 0.0.0.0 -port 8080`
- Make ビルド（Linux/AMD64）: `make build`（`./bin` に出力）
- リモート実行: `make run`（Makefile の `VM_*` 設定を使用）
- フォーマット/静的解析: `go fmt ./...` / `go vet ./...`
- 依存解決: `go mod tidy`

## コーディングスタイルと命名規約
- インデント 2 スペース、LF、UTF-8（`.editorconfig` 準拠）。
- Go 慣用: パッケージ名は小文字、ファイルは `snake_case`（例: `network_server.go`）。
- レイヤ分離を厳守（`core` ↔ `transport` ↔ `server`）。循環依存を避ける。
- 新規コードは `panic` を避け、エラーを返す。小さく明確な関数を心掛ける。

## テストガイドライン
- 現状テスト未整備。`testing` を用い `*_test.go` を追加し段階的に拡充。
- 実行: `go test ./... -v -cover`
- 重点: `transport/http` のパーサ・ルータ、`core/engine` はフェイクで境界を検証。

## コミットと Pull Request
- コミット: 簡潔な命令形の日本語（例: 「プロトコル IF を追加」）。1 コミット 1 目的。
- 参考: Git 履歴では短い要約と要点重視（例: 「ソケットの再利用を許可」）。
- Issue 連携: 必要に応じて `Fix #123` 等を併記。
- PR: 目的・要約・変更点・確認手順・パフォーマンス影響（必要ならベンチ/ログ）・Linux/io_uring 前提を記載。スクリーンショットやログは適宜添付。

## セキュリティと設定の注意
- io_uring 対応の Linux カーネルが必須。開発時は `-host 127.0.0.1` を推奨。
- 公開環境でのポート/IF バインドを慎重に。Makefile の `VM_*` は実環境に合わせて調整。
