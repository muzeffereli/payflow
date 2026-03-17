package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type UploadHandler struct {
	client    *minio.Client
	bucket    string
	publicURL string // externally reachable base URL, e.g. http://localhost:9000
}

func NewUploadHandler(client *minio.Client, bucket, publicURL string) *UploadHandler {
	return &UploadHandler{client: client, bucket: bucket, publicURL: publicURL}
}

func (h *UploadHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/products/upload-image", h.UploadImage)
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "field 'file' is required"})
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only image files are allowed"})
		return
	}

	const maxSize = 5 << 20
	if header.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file must not exceed 5 MB"})
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	objectName := fmt.Sprintf("products/%s%s", uuid.New().String(), ext)

	_, err = h.client.PutObject(
		c.Request.Context(),
		h.bucket,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{ContentType: ct},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: " + err.Error()})
		return
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(h.publicURL, "/"), h.bucket, objectName)
	c.JSON(http.StatusOK, gin.H{"url": url})
}
