package phraseapp

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

type Config struct {
	*Credentials

	ProjectID  string
	Page       *int
	PerPage    *int

	FileFormat string

	Defaults   map[string]map[string]interface{}

	Targets    []byte
	Sources    []byte
}

const configName = ".phraseapp.yml"

func ReadConfig() (*Config, error) {
	cfg := new(Config)
	cfg.Credentials = new(Credentials)
	rawCfg := struct{ PhraseApp *Config }{PhraseApp: cfg}

	content, err := configContent()
	switch {
	case err != nil:
		return nil, err
	case content == nil:
		return cfg, nil
	default:
		return cfg, yaml.Unmarshal(content, rawCfg)
	}
}

func configContent() ([]byte, error) {
	path, err := configPath()
	switch {
	case err != nil:
		return nil, err
	case path == "":
		return nil, nil
	default:
		return ioutil.ReadFile(path)
	}
}

func configPath() (string, error) {
	if envConfig := os.Getenv("PHRASEAPP_CONFIG"); envConfig != "" {
		possiblePath := path.Join(envConfig)
		switch _, err := os.Stat(possiblePath); {
		case err == nil:
			return possiblePath, nil
		case os.IsNotExist(err):
			return "", fmt.Errorf("file %q (given in PHRASEAPP_CONFIG) doesn't exist", possiblePath)
		default:
			return "", err
		}
	}

	callerPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	possiblePath := path.Join(callerPath, configName)
	if _, err := os.Stat(possiblePath); err == nil {
		return possiblePath, nil
	}

	possiblePath = defaultConfigDir()
	if _, err := os.Stat(possiblePath); err != nil {
		return "", nil
	}

	return possiblePath, nil
}

func (cfg *Config) UnmarshalYAML(unmarshal func(i interface{}) error) error {
	if cfg.Credentials == nil {
		cfg.Credentials = new(Credentials)
	}

	m := map[string]interface{}{}
	err := ParseYAMLToMap(unmarshal, map[string]interface{}{
		"access_token": &cfg.Credentials.Token,
		"host":         &cfg.Credentials.Host,
		"debug":        &cfg.Credentials.Debug,
		"page":         &cfg.Page,
		"perpage":      &cfg.PerPage,
		"project_id":   &cfg.ProjectID,
		"file_format":  &cfg.FileFormat,
		"push":         &cfg.Sources,
		"pull":         &cfg.Targets,
		"defaults":     &m,
	})
	if err != nil {
		return err
	}

	cfg.Defaults = map[string]map[string]interface{}{}
	for path, rawConfig := range m {
		cfg.Defaults[path], err = ValidateIsRawMap("defaults."+path, rawConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

const cfgValueErrStr = "configuration key %q has invalid value: %T\nsee https://phraseapp.com/docs/developers/cli/configuration/"
const cfgKeyErrStr = "configuration key %q has invalid type: %T\nsee https://phraseapp.com/docs/developers/cli/configuration/"
const cfgInvalidKeyErrStr = "configuration key %q unknown\nsee https://phraseapp.com/docs/developers/cli/configuration/"

func ValidateIsString(k string, v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf(cfgValueErrStr, k, v)
	}
	return s, nil
}

func ValidateIsBool(k string, v interface{}) (bool, error) {
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf(cfgValueErrStr, k, v)
	}
	return b, nil
}

func ValidateIsInt(k string, v interface{}) (int, error) {
	i, ok := v.(int)
	if !ok {
		return 0, fmt.Errorf(cfgValueErrStr, k, v)
	}
	return i, nil
}

func ValidateIsRawMap(k string, v interface{}) (map[string]interface{}, error) {
	raw, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf(cfgValueErrStr, k, v)
	}

	ps := map[string]interface{}{}
	for mk, mv := range raw {
		s, ok := mk.(string)
		if !ok {
			return nil, fmt.Errorf(cfgKeyErrStr, fmt.Sprintf("%s.%v", k, mk), mk)
		}
		ps[s] = mv
	}
	return ps, nil
}

func ConvertToStringMap(raw map[string]interface{}) (map[string]string, error) {
	ps := map[string]string{}
	for mk, mv := range raw {
		switch v := mv.(type) {
		case string:
			ps[mk] = v
		case bool:
			ps[mk] = fmt.Sprintf("%t", v)
		case int:
			ps[mk] = fmt.Sprintf("%d", v)
		default:
			return nil, fmt.Errorf("invalid type of key %q: %T", mk, mv)
		}
	}
	return ps, nil
}

// Calls the YAML parser function (see yaml.v2/Unmarshaler interface) with a map
// of string to interface. This map is then iterated to match against the given
// map of keys to fields, validates the type and sets the fields accordingly.
func ParseYAMLToMap(unmarshal func(interface{}) error, keysToField map[string]interface{}) error {
	m := map[string]interface{}{}
	if err := unmarshal(m); err != nil {
		return err
	}

	var err error
	for k, v := range m {
		value, found := keysToField[k]
		if !found {
			return fmt.Errorf(cfgInvalidKeyErrStr, k)
		}

		switch val := value.(type) {
		case *string:
			*val, err = ValidateIsString(k, v)
		case *int:
			*val, err = ValidateIsInt(k, v)
		case **int:
			*val = new(int)
			**val, err = ValidateIsInt(k, v)
		case *bool:
			*val, err = ValidateIsBool(k, v)
		case *map[string]interface{}:
			*val, err = ValidateIsRawMap(k, v)
		case *[]byte:
			*val, err = yaml.Marshal(v)
		default:
			err = fmt.Errorf(cfgValueErrStr, k, value)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
