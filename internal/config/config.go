package config

import (
	"bytes"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/viper"

	"goblog/internal/util"
)

type (
	Config struct {
		LogLevel string        `mapstructure:"log_level"`
		Server   *ServerConfig `mapstructure:"server"`
		MySQL    *MySQLConfig  `mapstructure:"mysql"`
		Redis    *RedisConfig  `mapstructure:"redis"`
		App      *AppConfig    `mapstructure:"app"`
	}
	AppConfig struct {
		Name          string `mapstructure:"name"`
		Version       bool   `mapstructure:"version"`
		Mode          string `mapstructure:"mode"`
		Addr          string `mapstructure:"addr"`
		Host          string `mapstructure:"host"`
		Cdn           string `mapstructure:"cdn"`
		Music         string `mapstructure:"music"`
		SessionSecret string `mapstructure:"session_secret"`
		DataDir       string `mapstructure:"data_dir"`
		GitRepo       string `mapstructure:"git_repo"`
		GitToken      string `mapstructure:"git_token"`
	}
	ServerConfig struct {
		Host                     string        `mapstructure:"host"`
		GrpcPort                 uint32        `mapstructure:"grpc_port"`
		HttpPort                 uint32        `mapstructure:"http_port"`
		GrpcEnableReflection     bool          `mapstructure:"grpc_enable_reflection"`
		GracefulShutdownTimeout  time.Duration `mapstructure:"graceful_shutdown_timeout"`
		ApiSigningKey            string        `mapstructure:"api_signing_key"`
		ForbiddenCommandRedisTTL time.Duration `mapstructure:"forbidden_command_redis_ttl"`
	}

	MySQLConfig struct {
		WriterEndpoint         string        `mapstructure:"writer_endpoint"`
		ReaderEndpoints        []string      `mapstructure:"reader_endpoints"`
		TlsConfig              string        `mapstructure:"tls_config"`
		TlsRootCAFilePath      string        `mapstructure:"tls_root_ca_file_path"` // aws aurora tls root ca file path
		MaxOpenConns           uint16        `mapstructure:"max_open_conns"`
		MaxIdleConns           uint16        `mapstructure:"max_idle_conns"`
		ConnMaxLifetime        time.Duration `mapstructure:"conn_max_lifetime"`
		SkipDefaultTransaction bool          `mapstructure:"skip_default_transaction"`
	}

	RedisConfig struct {
		Addrs           []string      `mapstructure:"addrs"`
		EnableTLS       bool          `mapstructure:"enable_tls"`
		Username        string        `mapstructure:"username"`
		Password        string        `mapstructure:"password"`
		PoolSize        int           `mapstructure:"pool_size"`
		MinIdleConns    int           `mapstructure:"min_idle_conns"`
		MaxIdleConns    int           `mapstructure:"max_idle_conns"`
		MaxActiveConns  int           `mapstructure:"max_active_conns"`
		ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	}
)

var (
	defaultConfigYamlString = []byte(`
log_level: debug
server:
  host: 0.0.0.0
  grpc_port: 8080
  http_port: 8090
  grpc_enable_reflection: true
  graceful_shutdown_timeout: 15s
  forbidden_command_redis_ttl: 720h
mysql:
  max_open_conns: 20
  max_idle_conns: 20
  skip_default_transaction: true
redis:
  enable_tls: false
  min_idle_conns: 25
  max_idle_conns: 50
  max_active_conns: 50
  conn_max_idle_time: -1
`)
)

func LoadConfig(config string) *Config {
	v := viper.New()
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBuffer(defaultConfigYamlString))
	if err != nil {
		slog.Error("LoadConfig", "err", err)
		util.Exit(1)
	}
	_, err = os.Stat(config)
	if err == nil {
		v.SetConfigFile(config)
		if err = v.MergeInConfig(); err != nil {
			slog.Error("LoadConfig", "err", err)
			util.Exit(1)
		}
	}
	conf := &Config{}
	err = v.Unmarshal(&conf)
	if err != nil {
		slog.Error("LoadConfig", "err", err)
		util.Exit(1)
	}
	return conf
}
