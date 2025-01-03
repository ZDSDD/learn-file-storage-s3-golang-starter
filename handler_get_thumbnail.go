package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerThumbnailGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	vd, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", nil)
		return
	}

	if vd.ThumbnailURL == nil {
		respondWithError(w, http.StatusNotFound, "Thumbnail not found", nil)
		return
	}

	// Extract the file path from the ThumbnailURL
	thumbnailPath := filepath.Join(cfg.assetsRoot, filepath.Base(*vd.ThumbnailURL))

	// Open the file from the local filesystem
	file, err := os.Open(thumbnailPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondWithError(w, http.StatusNotFound, "Thumbnail file not found", nil)
		} else {
			respondWithError(w, http.StatusInternalServerError, "Error opening thumbnail file", err)
		}
		return
	}
	defer file.Close()

	// Get the file's MIME type
	buffer := make([]byte, 512) // Read the first 512 bytes for MIME type detection
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		respondWithError(w, http.StatusInternalServerError, "Error reading thumbnail file", err)
		return
	}
	// Reset file pointer after reading
	file.Seek(0, io.SeekStart)
	mediaType := http.DetectContentType(buffer)
	fmt.Printf(">>> fileType: %s\n", mediaType)
	// Serve the file
	w.Header().Set("Content-Type", mediaType)
	fileInfo, err := os.Stat(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting file info", err)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	fmt.Printf("file size: %d\n", fileInfo.Size())
	_, err = io.Copy(w, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing thumbnail to response", err)
		return
	}
}
