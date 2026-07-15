package image

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// cacheControl key'ler rastgele ve içerik hiç değişmez (yeni fotoğraf =
// yeni key), bu yüzden immutable + 1 yıl. Tarayıcı ve CDN sonsuza dek
// cache'leyebilir.
const cacheControl = "public, max-age=31536000, immutable"

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string // R2 public bucket URL'i veya özel domain
}

// R2Store Cloudflare R2'ye yazar. S3-uyumlu API kullanır.
type R2Store struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewR2Store(cfg R2Config) (*R2Store, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" ||
		cfg.SecretAccessKey == "" || cfg.Bucket == "" || cfg.PublicURL == "" {
		return nil, fmt.Errorf("R2 ayarları eksik (account_id, access_key, secret, bucket, public_url gerekli)")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("aws yapılandırması yüklenemedi: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &R2Store{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}, nil
}

func (s *R2Store) Put(ctx context.Context, key string, size Size, data []byte) error {
	if err := safeKey(key); err != nil {
		return err
	}
	if !size.Valid() {
		return fmt.Errorf("geçersiz boyut: %q", size)
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(objectPath(key, size)),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("image/jpeg"),
		CacheControl: aws.String(cacheControl),
	})
	if err != nil {
		return fmt.Errorf("r2'ye yazılamadı (%s/%s): %w", key, size, err)
	}
	return nil
}

// Delete key'in tüm boyutlarını siler. Olmayan obje hata değil — S3
// DeleteObject zaten idempotent.
func (s *R2Store) Delete(ctx context.Context, key string) error {
	if err := safeKey(key); err != nil {
		return err
	}

	for _, size := range AllSizes {
		_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(objectPath(key, size)),
		})
		if err != nil {
			return fmt.Errorf("r2'den silinemedi (%s/%s): %w", key, size, err)
		}
	}
	return nil
}

func (s *R2Store) URL(key string, size Size) string {
	return s.publicURL + "/" + objectPath(key, size)
}
