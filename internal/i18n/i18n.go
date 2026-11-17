package i18n

import (
	"os"
	"strings"
)

// Message keys for user-facing strings.
const (
	DriverMenuTitle            = "driver_menu_title"
	DriverMenuCLI              = "driver_menu_cli"
	DriverMenuNative           = "driver_menu_native"
	DriverMenuInvalid          = "driver_menu_invalid"
	NativeDriverWarning1       = "native_driver_warning_1"
	NativeDriverWarning2       = "native_driver_warning_2"
	ImportNoConnections        = "import_no_connections"
	ImportPasswordPrompt       = "import_password_prompt"
	ImportNameConflict         = "import_name_conflict"
	ImportSuccess              = "import_success"
	ImportMaskAsk              = "import_mask_ask"
	ImportMaskPrompt           = "import_mask_prompt"
	ImportMaskApplied          = "import_mask_applied"
	ImportPostHint             = "import_post_hint"
	ImportPingHint             = "import_ping_hint"
	ImportPasswordInput        = "import_password_input"
	ImportSharedPasswordInput  = "import_shared_password_input"
	ImportPasswordRetry        = "import_password_retry"
	ImportAddNoPassword        = "import_add_no_password"
	ImportConnFailed           = "import_conn_failed"
	ImportRetryHint            = "import_retry_hint"
	ImportRetryExhausted       = "import_retry_exhausted"
	ImportSharedPasswordFailed = "import_shared_password_failed"

	// add command: interactive prompts and labels
	AddUseSSH             = "add_use_ssh"
	AddSSHHost            = "add_ssh_host"
	AddSSHPort            = "add_ssh_port"
	AddSSHUser            = "add_ssh_user"
	AddSSHKeyDefault      = "add_ssh_key_default"
	AddSSHKey             = "add_ssh_key"
	AddDBHost             = "add_db_host"
	AddDBPort             = "add_db_port"
	AddDBUser             = "add_db_user"
	AddDBPassword         = "add_db_password"
	AddDBPasswordKeep     = "add_db_password_keep"
	AddSSLMode            = "add_sslmode"
	AddDatabaseName       = "add_database_name"
	AddProdMaskNotice     = "add_prod_mask_notice"
	AddProdConfirm        = "add_prod_confirm"
	AddMaskColumns        = "add_mask_columns"
	AddMaskColumnsBracket = "add_mask_columns_bracket"
	AddConnName           = "add_conn_name"
	AddConnAdded          = "add_conn_added"
	AddRedashAdded        = "add_redash_added"
	AddTestConn           = "add_test_conn"
	AddConnFailed         = "add_conn_failed"
	AddSkippedFix         = "add_skipped_fix"
	AddRetesting          = "add_retesting"
	AddConnOK             = "add_conn_ok"
	AddChoice             = "add_choice"
	AddRequiredSuffix     = "add_required_suffix"
	AddDataSourceID       = "add_data_source_id"
	AddRedashAPIKey       = "add_redash_api_key"

	// add command: database type menu and connection-string input
	AddDBTypeTitle        = "add_db_type_title"
	AddDBTypeMySQL        = "add_db_type_mysql"
	AddDBTypePostgres     = "add_db_type_postgres"
	AddDBTypeSupabase     = "add_db_type_supabase"
	AddDBTypeInvalid      = "add_db_type_invalid"
	AddSupabaseURLHelp1   = "add_supabase_url_help_1"
	AddSupabaseURLHelp2   = "add_supabase_url_help_2"
	AddSupabaseURLHelp3   = "add_supabase_url_help_3"
	AddSupabaseURL        = "add_supabase_url"
	AddURLNoPassword      = "add_url_no_password"
	AddURLPasswordWarning = "add_url_password_warning"
	AddURLIgnoredParam    = "add_url_ignored_param"

	// add command: fix menu
	AddFixTitle      = "add_fix_title"
	AddFixDBHostPort = "add_fix_db_host_port"
	AddFixDBUserPass = "add_fix_db_user_pass"
	AddFixDBName     = "add_fix_db_name"
	AddFixSSH        = "add_fix_ssh"
	AddFixSkip5      = "add_fix_skip_5"
	AddFixSkip4      = "add_fix_skip_4"
	AddInvalidChoice = "add_invalid_choice"

	// add command: environment menu
	AddEnvTitle   = "add_env_title"
	AddEnvProd    = "add_env_prod"
	AddEnvStg     = "add_env_stg"
	AddEnvDev     = "add_env_dev"
	AddEnvInvalid = "add_env_invalid"

	// add command: master password
	AddMasterSetupTitle = "add_master_setup_title"
	AddMasterSetupDesc  = "add_master_setup_desc"
	AddMasterPrompt     = "add_master_prompt"
	AddMasterConfirm    = "add_master_confirm"
	AddMasterSaved      = "add_master_saved"

	// add command: user-facing validation errors
	ErrNameRequired     = "err_name_required"
	ErrConnExists       = "err_conn_exists"
	ErrMasterEmpty      = "err_master_empty"
	ErrPasswordMismatch = "err_password_mismatch"
	ErrAPIKeyRequired   = "err_api_key_required"

	// master password resolution (shared by interactive and MCP contexts)
	ErrMasterUnavailable = "err_master_unavailable"

	// mcp command: runtime messages returned to the agent/user
	McpNoConnections     = "mcp_no_connections"
	McpQueryOK           = "mcp_query_ok"
	McpErrSQLRequired    = "mcp_err_sql_required"
	McpErrPDFUnsupported = "mcp_err_pdf_unsupported"
	McpErrTablesRedash   = "mcp_err_tables_redash"
	McpPingOK            = "mcp_ping_ok"
	McpPingOKRedash      = "mcp_ping_ok_redash"
	McpPingFailed        = "mcp_ping_failed"
	McpPingFailedRedash  = "mcp_ping_failed_redash"
)

