# mackerel-plugin-axslog

`mackerel-plugin-axslog` は、Nginx などのアクセスログを読み取り、Mackerel のカスタムメトリクスとして送信するプラグインです。JSON および LTSV 形式に対応し、応答時間・ステータスコードのフィールド名をログ形式に合わせて指定できます。

`mackerel-plugin-accesslog` の代替として、高速なログ解析を目的にしています。

日本語の紹介記事: https://kazeburo.hatenablog.com/entry/2019/04/05/093000

## インストール

Mackerel のプラグイン管理コマンド `mkr` を使用します。

```sh
mkr plugin install monitoring-forge/mackerel-plugin-axslog
```

リリースページからバイナリを直接ダウンロードして配置する方法も利用できます。

## 前提となるログ形式

各ログ行には、応答時間と HTTP ステータスコードが含まれている必要があります。デフォルトでは次のキーを読み取ります。

| 項目 | デフォルトのキー | 値 |
| --- | --- | --- |
| 応答時間 | `ptime` | 秒単位の数値。Nginx の `$request_time` など |
| ステータスコード | `status` | HTTP ステータスコードの整数 |

LTSV の例です。

```text
time:2026-08-28T12:00:00+09:00\tstatus:200\tptime:0.123\trequest:/api/items
```

JSON の例です。

```json
{"time":"2026-08-28T12:00:00+09:00","status":200,"ptime":0.123,"request":"/api/items"}
```

値が空または `-` の行、応答時間またはステータスコードを取得できない行は集計対象外です。

## 使い方

LTSV ログを直接読み取る例です。

```sh
mackerel-plugin-axslog \
  --logfile /var/log/nginx/access.log \
  --key-prefix nginx
```

応答時間のキーが `request_time` の場合は `--ptime-key` を指定します。

```sh
mackerel-plugin-axslog \
  --logfile /var/log/nginx/access.log \
  --key-prefix nginx \
  --ptime-key request_time
```

JSON ログを読む場合は `--format json` を指定します。

```sh
mackerel-plugin-axslog \
  --logfile /var/log/nginx/access.json.log \
  --format json \
  --key-prefix nginx_json \
  --ptime-key request_time
```

複数のログを合算するには、`--logfile` にカンマ区切りで指定します。

```sh
mackerel-plugin-axslog \
  --logfile /var/log/nginx/access.log,/var/log/nginx/api.log \
  --key-prefix nginx
```

複数の候補からステータスコードを取得したい場合は、`--status-key` を繰り返して指定します。指定順で最初に見つかったキーを使います。

```sh
mackerel-plugin-axslog \
  --logfile /var/log/nginx/access.log \
  --key-prefix nginx \
  --status-key status \
  --status-key upstream_status
```

### 主なオプション

| オプション | 説明 |
| --- | --- |
| `--logfile` | 読み取るログファイル。複数指定時はカンマ区切り |
| `--format` | ログ形式。`ltsv`（既定値）または `json` |
| `--key-prefix` | メトリクス名に使用する識別子。必須 |
| `--ptime-key` | 応答時間のキー名。既定値は `ptime` |
| `--status-key` | ステータスコードのキー名。複数回指定可能。既定値は `status` |
| `--filter` | 指定文字列を含む行だけを集計 |
| `--invert-filter` | `--filter` 指定時に、指定文字列を含まない行だけを集計 |
| `--skip-until-json` | JSON の前にプレーンテキストの接頭辞がある行で、最初の `{` までを無視 |
| `--max-read-size` | 1 回の実行で読む最大サイズ。`10MB`、`2GiB` のように指定。未指定時は LTSV が 1GB、JSON が 500MB |
| `-q`, `--quiet` | 解析時のログ出力を抑制 |

## mackerel-agent での設定

`/etc/mackerel-agent/mackerel-agent.conf` にプラグイン定義を追加します。`command` には、`command -v mackerel-plugin-axslog` で確認した絶対パスを指定してください。

