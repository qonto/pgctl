package config

import (
	"github.com/qonto/pgctl/internal/postgres"
	"github.com/spf13/viper"
)

type Config struct {
	Databases map[string]postgres.DB `mapstructure:",remain"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(".pgctl")
	v.AddConfigPath(".")

	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = v.Unmarshal(c)
	return c, err
}

func Write(c *Config) error {
	v := viper.New()
	v.SetConfigName(".pgctl")
	v.AddConfigPath(".")

	for name, db := range c.Databases {
		v.Set(name+".host", db.Host)
		v.Set(name+".port", db.Port)
		v.Set(name+".database", db.Database)
		v.Set(name+".role", db.Role)
		v.Set(name+".password", db.Password)
	}

	return v.WriteConfigAs(".pgctl.yaml")
}

func (c *Config) Aliases() []string {
	aliases := make([]string, 0, len(c.Databases))
	for alias := range c.Databases {
		aliases = append(aliases, alias)
	}
	return aliases
}
