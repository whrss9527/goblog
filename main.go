package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"goblog/internal/config"
	ginpkg "goblog/internal/pkg/gin"
	"goblog/internal/pkg/view"
	"goblog/internal/routers"
)

var (
	configPath   string
	hashPassword bool
)

func main() {
	flag.StringVar(&configPath, "config", "./conf/dev.yaml", "yaml config file to be load, e.g: -config=/local/config.yaml")
	flag.BoolVar(&hashPassword, "hash-password", false, "read a password from stdin and print the bcrypt hash for app.admin_password_hash")
	flag.Parse()
	if hashPassword {
		os.Exit(printPasswordHash(os.Stdin, os.Stdout, os.Stderr))
	}
	conf := config.LoadConfig(configPath)

	if conf.Server == nil || conf.Server.HttpPort == 0 {
		slog.Error("server.http_port is not configured")
		return
	}

	if conf.App != nil && !conf.App.ConfigAdmin() && conf.App.GitRepo != "" {
		slog.Warn("the admin account is read from users.json in the content repository; if that repository is public its password hash is too — " +
			"set app.admin_email / app.admin_password_hash (goblog -hash-password), remove users.json from the repository and change the password")
	}

	view.InitTemplates()
	server := routers.NewServer(conf)
	router := ginpkg.InitGinConfig(conf.App.Mode)
	cleanup := server.InitRouter(router)
	defer cleanup()
	ginpkg.RunGin(router, conf.Server.HttpPort, conf.Server.GracefulShutdownTimeout)
}

// printPasswordHash implements -hash-password: one line from in, the bcrypt
// hash to out. The password is read from stdin so it never shows up in the
// shell history or the process list.
func printPasswordHash(in io.Reader, out, errOut io.Writer) int {
	fmt.Fprint(errOut, "password: ")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Fprintln(errOut, "read password:", err)
		return 1
	}
	password := strings.TrimRight(line, "\r\n")
	if len(password) < 8 {
		fmt.Fprintln(errOut, "\nthe password needs at least 8 characters")
		return 1
	}
	if len(password) > 72 {
		fmt.Fprintln(errOut, "\nbcrypt only looks at the first 72 bytes, please use a shorter password")
		return 1
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Fprintln(errOut, "hash password:", err)
		return 1
	}
	fmt.Fprintln(errOut)
	fmt.Fprintln(out, string(hash))
	return 0
}