var en = map[string]string{
	DriverMenuTitle:            "Connection driver:",
	DriverMenuCLI:              "  1) cli    - mysql/mycli command-line client",
	DriverMenuNative:           "  2) native - Go driver (supports MySQL 4.x old_password)",
	DriverMenuInvalid:          "  Invalid choice. Enter 1-2 or driver name.",
	NativeDriverWarning1:       "  ⚠ The native driver supports MySQL 4.x old_password authentication,",
	NativeDriverWarning2:       "    but old_password is cryptographically weak. Use only for legacy systems.",
	ImportNoConnections:        "No MySQL connections found in %s.",
	ImportPasswordPrompt:       "Password cannot be imported from %s. Please enter it now.",
	ImportNameConflict:         "Connection %q already exists. Enter a new name:",
	ImportSuccess:              "Imported %d connection(s) from %s.",
	ImportMaskAsk:              "Default mask columns: %s",
	ImportMaskPrompt:           "Apply output masking to protect sensitive data?",
	ImportMaskApplied:          "Applied mask settings. Query results will automatically hide sensitive columns.",
	ImportPostHint:             "To set up masking later, run:",
	ImportPingHint:             "Verify connections with: mysh ping <name>",
	ImportPasswordInput:        "%s password (Enter to skip): ",
	ImportSharedPasswordInput:  "Password for all selected DB connections (Enter to skip): ",
	ImportPasswordRetry:        "%s password (retry): ",
	ImportAddNoPassword:        "  Add without password?",
	ImportConnFailed:           "  Connection failed: %v",
	ImportRetryHint:            "  Re-enter password to try again.",
	ImportRetryExhausted:       "  Adding with last entered password. Fix later with `mysh edit`.",
	ImportSharedPasswordFailed: "%d connection(s) failed the connection test with the shared password. Fix them with:",

	AddUseSSH:             "Use SSH tunnel?",
	AddSSHHost:            "SSH host",
	AddSSHPort:            "SSH port",
	AddSSHUser:            "SSH user",
	AddSSHKeyDefault:      "SSH key path (empty for default)",
	AddSSHKey:             "SSH key path",
	AddDBHost:             "%s host",
	AddDBPort:             "%s port",
	AddDBUser:             "%s user",
	AddDBPassword:         "%s password: ",
	AddDBPasswordKeep:     "%s password (Enter to keep): ",
	AddSSLMode:            "SSL mode (disable/allow/prefer/require/verify-ca/verify-full)",
	AddDatabaseName:       "Database name",
	AddProdMaskNotice:     "  ⚠ Masking is always enabled for production connections.",
	AddProdConfirm:        "  Are you sure this is a production connection?",
	AddMaskColumns:        "Columns to mask (comma-separated, wildcards OK)",
	AddMaskColumnsBracket: "Mask columns [%s]: ",
	AddConnName:           "Connection name",
	AddConnAdded:          "Connection %q added.",
	AddRedashAdded:        "Connection %q (Redash) added.",
	AddTestConn:           "Test connection?",
	AddConnFailed:         "Connection failed: %v",
	AddSkippedFix:         "Skipped. You can fix settings later with `mysh edit`.",
	AddRetesting:          "Retesting...",
	AddConnOK:             "Connection %q: OK (%s)",
	AddChoice:             "Choice",
	AddRequiredSuffix:     "  %s is required.",
	AddDataSourceID:       "Data source ID",
	AddRedashAPIKey:       "Redash API key: ",

	AddDBTypeTitle:        "Database type:",
	AddDBTypeMySQL:        "  1) MySQL",
	AddDBTypePostgres:     "  2) PostgreSQL",
	AddDBTypeSupabase:     "  3) Supabase (PostgreSQL)",
	AddDBTypeInvalid:      "  Invalid choice. Enter 1-3 or mysql/postgres/supabase.",
	AddSupabaseURLHelp1:   "Paste the connection string from the Connect dialog in the Supabase dashboard.",
	AddSupabaseURLHelp2:   "  Direct connection is IPv6-only by default; on IPv4 networks use the Session pooler (port 5432).",
	AddSupabaseURLHelp3:   "  Use the Transaction pooler (port 6543) for short-lived connections.",
	AddSupabaseURL:        "Connection string",
	AddURLNoPassword:      "The connection string has no password (or only the [YOUR-PASSWORD] placeholder).",
	AddURLPasswordWarning: "  ⚠ A password passed in --url stays in your shell history and the process list. Leave it out (or keep [YOUR-PASSWORD]) to be prompted instead.",
	AddURLIgnoredParam:    "[mysh] warning: ignoring unsupported connection string parameter %q; it is neither saved nor used.",

	AddFixTitle:      "What would you like to fix?",
	AddFixDBHostPort: "  1) %s host/port",
	AddFixDBUserPass: "  2) %s user/password",
	AddFixDBName:     "  3) Database name",
	AddFixSSH:        "  4) SSH settings",
	AddFixSkip5:      "  5) Skip",
	AddFixSkip4:      "  4) Skip",
	AddInvalidChoice: "  Invalid choice.",

	AddEnvTitle:   "Environment:",
	AddEnvProd:    "  1) production",
	AddEnvStg:     "  2) staging",
	AddEnvDev:     "  3) development",
	AddEnvInvalid: "  Invalid choice. Enter 1-3 or environment name.",

	AddMasterSetupTitle: "Setting up master password for the first time.",
	AddMasterSetupDesc:  "This password protects your stored database credentials.",
	AddMasterPrompt:     "Master password: ",
	AddMasterConfirm:    "Confirm master password: ",
	AddMasterSaved:      "Master password saved to %s.",

	ErrNameRequired:     "name is required",
	ErrConnExists:       "connection %q already exists",
	ErrMasterEmpty:      "master password cannot be empty",
	ErrPasswordMismatch: "passwords do not match",
	ErrAPIKeyRequired:   "API key is required",

	ErrMasterUnavailable: "master password is required to decrypt stored credentials but is not available: set MYSH_MASTER_PASSWORD, or run any mysh command interactively once so it is saved to the OS credential store",

	McpNoConnections:     "No connections configured. Run `mysh add` to add one.",
	McpQueryOK:           "Query OK",
	McpErrSQLRequired:    "the \"sql\" argument is required",
	McpErrPDFUnsupported: "pdf format is not supported over MCP",
	McpErrTablesRedash:   "mysh_tables is not supported for Redash connections",
	McpPingOK:            "Connection %q: OK",
	McpPingOKRedash:      "Connection %q (Redash): OK",
	McpPingFailed:        "connection %q: FAILED",
	McpPingFailedRedash:  "connection %q (Redash): FAILED",
}

