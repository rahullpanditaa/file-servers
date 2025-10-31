package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "unable to find video metadata, no match found for video ID", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "unable to retreive video from db", err)
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

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner of video", nil)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	const uploadLimit = 1 << 30
	err = r.ParseMultipartForm(uploadLimit)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to parse request body", nil)
		return
	}

	file, headers, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "could not retreive file for given form key", nil)
		return
	}
	defer file.Close()

	headerValueReceived := headers.Header.Get("Content-Type")

	// get mime type
	mediatype, _, err := mime.ParseMediaType(headerValueReceived)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to get media type from header", nil)
		return
	}

	if mediatype != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "uploaded file not video/mp4", nil)
		return
	}

	m := strings.Split(mediatype, "/")
	extension := m[1]

	// save uploaded file to a temp file on disk
	tempFile, err := os.CreateTemp("", "tubely-upload")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to copy uploaded file to local file", err)
		return
	}
	tempFile.Seek(0, io.SeekStart)

	processedVideo, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to encode video with faststart", err)
		return
	}

	ratio, err := getVideoAspectRatio(processedVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to get aspect ratio of video", err)
		return
	}

	var prefix string
	switch ratio {
	case "16:9":
		prefix = "landscape/"
	case "9:16":
		prefix = "portrait/"
	case "other":
		prefix = "other/"
	}

	// fill a 32 bit slice with random bytes
	randomSlice := make([]byte, 32)
	rand.Read(randomSlice)

	// random string to use as file name
	key := prefix + "/" + base64.RawURLEncoding.EncodeToString(randomSlice) + "." + extension

	videoFile, _ := os.Open(processedVideo)
	defer videoFile.Close()

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        videoFile,
		ContentType: &mediatype,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to upload video to aws", err)
		return
	}

	cloudFrontURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, key)
	video.VideoURL = &cloudFrontURL
	video.UpdatedAt = time.Now().UTC()

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