```toml
[plugin.metrics.axslog]
command = "/usr/local/bin/mackerel-plugin-axslog --logfile /var/log/nginx/access.log --key-prefix nginx --ptime-key request_time"
```

JSON ログの場合の例です。

```toml
[plugin.metrics.axslog-json]
command = "/usr/local/bin/mackerel-plugin-axslog --logfile /var/log/nginx/access.json.log --format json --key-prefix nginx_json --ptime-key request_time"
```

設定後にエージェントを再起動します。

```sh
sudo systemctl restart mackerel-agent
```

macOS などでサービス管理方法が異なる場合は、その環境の手順で `mackerel-agent` を再起動してください。エージェントの実行ユーザーがログファイルを読み取れることも確認してください。

プラグインは読み取り済み位置を Mackerel プラグインの作業ディレクトリに保持するため、通常は新しく追記されたログ行だけを処理します。ログローテーション後も継続して監視できるよう、ログファイルへの読み取り権限を維持してください。

## 動作速度とリソース利用

LTSV は専用パーサー、JSON は必要なキーを直接検索するパーサーで処理し、応答時間の集計値とステータス別カウンターのみを保持します。全ログを構造体として展開しないため、大きなアクセスログを定期収集する用途を想定した実装です。

参考値として、100 万行(240MB)のログを 0.1 秒強で読み取れます。実際の処理時間は、ログ形式・各行のサイズ・フィルターの有無・ディスク性能などの実行環境により変動します。

初回実行時や位置情報が失われた場合は、未処理範囲を読み取るため、ログ量に応じて CPU・I/O を使用します。読み取り量を制限したい場合は `--max-read-size` を設定してください。複数ログは並列に処理しますが、レイテンシのパーセンタイル計算では対象ログ行の応答時間を保持するため、非常に大きな未処理区間ではメモリ使用量にも注意してください。

## 出力されるメトリクス

`--key-prefix nginx` を指定した場合、以下のメトリクスが出力されます。応答時間の単位はログに記録された値に従います。Nginx の `$request_time` を使う場合は秒です。

| メトリクス | 意味 |
| --- | --- |
| `axslog.latency_nginx.average` | 対象リクエストの平均応答時間 |
| `axslog.latency_nginx.90_percentile` | 応答時間の 90 パーセンタイル |
| `axslog.latency_nginx.95_percentile` | 応答時間の 95 パーセンタイル |
| `axslog.latency_nginx.99_percentile` | 応答時間の 99 パーセンタイル |
| `axslog.access_num_nginx.1xx_count` | 1xx 応答の毎秒件数 |
| `axslog.access_num_nginx.2xx_count` | 2xx 応答の毎秒件数 |
| `axslog.access_num_nginx.3xx_count` | 3xx 応答の毎秒件数 |
| `axslog.access_num_nginx.4xx_count` | 4xx 応答の毎秒件数。499 を除く |
| `axslog.access_num_nginx.499_count` | Nginx のクライアント切断を表す 499 応答の毎秒件数 |
| `axslog.access_num_nginx.5xx_count` | 5xx 応答の毎秒件数 |
| `axslog.access_total_nginx.count` | 全応答の毎秒件数 |
| `axslog.access_ratio_nginx.1xx_percentage` | 1xx 応答の割合（%） |
| `axslog.access_ratio_nginx.2xx_percentage` | 2xx 応答の割合（%） |
| `axslog.access_ratio_nginx.3xx_percentage` | 3xx 応答の割合（%） |
| `axslog.access_ratio_nginx.4xx_percentage` | 4xx 応答の割合（%）。499 を除く |
| `axslog.access_ratio_nginx.499_percentage` | 499 応答の割合（%） |
| `axslog.access_ratio_nginx.5xx_percentage` | 5xx 応答の割合（%） |

`access_num` と `access_total` は、今回処理したログの時間幅から算出するレートです。処理対象がない場合、レイテンシおよび割合のメトリクスは出力されません。

## ライセンス

[MIT License](LICENSE) で提供しています。
