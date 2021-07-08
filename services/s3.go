package services

import (
	"bytes"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pantyukhov/imageresizeserver/pkg/setting"
	"image"
	"io"
	"log"
	"strconv"
	"strings"
)

type S3Service struct {
	MinioClient *minio.Client
}

func NewS3Service() S3Service {
	endpoint := setting.Settings.S3Config.Endpoint
	accessKeyID := setting.Settings.S3Config.AccessKeyID
	secretAccessKey := setting.Settings.S3Config.SecretAccessKey
	useSSL := setting.Settings.S3Config.UseSSL
	regionName := setting.Settings.S3Config.RegionName

	minioClient, err := minio.New(
		endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
			Region: regionName,
		},
	)
	if err != nil {
		log.Fatalln(err)
	}
	return S3Service{
		MinioClient: minioClient,
	}
}

// If either width or height is set to 0, it will be set to an aspect ratio preserving value.
func (s *S3Service) GetResizeSettings(filepath string) (uint, uint, string) {
	items := strings.Split(filepath, "/")

	if len(items) < 1 {
		// if can't split filepath, return 0x0
		return uint(0), uint(0), filepath
	}

	sizes := strings.SplitN(strings.ToLower(items[len(items)-2]), "x", 2)

	// len(sizes) always more or equal 1
	height, err := strconv.Atoi(sizes[0])
	if err != nil {
		height = 0
	}

	var width = 0

	if len(sizes) > 1 {
		// only if split on 2 or more parts
		width, err = strconv.Atoi(sizes[1])
		if err != nil {
			width = 0
		}
	}

	path := strings.Join(items[:len(items)-2], "/") + "/" + items[len(items)-1]

	return uint(width), uint(height), path
}

func (s *S3Service) ResizeImage(localPath string, height uint, width uint) (image.Image, error) {
	img, err := imaging.Open(localPath, imaging.AutoOrientation(true))

	if err != nil {
		return nil, err
	}
	m := imaging.Resize(img, int(width), int(height), imaging.Lanczos)
	return m, err
}

func (s *S3Service) ResizeBytesImage(file io.Reader, height uint, width uint) (image.Image, error) {
	img, err := imaging.Decode(file, imaging.AutoOrientation(true))

	if err != nil {
		return nil, err
	}

	newImg := imaging.Fill(img, int(width), int(height), imaging.Center, imaging.Lanczos)

	return newImg, err
}

func (s *S3Service) ResizeFilePath(bucket, filepath string) error {
	height, width, path := s.GetResizeSettings(filepath)

	file, err := s.MinioClient.GetObject(
		setting.Settings.Context.Context,
		bucket,
		path,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		return err
	}

	newImg, err := s.ResizeBytesImage(file, height, width)

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	f, _ := imaging.FormatFromFilename(filepath)

	if err := imaging.Encode(&buf, newImg, f); err != nil {
		return err
	}

	_, err = s.MinioClient.PutObject(
		setting.Settings.Context.Context,
		bucket,
		filepath,
		&buf,
		int64(buf.Len()),
		minio.PutObjectOptions{
			ContentType: info.ContentType,
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func IsBucketAllowed(bucket string) bool {
	if bucket == setting.Settings.S3Config.Bucket {
		return true
	}
	for _, val := range setting.Settings.S3Config.Buckets {
		if val == bucket {
			return true
		}
	}

	return false
}

func (s *S3Service) SeparateBucket(filepath string) (bucket string, path string) {
	items := strings.SplitN(filepath, "/", 2)

	if len(items) == 2 {
		bucket = items[0]
		path = items[1]

		if !IsBucketAllowed(bucket) {
			bucket = ""
			path = filepath
		}
	} else {
		bucket = ""
		path = filepath
	}

	return bucket, path
}

func (s *S3Service) GetOrCreteFile(filepath string, allowResize bool) (*minio.Object, *minio.ObjectInfo, error) {

	// filepath = {bucket}/{path}
	filepath = strings.TrimLeft(filepath, "/")

	bucket, path := s.SeparateBucket(filepath)
	path = strings.TrimLeft(path, "/")

	if len(bucket) == 0 {
		// use default bucket if can't separate from filepath
		bucket = setting.Settings.S3Config.Bucket
	}

	file, err := s.MinioClient.GetObject(
		setting.Settings.Context.Context,
		bucket,
		path,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()

	if info.Key == "" && allowResize {
		err := s.ResizeFilePath(bucket, path)
		if err != nil {
			return nil, nil, err
		}
		return s.GetOrCreteFile(filepath, false)
	}
	return file, &info, err
}
