package configutil

import (
	"docupdatesexecutor/pkg/structures"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

func LoadConfig(path string) (*structures.Config, error) {
	config := &structures.Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func LoadDotenv(path string) (map[string]string, error) {
	envMap, err := godotenv.Read(path)
	if err != nil {
		return nil, err
	}
	return envMap, nil
}

func BuildDSN(config *structures.DatabaseConfig) string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		config.Host, config.User, config.Password, config.Dbname, config.Port, config.Sslmode, config.TimeZone,
	)
}

func SetConfigs(pathYaml string, pathDotenv string) (*structures.Config, error) {
	config, err := LoadConfig(pathYaml)
	if err != nil {
		return nil, err
	}
	envMap, err := LoadDotenv(pathDotenv)
	if err != nil {
		return nil, err
	}

	config.Database.User = envMap["DB_USER"]
	config.Database.Password = envMap["DB_PASSWORD"]
	config.Redis.Password = envMap["REDIS_PASSWORD"]
	config.Redis.User = envMap["REDIS_USER"]

	return config, nil
}
