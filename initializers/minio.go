package initializers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioConfig struct {
	Endpoint         string
	AccessKey        string
	SecretKey        string
	Bucket           string
	UseSSL           bool
	MaxSize          int64
	FileTypes        []string
	Expiry           time.Duration
	ExternalEndpoint string
}

var MinioClient *minio.Client
var Conf MinioConfig

func InitMinio() error {
	Conf = MinioConfig{
		Endpoint:         os.Getenv("MINIO_ENDPOINT"),
		AccessKey:        os.Getenv("MINIO_ACCESS_KEY"),
		SecretKey:        os.Getenv("MINIO_SECRET_KEY"),
		Bucket:           os.Getenv("MINIO_BUCKET"),
		UseSSL:           parseBool(os.Getenv("MINIO_USE_SSL")),
		MaxSize:          parseInt64(os.Getenv("MAX_FILE_SIZE"), 10485760),
		FileTypes:        parseFileTypes(os.Getenv("ALLOWED_FILE_TYPES")),
		Expiry:           parseExpiry(os.Getenv("PRESIGNED_URL_EXPIRY")),
		ExternalEndpoint: os.Getenv("MINIO_EXTERNAL_ENDPOINT"),
	}
	client, err := minio.New(Conf.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(Conf.AccessKey, Conf.SecretKey, ""),
		Secure: Conf.UseSSL,
	})
	if err != nil {
		return err
	}
	MinioClient = client
	exists, errBucket := client.BucketExists(context.Background(), Conf.Bucket)
	if errBucket != nil {
		return errBucket
	}
	if !exists {
		errCreate := client.MakeBucket(context.Background(), Conf.Bucket, minio.MakeBucketOptions{})
		if errCreate != nil {
			return errCreate
		}
	}
	log.Println("Minio bucket ready:", Conf.Bucket)
	return nil
}

func parseBool(val string) bool {
	return strings.ToLower(val) == "true"
}

func parseInt64(val string, def int64) int64 {
	if val == "" {
		return def
	}
	v, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func parseFileTypes(val string) []string {
	if val == "" {
		return []string{"image/jpeg", "image/png", "application/pdf"}
	}
	return strings.Split(val, ",")
}

func parseExpiry(val string) time.Duration {
	if val == "" {
		return time.Hour
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return time.Hour
	}
	return time.Duration(v) * time.Second
}

func baseMIME(mime string) string {
	if mime == "" {
		return ""
	}
	parts := strings.Split(mime, ";")
	return strings.TrimSpace(parts[0])
}

func CheckFileAllowed(size int64, mime string) error {
	if size > Conf.MaxSize {
		return fmt.Errorf("file size exceeds the limit")
	}
	incoming := baseMIME(mime)
	allowed := false
	for _, t := range Conf.FileTypes {
		if baseMIME(t) == incoming {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("file type is not allowed")
	}
	return nil
}

func GenerateAttachmentURL(id, fileName string) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(fileName)))
	expiry := Conf.Expiry
	endpoint := Conf.ExternalEndpoint
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}
	externalClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(Conf.AccessKey, Conf.SecretKey, ""),
		Secure: Conf.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create external minio client: %v", err)
	}
	presignedURL, err := externalClient.PresignedGetObject(context.Background(), Conf.Bucket, id, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to create presigned url: %v", err)
	}
	return presignedURL.String(), nil
}

func sanitizeFilename(name string) string {
	return strings.ReplaceAll(name, "\"", "")
}
