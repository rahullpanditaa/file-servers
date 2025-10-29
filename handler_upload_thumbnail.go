package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		return
	}

	data, headers, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not retreive file for given form key", err)
		return
	}
	defer data.Close()

	mediaType := headers.Header.Get("Content-Type")

	imgData, err := io.ReadAll(data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not read image data", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "authenticated user not the video owner", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "unable to retreive video", err)
		return
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "invalid user", nil)
		return
	}

	// convert image data ([]byte) to a base64 string
	imgDataStr := base64.StdEncoding.EncodeToString(imgData)

	// create a dataurl
	imgDataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, imgDataStr)

	// safer to mutate the fetched video from db
	// then update (avoid accidentally zeroing fields not included)
	video.ThumbnailURL = &imgDataURL
	video.UpdatedAt = time.Now().UTC()

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
