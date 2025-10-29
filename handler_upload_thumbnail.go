package main

import (
	"database/sql"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// determine file extension
	headerValueReceived := headers.Header.Get("Content-Type")

	mediatype, _, err := mime.ParseMediaType(headerValueReceived)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to get media type from header", nil)
		return
	}
	if mediatype != "image/jpeg" && mediatype != "image/png" {
		respondWithError(w, http.StatusBadRequest, "media type not an image", nil)
		return
	}
	media := strings.Split(mediatype, "/")
	extension := media[1]

	// create unique file path
	filePtr, err := os.Create(filepath.Join(cfg.assetsRoot, video.ID.String()+"."+extension))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create file", err)
		return
	}
	defer filePtr.Close()

	_, err = io.Copy(filePtr, data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create file", err)
		return
	}

	// safer to mutate the fetched video from db
	// then update (avoid accidentally zeroing fields not included)
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, video.ID.String(), extension)
	video.ThumbnailURL = &thumbnailURL
	video.UpdatedAt = time.Now().UTC()

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
