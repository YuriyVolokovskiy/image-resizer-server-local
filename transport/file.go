package transport

import (
	"github.com/gin-gonic/gin"
	"github.com/pantyukhov/imageresizeserver/services"
	"net/http"
	"os"
)

type FileHandler struct {
	FileService services.FileService
}

func NewFileHandler(fileService services.FileService) FileHandler {
	return FileHandler{
		FileService: fileService,
	}
}

// HandleFile Handle request to file from local storage If file not found, try select from url
func (f *FileHandler) HandleFile(ctx *gin.Context) {
	file, info, err := f.FileService.GetOrCreteFile(ctx.Request.URL.Path, true)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	if err != nil {
		ctx.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}
	extraHeaders := map[string]string{
		//"Content-Disposition": "attachment; filename=" + info.Name(),
	}

	contentType := "application/octet-stream"

	if info.Mode().IsRegular() {
		contentType = http.DetectContentType(nil)
	}

	ctx.DataFromReader(http.StatusOK, info.Size(), contentType, file, extraHeaders)
}
