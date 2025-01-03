package main

import (
	"fmt"
	"net/http"

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

	// Decode the data URL
	mediaType, data, err := DecodeDataURL(*vd.ThumbnailURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding thumbnail data URL", err)
		return
	}

	// Set response headers and write the binary data
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

	_, err = w.Write(data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error writing response", err)
		return
	}
}
