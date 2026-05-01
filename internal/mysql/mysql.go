package mysql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	goMysql "github.com/go-sql-driver/mysql"
	"github.com/spf13/cast"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"

	"goblog/internal/config"
	"goblog/internal/pkg/slogx"
)

type GormLogger struct {
	*slog.Logger
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
}

func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= logger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		msg, rows := fc()
		l.ErrorContext(ctx, msg, "rows", rows, "duration", elapsed.Truncate(time.Millisecond), "err", err)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		msg, rows := fc()
		l.WarnContext(ctx, msg, "rows", rows, "slow_threshold", l.SlowThreshold, "duration", elapsed.Truncate(time.Millisecond))
	case l.LogLevel == logger.Info:
		msg, rows := fc()
		l.InfoContext(ctx, msg, "rows", rows, "duration", elapsed.Truncate(time.Millisecond))
	}
}

func fileWithSqlLineNum() string {
	for i := 8; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && strings.Contains(file, "repo.go") {
			return path.Base(file) + ":" + cast.ToString(line)
		}
	}
	return ""
}

func InitMySQL(opts *config.MySQLConfig, logLevel string) *gorm.DB {
	var enableTls bool
	if opts.TlsConfig != "" && opts.TlsRootCAFilePath != "" {
		enableTls = true
		pem, err := os.ReadFile(opts.TlsRootCAFilePath)
		if err != nil {
			log.Fatal("reade tls_root_ca_file_path error: ", err)
		}
		rootCertPool := x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			log.Fatal("failed to append tls pem root ca")
		}
		err = goMysql.RegisterTLSConfig(opts.TlsConfig, &tls.Config{
			RootCAs: rootCertPool,
		})
		if err != nil {
			log.Fatal("register tls config error: ", err)
		}
	}
	slog.Info("InitMySQL", "enable_tls", enableTls)
	dialector := &PrometheusDialector{Dialector: mysql.Open(opts.WriterEndpoint).(*mysql.Dialector), DBName: "writer"}
	level := logger.Warn
	if strings.ToUpper(logLevel) == "DEBUG" {
		level = logger.Info
	}
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			a.Value = slog.StringValue(fileWithSqlLineNum())
		}
		return a
	}

	gl := &GormLogger{
		Logger: slogx.New(
			slogx.WithAddSource(true),
			slogx.WithReplaceAttr(replace),
			slogx.WithHandler(slogx.TraceID),
		),
		LogLevel:      level,
		SlowThreshold: 200 * time.Millisecond,
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                 gl,
		SkipDefaultTransaction: opts.SkipDefaultTransaction,
		TranslateError:         true,
	})
	if err != nil {
		log.Fatal("open gorm error: ", err)
	}
	// Create MySQL ReaderEndpoints
	var replicas []gorm.Dialector
	for i, r := range opts.ReaderEndpoints {
		dialector := &PrometheusDialector{Dialector: mysql.Open(r).(*mysql.Dialector), DBName: fmt.Sprintf("reader-%d", i)}
		replicas = append(replicas, dialector)
	}
	sources := []gorm.Dialector{dialector}
	dbResolverCfg := dbresolver.Config{
		Sources:           sources,
		Replicas:          replicas,
		Policy:            dbresolver.RandomPolicy{},
		TraceResolverMode: true,
	}

	err = db.Use(dbresolver.Register(dbResolverCfg).
		SetMaxIdleConns(int(opts.MaxIdleConns)).
		SetMaxOpenConns(int(opts.MaxOpenConns)).
		SetConnMaxLifetime(opts.ConnMaxLifetime),
	)
	if err != nil {
		log.Fatal("gorm db resolver error: ", err)
	}
	printAddrs(opts)
	return db
}

func printAddrs(opts *config.MySQLConfig) {
	writerDSN, _ := goMysql.ParseDSN(opts.WriterEndpoint)
	var readerDSNs []string
	for _, endpoint := range opts.ReaderEndpoints {
		readerDSN, _ := goMysql.ParseDSN(endpoint)
		readerDSNs = append(readerDSNs, readerDSN.Addr)
	}
	slog.Info("InitMySQL", "writer_endpoint", writerDSN.Addr, "reader_endpoints", readerDSNs)
}

type PrometheusDialector struct {
	*mysql.Dialector
	DBName string
	*gorm.Config
}

func (dialector *PrometheusDialector) Apply(config *gorm.Config) error {
	dialector.Config = config
	return nil
}

// GormRepository 实现 PageRepository 接口
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository 创建一个新的 GormRepository 实例
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}
