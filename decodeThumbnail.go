package main

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// DecodeDataURL takes a data URL string and returns the media type and decoded data.
func DecodeDataURL(dataURL string) (string, []byte, error) {
	// Ensure the data URL starts with "data:"
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("invalid data URL")
	}

	// Split the data URL into metadata and the actual data
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid data URL format")
	}

	// Extract the metadata (e.g., "data:image/png;base64")
	metadata := parts[0]
	data := parts[1]

	// Check if the metadata specifies Base64 encoding
	if !strings.Contains(metadata, ";base64") {
		return "", nil, fmt.Errorf("only base64-encoded data URLs are supported")
	}

	// Extract the media type (e.g., "image/png")
	mediaType := strings.TrimPrefix(metadata, "data:")
	mediaType = strings.SplitN(mediaType, ";", 2)[0]

	// Decode the Base64-encoded data
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", nil, fmt.Errorf("error decoding base64 data: %w", err)
	}

	return mediaType, decodedData, nil
}
