package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil || *video.VideoURL == "" {
		return video, fmt.Errorf("video url is nil or empty")
	}
	videoURLSplit := strings.SplitN(*video.VideoURL, ",", 2)
	if len(videoURLSplit) != 2 {
		return database.Video{}, fmt.Errorf("invalid video url format")
	}
	bucket := videoURLSplit[0]
	key := videoURLSplit[1]

	url, err := generatePresignedURL(cfg.s3Client, bucket, key, 5*time.Minute)
	if err != nil {
		return video, err
	}

	video.VideoURL = &url

	return video, nil
}
