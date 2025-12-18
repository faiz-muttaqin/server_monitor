package utils

import (
	"bufio"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func LoadEnvConfigToViper(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines or comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Split key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // skip invalid lines
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Optionally remove surrounding quotes
		value = strings.Trim(value, `"'`)

		// Set to environment
		// os.Setenv(key, value)
		viper.SetDefault(key, value)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
