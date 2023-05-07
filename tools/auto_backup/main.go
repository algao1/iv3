package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Copy pasted previous config file.
type Config struct {
	Spaces SpacesConfig `yaml:"spaces"`
}

type SpacesConfig struct {
	Key    string `yaml:"key"`
	Secret string `yaml:"secret"`
}

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "config file")
	flag.Parse()
}

func main() {
	logger, _ := zap.NewDevelopment()

	file, err := os.ReadFile(configFile)
	if err != nil {
		logger.Fatal("unable to read config file", zap.Error(err))
	}

	cfg := Config{}
	if err = yaml.Unmarshal(file, &cfg); err != nil {
		logger.Fatal("unable to unmarshal config file", zap.Error(err))
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.Spaces.Key, cfg.Spaces.Secret, ""),
		Endpoint:         aws.String("https://nyc3.digitaloceanspaces.com"),
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(false), // // Configures to use subdomain/virtual calling format. Depending on your version, alternatively use o.UsePathStyle = false
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		logger.Fatal("unable to create new s3 session", zap.Error(err))
	}
	s3Client := s3.New(newSession)

	token := os.Getenv("INFLUXDB_TOKEN")
	url := os.Getenv("INFLUXDB_URL")

	s := gocron.NewScheduler(time.UTC)
	_, err = s.Every(24).Hours().Do(func() {
		err := backupAndUpload(s3Client, token, url)
		if err != nil {
			logger.Error("unable to backup and upload to s3", zap.Error(err))
		}
	})
	if err != nil {
		logger.Fatal("unable to create new scheduler", zap.Error(err))
	}

	s.StartBlocking()
}

func backupAndUpload(s3Client *s3.S3, influxdbToken, influxdbUrl string) error {
	dateStr := time.Now().Format(time.RFC3339)

	backUpFilePath := ".data/auto_backup/" + dateStr
	backupCmd := exec.Command(
		"influx",
		"backup", backUpFilePath,
		"-t", influxdbToken,
		"--host", influxdbUrl,
	)
	out, err := backupCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("backup cmd failed with %s: %w", out, err)
	}

	tarFilePath := ".data/auto_backup/archive_" + dateStr + ".tar.gz"
	compressCmd := exec.Command("tar", "-zcvf", tarFilePath, backUpFilePath)
	out, err = compressCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compress cmd failed with %s: %w", out, err)
	}

	file, err := os.Open(tarFilePath)
	if err != nil {
		return err
	}

	awsFilePath := "auto_backup/archive_" + dateStr + ".tar.gz"
	object := s3.PutObjectInput{
		Bucket: aws.String("iv3"),
		Key:    &awsFilePath,
		Body:   file,
	}
	_, err = s3Client.PutObject(&object)
	if err != nil {
		return err
	}

	// Maybe delete the file after uploading?
	err = os.RemoveAll(backUpFilePath)
	if err != nil {
		return fmt.Errorf("unable to clean up backup file: %w", err)
	}

	err = os.Remove(tarFilePath)
	if err != nil {
		return fmt.Errorf("unable to clean up tar file: %w", err)
	}

	return nil
}
