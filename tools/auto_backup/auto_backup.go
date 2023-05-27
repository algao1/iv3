package auto_backup

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/algao1/iv3/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
)

type S3Backuper struct {
	client *s3.S3
	token  string
	url    string
	logger *zap.Logger
}

func NewS3Backuper(token, url string, cfg config.SpacesConfig,
	logger *zap.Logger) (*S3Backuper, error) {
	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(cfg.Key, cfg.Secret, ""),
		Endpoint:    aws.String("https://nyc3.digitaloceanspaces.com"),
		Region:      aws.String("us-east-1"),
		// Configures to use subdomain/virtual calling format.
		// Depending on your version, alternatively use o.UsePathStyle = false
		S3ForcePathStyle: aws.Bool(false),
	}
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		logger.Fatal("unable to create new s3 session", zap.Error(err))
	}
	s3Client := s3.New(newSession)

	b := &S3Backuper{
		client: s3Client,
		token:  token,
		url:    url,
		logger: logger,
	}

	return b, nil
}

func (b *S3Backuper) Start() error {
	s := gocron.NewScheduler(time.UTC)
	_, err := s.Every(12).Hours().Do(func() {
		err := b.backupAndUpload()
		if err != nil {
			b.logger.Error("unable to backup and upload to s3", zap.Error(err))
		}
		b.logger.Info("backup and upload to s3 successful")
	})
	if err != nil {
		return fmt.Errorf("unable to schedule backup: %w", err)
	}

	s.StartAsync()
	s.RunAll()
	return nil
}

func (b *S3Backuper) backupAndUpload() error {
	dateStr := time.Now().Format(time.RFC3339)

	backUpFilePath := ".data/auto_backup/" + dateStr
	backupCmd := exec.Command(
		"influx",
		"backup", backUpFilePath,
		"-t", b.token,
		"--host", b.url,
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
	_, err = b.client.PutObject(&object)
	if err != nil {
		return err
	}

	// Maybe delete the file after uploading?
	err = os.RemoveAll(backUpFilePath)
	if err != nil {
		b.logger.Warn("unable to clean up backup file", zap.Error(err))
	}

	err = os.Remove(tarFilePath)
	if err != nil {
		b.logger.Warn("unable to clean up tar file", zap.Error(err))
	}

	return nil
}
