package utils

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func LoadYamlOrJSONConfigToViper(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	filename := filepath.Base(path)
	configName := strings.TrimSuffix(filename, ext)
	configPath := filepath.Dir(path)

	viper.SetConfigName(configName)
	viper.AddConfigPath(configPath)

	switch ext {
	case ".yaml", ".yml":
		viper.SetConfigType("yaml")
	case ".json":
		viper.SetConfigType("json")
	default:
		return fmt.Errorf("unsupported config file extension: %s", ext)
	}

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Error(err)
		return fmt.Errorf("failed to read config from %s: %w", path, err)
	}

	log.Printf("Loaded config from %s", path)
	return nil
}