var ja = map[string]string{
	DriverMenuTitle:            "接続ドライバ:",
	DriverMenuCLI:              "  1) cli    - mysql/mycli コマンドラインクライアント",
	DriverMenuNative:           "  2) native - Go ドライバ (MySQL 4.x old_password 対応)",
	DriverMenuInvalid:          "  無効な選択です。1-2 またはドライバ名を入力してください。",
	NativeDriverWarning1:       "  ⚠ native ドライバは MySQL 4.x の old_password 認証に対応していますが、",
	NativeDriverWarning2:       "    old_password はセキュリティ的に脆弱です。レガシーシステムへの接続用途に限定してください。",
	ImportNoConnections:        "%s に MySQL 接続が見つかりませんでした。",
	ImportPasswordPrompt:       "%s からパスワードはインポートできません。手動で入力してください。",
	ImportNameConflict:         "接続 %q は既に存在します。別の名前を入力してください:",
	ImportSuccess:              "%d 件の接続を %s からインポートしました。",
	ImportMaskAsk:              "デフォルトのマスク対象カラム: %s",
	ImportMaskPrompt:           "個人情報の秘匿化（出力マスク）を設定しますか？",
	ImportMaskApplied:          "マスク設定を適用しました。クエリ結果の機密カラムが自動で秘匿化されます。",
	ImportPostHint:             "後からマスクを設定するには:",
	ImportPingHint:             "接続を確認するには: mysh ping <name>",
	ImportPasswordInput:        "%s パスワード (Enter でスキップ): ",
	ImportSharedPasswordInput:  "選択した全DB接続で使うパスワード (Enter でスキップ): ",
	ImportPasswordRetry:        "%s パスワード (再入力): ",
	ImportAddNoPassword:        "  パスワードなしで追加しますか？",
	ImportConnFailed:           "  接続失敗: %v",
	ImportSharedPasswordFailed: "%d 件の接続が共有パスワードでの接続テストに失敗しました。次で修正できます:",
	ImportRetryHint:            "  パスワードを再入力してください。",
	ImportRetryExhausted:       "  最後に入力したパスワードで追加します。後から `mysh edit` で修正できます。",

	AddUseSSH:             "SSH トンネルを使いますか？",
	AddSSHHost:            "SSH ホスト",
	AddSSHPort:            "SSH ポート",
	AddSSHUser:            "SSH ユーザー",
	AddSSHKeyDefault:      "SSH 鍵のパス (空でデフォルト)",
	AddSSHKey:             "SSH 鍵のパス",
	AddDBHost:             "%s ホスト",
	AddDBPort:             "%s ポート",
	AddDBUser:             "%s ユーザー",
	AddDBPassword:         "%s パスワード: ",
	AddDBPasswordKeep:     "%s パスワード (Enter で変更なし): ",
	AddSSLMode:            "SSL モード (disable/allow/prefer/require/verify-ca/verify-full)",
	AddDatabaseName:       "データベース名",
	AddProdMaskNotice:     "  ⚠ 本番接続では常にマスクが有効になります。",
	AddProdConfirm:        "  本当に本番接続でよろしいですか？",
	AddMaskColumns:        "マスク対象カラム (カンマ区切り、ワイルドカード可)",
	AddMaskColumnsBracket: "マスク対象カラム [%s]: ",
	AddConnName:           "接続名",
	AddConnAdded:          "接続 %q を追加しました。",
	AddRedashAdded:        "接続 %q (Redash) を追加しました。",
	AddTestConn:           "接続をテストしますか？",
	AddConnFailed:         "接続に失敗しました: %v",
	AddSkippedFix:         "スキップしました。設定は後から `mysh edit` で修正できます。",
	AddRetesting:          "再テストしています…",
	AddConnOK:             "接続 %q: OK (%s)",
	AddChoice:             "選択",
	AddRequiredSuffix:     "  %s は必須です。",
	AddDataSourceID:       "データソース ID",
	AddRedashAPIKey:       "Redash API キー: ",

	AddDBTypeTitle:        "データベースの種類:",
	AddDBTypeMySQL:        "  1) MySQL",
	AddDBTypePostgres:     "  2) PostgreSQL",
	AddDBTypeSupabase:     "  3) Supabase (PostgreSQL)",
	AddDBTypeInvalid:      "  無効な選択です。1-3 または mysql/postgres/supabase を入力してください。",
	AddSupabaseURLHelp1:   "Supabase ダッシュボードの Connect ダイアログにある接続文字列を貼り付けてください。",
	AddSupabaseURLHelp2:   "  Direct connection は既定で IPv6 のみです。IPv4 環境では Session pooler (ポート 5432) を使ってください。",
	AddSupabaseURLHelp3:   "  短時間の接続を繰り返す用途には Transaction pooler (ポート 6543) を使えます。",
	AddSupabaseURL:        "接続文字列",
	AddURLNoPassword:      "接続文字列にパスワードがありません (または [YOUR-PASSWORD] のままです)。",
	AddURLPasswordWarning: "  ⚠ --url に含めたパスワードはシェル履歴とプロセス一覧に残ります。パスワードを省く (または [YOUR-PASSWORD] のままにする) とプロンプトで入力できます。",
	AddURLIgnoredParam:    "[mysh] 警告: 接続文字列のパラメータ %q には対応していないため無視します (保存も使用もしません)。",

	AddFixTitle:      "どの項目を修正しますか？",
	AddFixDBHostPort: "  1) %s ホスト/ポート",
	AddFixDBUserPass: "  2) %s ユーザー/パスワード",
	AddFixDBName:     "  3) データベース名",
	AddFixSSH:        "  4) SSH 設定",
	AddFixSkip5:      "  5) スキップ",
	AddFixSkip4:      "  4) スキップ",
	AddInvalidChoice: "  無効な選択です。",

	AddEnvTitle:   "環境:",
	AddEnvProd:    "  1) production (本番)",
	AddEnvStg:     "  2) staging (検証)",
	AddEnvDev:     "  3) development (開発)",
	AddEnvInvalid: "  無効な選択です。1-3 または環境名を入力してください。",

	AddMasterSetupTitle: "マスターパスワードを初めて設定します。",
	AddMasterSetupDesc:  "このパスワードは保存する DB 認証情報を保護します。",
	AddMasterPrompt:     "マスターパスワード: ",
	AddMasterConfirm:    "マスターパスワード (確認): ",
	AddMasterSaved:      "マスターパスワードを %s に保存しました。",

	ErrNameRequired:     "接続名は必須です",
	ErrConnExists:       "接続 %q は既に存在します",
	ErrMasterEmpty:      "マスターパスワードを空にはできません",
	ErrPasswordMismatch: "パスワードが一致しません",
	ErrAPIKeyRequired:   "API キーは必須です",

	ErrMasterUnavailable: "保存された認証情報の復号にマスターパスワードが必要ですが、取得できませんでした: MYSH_MASTER_PASSWORD を設定するか、mysh コマンドを一度対話的に実行して OS の資格情報ストアに保存してください",

	McpNoConnections:     "接続が設定されていません。`mysh add` で追加してください。",
	McpQueryOK:           "Query OK",
	McpErrSQLRequired:    "\"sql\" 引数は必須です",
	McpErrPDFUnsupported: "MCP では pdf 形式はサポートしていません",
	McpErrTablesRedash:   "mysh_tables は Redash 接続では使用できません",
	McpPingOK:            "接続 %q: OK",
	McpPingOKRedash:      "接続 %q (Redash): OK",
	McpPingFailed:        "接続 %q: 失敗",
	McpPingFailedRedash:  "接続 %q (Redash): 失敗",
}

var locales = map[string]map[string]string{
	"en": en,
	"ja": ja,
}

var current map[string]string

func init() {
	current = detect()
}

// T returns the translated message for the given key.
// Falls back to English if the key is not found in the current locale.
func T(key string) string {
	if msg, ok := current[key]; ok {
		return msg
	}
	if msg, ok := en[key]; ok {
		return msg
	}
	return key
}

func detect() map[string]string {
	for _, env := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := os.Getenv(env); val != "" {
			lang := strings.SplitN(val, "_", 2)[0]
			lang = strings.SplitN(lang, ".", 2)[0]
			lang = strings.ToLower(lang)
			if loc, ok := locales[lang]; ok {
				return loc
			}
		}
	}
	return en
}
