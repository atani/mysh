package i18n

import (
	"os"
	"strings"
)

// Message keys for user-facing strings.
const (
	DriverMenuTitle       = "driver_menu_title"
	DriverMenuCLI         = "driver_menu_cli"
	DriverMenuNative      = "driver_menu_native"
	DriverMenuInvalid     = "driver_menu_invalid"
	NativeDriverWarning1  = "native_driver_warning_1"
	NativeDriverWarning2  = "native_driver_warning_2"
	ImportNoConnections   = "import_no_connections"
	ImportPasswordPrompt  = "import_password_prompt"
	ImportNameConflict    = "import_name_conflict"
	ImportSuccess         = "import_success"
	ImportMaskAsk         = "import_mask_ask"
	ImportMaskPrompt      = "import_mask_prompt"
	ImportMaskApplied     = "import_mask_applied"
	ImportPostHint        = "import_post_hint"
)

var en = map[string]string{
	DriverMenuTitle:       "Connection driver:",
	DriverMenuCLI:         "  1) cli    - mysql/mycli command-line client",
	DriverMenuNative:      "  2) native - Go driver (supports MySQL 4.x old_password)",
	DriverMenuInvalid:     "  Invalid choice. Enter 1-2 or driver name.",
	NativeDriverWarning1:  "  ⚠ The native driver supports MySQL 4.x old_password authentication,",
	NativeDriverWarning2:  "    but old_password is cryptographically weak. Use only for legacy systems.",
	ImportNoConnections:   "No MySQL connections found in %s.",
	ImportPasswordPrompt:  "Password cannot be imported from %s. Please enter it now.",
	ImportNameConflict:    "Connection %q already exists. Enter a new name:",
	ImportSuccess:         "Imported %d connection(s) from %s.",
	ImportMaskAsk:         "Default mask columns: %s",
	ImportMaskPrompt:      "Apply output masking to protect sensitive data?",
	ImportMaskApplied:     "Applied mask settings. Query results will automatically hide sensitive columns.",
	ImportPostHint:        "To set up masking later, run:",
}

var ja = map[string]string{
	DriverMenuTitle:       "接続ドライバ:",
	DriverMenuCLI:         "  1) cli    - mysql/mycli コマンドラインクライアント",
	DriverMenuNative:      "  2) native - Go ドライバ (MySQL 4.x old_password 対応)",
	DriverMenuInvalid:     "  無効な選択です。1-2 またはドライバ名を入力してください。",
	NativeDriverWarning1:  "  ⚠ native ドライバは MySQL 4.x の old_password 認証に対応していますが、",
	NativeDriverWarning2:  "    old_password はセキュリティ的に脆弱です。レガシーシステムへの接続用途に限定してください。",
	ImportNoConnections:   "%s に MySQL 接続が見つかりませんでした。",
	ImportPasswordPrompt:  "%s からパスワードはインポートできません。手動で入力してください。",
	ImportNameConflict:    "接続 %q は既に存在します。別の名前を入力してください:",
	ImportSuccess:         "%d 件の接続を %s からインポートしました。",
	ImportMaskAsk:         "デフォルトのマスク対象カラム: %s",
	ImportMaskPrompt:      "個人情報の秘匿化（出力マスク）を設定しますか？",
	ImportMaskApplied:     "マスク設定を適用しました。クエリ結果の機密カラムが自動で秘匿化されます。",
	ImportPostHint:        "後からマスクを設定するには:",
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
