package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("Uploading thumbnail for video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing multipart form", err)
		return
	}

	file, head, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading uploaded file", err)
		return
	}
	defer file.Close()

	contentType := head.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		respondWithError(w, http.StatusBadRequest, "Only image uploads are supported", nil)
		return
	}

	// Determine file extension from content type
	exts, _ := mime.ExtensionsByType(contentType)
	if len(exts) == 0 {
		respondWithError(w, http.StatusBadRequest, "Unsupported file type", nil)
		return
	}
	fileExt := exts[0]

	// Build file path
	filePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s%s", videoID, fileExt))

	// Create and save file to the filesystem
	outputFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating file on disk", err)
		return
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving file to disk", err)
		return
	}

	// Update video thumbnail URL
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s%s", cfg.port, videoID, fileExt)
	fmt.Printf(">>> saved as: %s", thumbnailURL)
	vd, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching video from database", err)
		return
	}

	if vd.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized to upload thumbnail for this video", nil)
		return
	}

	vd.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(vd)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video in database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vd)
}
