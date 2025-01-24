package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kkdai/youtube/v2"
)

const videoDuration = time.Minute * 10

func DownloadMp3(ctx context.Context, url string, itag int, outputPath string) (*os.File, error) {
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

	formatList := video.Formats.Itag(itag)
	if formatList == nil {
		err := fmt.Errorf("can't find mp3 audio format for this video")
		return nil, err
	}

	videoStream, _, err := client.GetStreamContext(ctx, video, &formatList[0])
	if err != nil {
		return nil, err
	}
	defer videoStream.Close()

	select {
	case <-ctx.Done():
		log.Printf("Download canceled: %s\n", video.Title)
		return nil, nil

	default:
		log.Printf("Video stream downloaded: %s\n", video.Title)
		videoName := sanitizeFileName(video.Title)
		videoFilePath := filepath.Join(outputPath, videoName+".mp3")
		videoFile, err := os.Create(videoFilePath)
		if err != nil {
			return nil, err
		}
		defer videoFile.Close()
		log.Printf("File created: %s\n", videoFilePath)

		_, err = videoFile.ReadFrom(videoStream)
		if err != nil {
			log.Printf("Error reading from video stream: %v", err)
			return nil, err
		}
		log.Printf("File downloaded stream: %s\n", videoFilePath)

		return videoFile, nil
	}
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
