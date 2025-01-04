package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const (
	maxUploadSize = 1 << 30 // 1 GB
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	// Extract videoID from URL
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
	vid, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Tried to upload video file for not existing video ID", err)
		return
	}
	if vid.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("User %s is not the owner of the video %s. Owner is: %s", &userID, &videoID, vid.UserID), err)
		return
	}
	// Parse video file from form data
	file, _, err := r.FormFile("video")
	if err != nil {
		http.Error(w, "Failed to parse file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate MIME type
	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	file.Seek(0, io.SeekStart)

	mimeType := http.DetectContentType(buffer)
	if mimeType != "video/mp4" {
		http.Error(w, "Invalid file type. Only MP4 videos are allowed", http.StatusBadRequest)
		return
	}

	// Save file to a temporary file on disk
	tempFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Failed to write to temporary file", http.StatusInternalServerError)
		return
	}
	tempFile.Seek(0, io.SeekStart)

	// Generate a random key for S3
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		http.Error(w, "Failed to generate random key", http.StatusInternalServerError)
		return
	}
	fileKey := fmt.Sprintf("%s.mp4", hex.EncodeToString(randomBytes))
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        tempFile,
		ContentType: &mimeType,
	})
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileKey)
	vid.VideoURL = &s3URL
	vid.UpdatedAt = time.Now()
	cfg.db.UpdateVideo(vid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Video uploaded successfully"))
}
