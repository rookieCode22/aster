package report

import _ "embed"

//go:embed fonts/wqy-zenhei-regular.ttf
var embeddedWQYFont []byte

func init() {
	wqyZenheiTTF = embeddedWQYFont
}
