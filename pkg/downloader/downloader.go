package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kkdai/youtube/v2"
)

const videoDuration = time.Minute * 10

func DownloadMp3(url string, itag int, outputPath string) (*os.File, error) {
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		err := os.MkdirAll(outputPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	video, client, err := GetVideoInfo(url)
	if err != nil {
		return nil, err
	}

	formatList140 := video.Formats.Itag(itag)
	if formatList140 == nil {
		err := fmt.Errorf("can't find mp3 audio format for this video")
		return nil, err
	}

	videoStream, _, err := client.GetStream(video, &formatList140[0])
	if err != nil {
		return nil, err
	}

	defer videoStream.Close()

	videoName := sanitizeFileName(video.Title)
	videoFilePath := filepath.Join(outputPath, videoName+".mp3")
	videoFile, err := os.Create(videoFilePath)
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

func GetVideoInfo(url string) (*youtube.Video, *youtube.Client, error) {
	client := youtube.Client{}

	video, err := client.GetVideo(url)
	if err != nil {
		return nil, nil, err
	}

	if video.Duration > videoDuration {
		err := fmt.Errorf("video is too long: %s", video.Duration)
		return nil, nil, err
	}

	return video, &client, nil
}

func sanitizeFileName(fileName string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := re.ReplaceAllString(fileName, " ")
	return sanitized
}
