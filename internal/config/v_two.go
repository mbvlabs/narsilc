package config

import (
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v3"
)

func v2ParseConfig(rd io.Reader) (Config, error) {
	dec := yaml.NewDecoder(rd)
	dec.KnownFields(true)
	var conf Config
	if err := dec.Decode(&conf); err != nil {
		return conf, err
	}
	if conf.Version == "" {
		return conf, ErrMissingVersion
	}
	if conf.Version != "2" {
		return conf, ErrUnknownVersion
	}
	if len(conf.SQL) == 0 {
		return conf, ErrNoPackages
	}
	if err := conf.validateGlobalOverrides(); err != nil {
		return conf, err
	}
	for j := range conf.SQL {
		if conf.SQL[j].Engine == "" {
			return conf, ErrMissingEngine
		}
		if conf.SQL[j].Gen.Go != nil {
			if conf.SQL[j].Gen.Go.Out == "" {
				return conf, ErrNoPackagePath
			}
		}
		if conf.SQL[j].StrictOrderBy == nil {
			defaultValidate := true
			conf.SQL[j].StrictOrderBy = &defaultValidate
		}
	}
	return conf, nil
}

func (c *Config) validateGlobalOverrides() error {
	engines := map[Engine]struct{}{}
	for _, pkg := range c.SQL {
		if _, ok := engines[pkg.Engine]; !ok {
			engines[pkg.Engine] = struct{}{}
		}
	}
	if c.Overrides.Go == nil {
		return nil
	}
	usesMultipleEngines := len(engines) > 1
	for _, oride := range c.Overrides.Go.Overrides {
		if usesMultipleEngines && oride.Engine == "" {
			return fmt.Errorf(`the "engine" field is required for global type overrides because your configuration uses multiple database engines`)
		}
	}
	return nil
}
