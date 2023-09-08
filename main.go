package main

import (
	"code.gitea.io/sdk/gitea"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

const (
	configFileName = "backupUtil.cfg"
)

type Config struct {
	GiteaURL     string `yaml:"GiteaURL"`
	GiteaToken   string `yaml:"GiteaToken"`
	RepoOwner    string `yaml:"RepoOwner"`
	RepoName     string `yaml:"RepoName"`
	BackupFolder string `yaml:"BackupFolder"`
	GitLogFile   string `yaml:"GitLogFile"`
	LogLevel     string `yaml:"LogLevel"`
}

func (cfg Config) valid() bool {
	return cfg.GiteaURL != "" ||
		cfg.GiteaToken != "" ||
		cfg.RepoOwner != "" ||
		cfg.RepoName != "" ||
		cfg.BackupFolder != "" ||
		cfg.GitLogFile != ""
}

func hash(content []byte) string {
	// Create a new SHA-1 hash
	sha := sha1.New()

	// Write the content to the hash object
	_, err := sha.Write(content)
	if err != nil {
		log.Fatalf("Не удалось вычислить SHA-1 hash: %v\n", err)
	}

	// Get the hash sum as a byte slice
	hashBytes := sha.Sum(nil)

	// Convert the byte slice to a hexadecimal string
	hashString := fmt.Sprintf("%x", hashBytes)

	return hashString
}

func getConfigPath(fileName string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)

	return filepath.Join(exeDir, fileName), nil
}

func readConfig(filePath string) (cfg Config, err error) {
	// Чтение файла конфигурации
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, err
	}

	// Разбор YAML и загрузка данных в структуру
	if err = yaml.Unmarshal(configFile, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func processGit(cfg Config) {
	log.Infof("Производится поиск файлов конфигурации в дирректории %s.\n", cfg.BackupFolder)

	var localFiles []string
	entries, err := os.ReadDir(cfg.BackupFolder)
	if err != nil {
		log.Fatalf("В указанной %s дирректории нет файлов.\n", cfg.BackupFolder)
	}
	for _, e := range entries {
		if e.Name() != cfg.GitLogFile {
			localFiles = append(localFiles, e.Name())
		}
	}

	log.Infof("Инициализируется клиент Gitea.\n")

	client, err := gitea.NewClient(cfg.GiteaURL, gitea.SetToken(cfg.GiteaToken))
	if err != nil {
		log.Fatalf("Не удалось подключиться к Gitea: %v\n", err)
	}

	log.Infof("Запрашивается список файлов из master-ветки репозитория %s.\n", cfg.RepoName)

	masterFiles, _, err := client.ListContents(cfg.RepoOwner, cfg.RepoName, "", "")
	if err != nil {
		log.Fatalf("Не удалось получить файлы из master: %v\n", err)
	}

	log.Infof("Получение локального commit сообщения из файла %s.\n", cfg.GitLogFile)

	commitMsg, err := os.ReadFile(filepath.Join(cfg.BackupFolder, cfg.GitLogFile))
	if err != nil {
		log.Fatalf("Ошибка чтения файла %s: %v\n", cfg.GitLogFile, err)
	}

	log.Infof("Производится поиск и передача локальных изменений.\n")

	for _, configName := range localFiles {
		localContent, err := os.ReadFile(filepath.Join(cfg.BackupFolder, configName))
		if err != nil {
			log.Fatalf("Не удалось прочитать файл с конфигурацией: %v\n", err)
		}
		encodedContent := base64.StdEncoding.EncodeToString(localContent)

		isFileExists := false
		for _, file := range masterFiles {
			if file.Name == configName {
				isFileExists = true

				remoteContent, _, err := client.GetFile(cfg.RepoOwner, cfg.RepoName, "", file.Path)
				if err != nil {
					log.Fatalf("Не удолось загрузить файл %s с удаленного репозитория.\n", file.Name)
				}
				if hash(localContent) != hash(remoteContent) {
					_, _, err := client.UpdateFile(cfg.RepoOwner, cfg.RepoName, configName, gitea.UpdateFileOptions{
						FileOptions: gitea.FileOptions{
							Message:    string(commitMsg),
							BranchName: "master",
						},
						Content: encodedContent,
						SHA:     file.SHA,
					})
					if err != nil {
						log.Fatalf("Не удалось добавить конфигурацию в master: %v\n", err)
					}
				} else {
					log.Warnf("Локальных изменений в файле %s не обнаружено.\n", configName)
				}
				break
			}
		}
		if !isFileExists {
			_, _, err := client.CreateFile(cfg.RepoOwner, cfg.RepoName, configName, gitea.CreateFileOptions{
				FileOptions: gitea.FileOptions{
					Message:    string(commitMsg),
					BranchName: "master",
				},
				Content: encodedContent,
			})
			if err != nil {
				log.Fatalf("Не удалось добавить конфигурацию в master: %v\n", err)
			}
		}
	}
	log.Infof("Процедура выполнена успешно.\n")
}

func processConfig(fileName string) Config {
	log.Infof("Определяется путь до конфигурационного файла %s.\n", fileName)

	cfgPath, err := getConfigPath(fileName)
	if err != nil {
		log.Fatalf("Не удалось определить путь до файла конфигураци %s.\n", fileName)
	}

	log.Infof("Загружается конфигурационный файл %s.\n", fileName)

	cfg, err := readConfig(cfgPath)
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию из файла %s.\n", fileName)
	}

	if !cfg.valid() {
		log.Fatalf("Проверьте файл конфигурации %s на наличие пустых полей.\n", fileName)
	}

	if cfg.LogLevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	}

	return cfg
}

func main() {
	log.SetOutput(os.Stdout)

	cfg := processConfig(configFileName)
	processGit(cfg)
}
