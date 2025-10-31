package main

import (
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	videoURLSplit := strings.Split(*video.VideoURL, ",")
	bucket := videoURLSplit[0]
	key := videoURLSplit[1]

	url, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &url

	return video, nil
}
