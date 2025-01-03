package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
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
		respondWithError(w, http.StatusInternalServerError, "Error forming file", err)
		return
	}
	defer file.Close()

	// Check Content-Type header
	contentType := head.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "image/") {
		fmt.Println("File is an image:", contentType)
	} else if strings.HasPrefix(contentType, "video/") {
		fmt.Println("File is a video:", contentType)
	} else {
		respondWithError(w, http.StatusBadRequest, "Unsupported file type", nil)
		return
	}

	// Optional: Inspect file bytes for robust type detection
	imgData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error when reading data from the file", err)
		return
	}

	mimeType := http.DetectContentType(imgData)
	fmt.Println("Detected MIME type:", mimeType)

	// Validate MIME type (if required)
	if !strings.HasPrefix(mimeType, "image/") && !strings.HasPrefix(mimeType, "video/") {
		respondWithError(w, http.StatusBadRequest, "Unsupported file type", nil)
		return
	}
	vd, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching video from database", err)
		return
	}
	if vd.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized to upload thumbnail for this video", nil)
		return
	}
	//Store image in the sqllite db
	var encodedImgBlob = base64.StdEncoding.EncodeToString(imgData)
	var dataUrl = fmt.Sprintf("data:%s;base64,%s", mimeType, encodedImgBlob)

	vd.ThumbnailURL = &dataUrl

	err = cfg.db.UpdateVideo(vd)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video in database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vd)
}
