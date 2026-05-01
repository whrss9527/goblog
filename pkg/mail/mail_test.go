package mail

import (
	"os"
	"strconv"
	"testing"
)

// TestSend 通过环境变量驱动，CI / 公开仓库下默认跳过。
// 本地运行示例：
//
//	MAIL_HOST=smtp.qq.com MAIL_PORT=465 \
//	MAIL_USER=xxx@qq.com MAIL_PASS=xxx \
//	MAIL_TO=xxx@example.com \
//	go test ./pkg/mail/...
func TestSend(t *testing.T) {
	host := os.Getenv("MAIL_HOST")
	user := os.Getenv("MAIL_USER")
	pass := os.Getenv("MAIL_PASS")
	to := os.Getenv("MAIL_TO")
	if host == "" || user == "" || pass == "" || to == "" {
		t.Skip("mail env not set; skipping (set MAIL_HOST/MAIL_USER/MAIL_PASS/MAIL_TO to enable)")
	}
	port, err := strconv.Atoi(os.Getenv("MAIL_PORT"))
	if err != nil || port == 0 {
		port = 465
	}

	options := &Options{
		MailHost: host,
		MailPort: port,
		MailUser: user,
		MailPass: pass,
		MailTo:   to,
		Subject:  "subject",
		Body:     "body",
	}
	if err := Send(options); err != nil {
		t.Errorf("Mail Send error: %v", err)
		return
	}
	t.Log("success")
}
