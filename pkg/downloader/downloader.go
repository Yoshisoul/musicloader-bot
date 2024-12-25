package downloader

import (
	"fmt"
	"os"
	"time"

	"github.com/kkdai/youtube/v2"
)

const videoDuration = time.Minute * 10

func DownloadMp3(url string, itag int) (*os.File, error) {
	client := youtube.Client{}

	video, err := client.GetVideo(url)
	if err != nil {
		return nil, err
	}

	if video.Duration > videoDuration {
		err := fmt.Errorf("video is too long: %s", video.Duration)
		return nil, err
	}

	formatList140 := video.Formats.Itag(itag) // 140 - audio MP3 128kbps, 141 - audio MP3 256kbps
	if formatList140 == nil {
		err := fmt.Errorf("can't find mp3 audio format for this video")
		return nil, err
	}

	videoStream, _, err := client.GetStream(video, &formatList140[0])
	if err != nil {
		return nil, err
	}

	defer videoStream.Close()

	videoName := video.Title
	videoFile, err := os.Create(videoName + ".mp3")
	if err != nil {
		return nil, err
	}
	defer videoFile.Close()

	_, err = videoFile.ReadFrom(videoStream)
	if err != nil {
		return nil, err
	}

	return videoFile, nil
}
