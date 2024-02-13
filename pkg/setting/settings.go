package setting

import (
	"context"
	"github.com/kelseyhightower/envconfig"
	"log"
)

type LocalFSConfig struct {
	RootDirectory string `yaml:"rootDirectory" envconfig:"LOCAL_FS_ROOT_DIRECTORY"`
}

type CorsConfig struct {
	AllowOrigins []string `yaml:"allowOrigins" envconfig:"CORS_ALLOW_ORIGINS"`
}

type Config struct {
	LocalFSConfig LocalFSConfig `yaml:"localFS"`
	CorsConfig    CorsConfig    `yaml:"cors"`

	Context struct {
		Context context.Context    `yaml:"-" envconfig:"-"`
		Cancel  context.CancelFunc `yaml:"-" envconfig:"-"`
	} `yaml:"-"`
}

var Settings = &Config{}

func init() {
	SetupSettings()
}

// SetupSettings initialize the configuration instance
func SetupSettings() {

	err := envconfig.Process("", Settings)
	if err != nil {
		log.Fatalf("setting, fail to get from env': %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	Settings.Context.Context = ctx
	Settings.Context.Cancel = cancel
}
