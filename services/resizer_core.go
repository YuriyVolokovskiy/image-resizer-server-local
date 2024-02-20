package services

import (
	"bufio"
	"bytes"
	"github.com/disintegration/imaging"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"github.com/pantyukhov/imageresizeserver/pkg/setting"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileService struct {
}

func NewFileService() FileService {
	return FileService{}
}

// GetResizeSettings If either width or height is set to 0, it will be set to an aspect ratio preserving value.
func (s *FileService) GetResizeSettings(filePath string) (uint, uint, string) {
	items := strings.Split(filePath, "/")

	if len(items) < 1 {
		// if can't split filePath, return 0x0
		return uint(0), uint(0), filePath
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

func (s *FileService) ResizeImage(localPath string, height uint, width uint) (image.Image, error) {
	img, err := imaging.Open(localPath, imaging.AutoOrientation(true))

	if err != nil {
		return nil, err
	}
	m := imaging.Resize(img, int(width), int(height), imaging.Lanczos)
	return m, err
}

func (s *FileService) resizeJpeg(file io.Reader, height uint, width uint) (bytes.Buffer, error) {
	img, err := imaging.Decode(file, imaging.AutoOrientation(true))

	var jpgBuf bytes.Buffer
	if err != nil {
		return jpgBuf, err
	}

	var newImg image.Image
	if height > 0 && width > 0 {
		newImg = imaging.Fill(img, int(width), int(height), imaging.Center, imaging.Lanczos)
	} else {
		newImg = imaging.Resize(img, int(width), int(height), imaging.Lanczos)
	}

	err = jpeg.Encode(bufio.NewWriter(&jpgBuf), newImg, &jpeg.Options{Quality: 65})

	return jpgBuf, err
}

func (s *FileService) ResizeBytesImage(file io.Reader, filePath string, height uint, width uint) (bytes.Buffer, error) {
	jpgBuf, err := s.resizeJpeg(file, height, width)

	if err != nil {
		return jpgBuf, err
	}

	ext := filepath.Ext(filePath)
	if ext == ".webp" {
		img, err := jpeg.Decode(bytes.NewReader(jpgBuf.Bytes()))
		if err != nil {
			return jpgBuf, err
		}

		var output bytes.Buffer
		options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 100)

		if err != nil {
			return output, err
		}
		if err := webp.Encode(bufio.NewWriter(&output), img, options); err != nil {
			return output, err
		}
		return output, nil
	}

	return jpgBuf, err

}

func (s *FileService) getOriginalPath(path string) string {

	var extension = filepath.Ext(path)
	if extension == ".webp" {
		return path[0 : len(path)-len(extension)]
	}

	return path
}

func (s *FileService) ResizeFilePath(filePath string) error {
	height, width, path := s.GetResizeSettings(filePath)

	originalPath := s.getOriginalPath(path)

	file, err := os.Open(originalPath)
	if err != nil {
		return err
	}
	defer file.Close()

	buf, err := s.ResizeBytesImage(file, filePath, height, width)

	if err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	err = os.WriteFile(filePath, buf.Bytes(), 0644)

	if err != nil {
		return err
	}

	return nil
}

func (s *FileService) GetOrCreteFile(filePath string, allowResize bool) (*os.File, os.FileInfo, error) {
	filePath = strings.TrimLeft(filePath, "/")

	// Check if filePath already starts with the root directory
	if !strings.HasPrefix(filePath, setting.Settings.LocalFSConfig.RootDirectory) {
		filePath = setting.Settings.LocalFSConfig.RootDirectory + "/" + filePath
	}

	for {
		file, err := os.Open(filePath)
		if err != nil {
			if os.IsNotExist(err) && allowResize {
				err := s.ResizeFilePath(filePath)
				if err != nil {
					return nil, nil, err
				}
				allowResize = false
				continue
			}
			return nil, nil, err
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, nil, err
		}

		return file, info, nil
	}
}
