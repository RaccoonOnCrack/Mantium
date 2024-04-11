// Package util implements utility functions
package util

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/AnthonyHewins/gotfy"
	"github.com/nfnt/resize"
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/config"
)

var logger *zerolog.Logger

// GetLogger returns the zerolog logger instance
func GetLogger() *zerolog.Logger {
	logLevel := config.GlobalConfigs.LogLevel
	if logger == nil {
		l := zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
		logger = &l
	}

	return logger
}

// AddErrorContext adds context to an error, like:
// "error downloading image: Get "https://example.com/image.jpg": dial tcp: lookup example.com: no such host".
// Should be used to add context to errors that are
// returned to the user, mostly in exported functions
// and methods
func AddErrorContext(err error, context string) error {
	return fmt.Errorf("%s: %w", context, err)
}

// RemoveLastOccurrence removes the last occurrence of a string from another string
func RemoveLastOccurrence(s, old string) string {
	if old == "" {
		return s
	}

	lastIndex := strings.LastIndex(s, old)
	modifiedString := s
	if lastIndex != -1 {
		modifiedString = s[:lastIndex] + s[lastIndex+len(old):]
	}

	return modifiedString
}

var (
	// DefaultImageHeight is the default height of an image
	DefaultImageHeight = 355
	// DefaultImageWidth is the default width of an image
	DefaultImageWidth = 250
)

// GetImageFromURL downloads an image from a URL and returns the image bytes
func GetImageFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		err = fmt.Errorf("error downloading image: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to download image. Status code: %d", resp.StatusCode)
		return nil, err
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf(`error reading image data at url "%s": %s`, url, err)
		return nil, err
	}

	img, err := ResizeImage(imageBytes, uint(DefaultImageWidth), uint(DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// And other libraries use the standard library under the hood
		if err.Error() == "unsupported JPEG feature: luma/chroma subsampling ratio" {
			img = imageBytes
		} else {
			err = fmt.Errorf("error resizing image: %s", err)
			return nil, err
		}
	}

	return img, nil
}

// ResizeImage resizes an image to the specified width and height
func ResizeImage(imgBytes []byte, width, height uint) ([]byte, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}

	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)

	var resizedBuf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&resizedBuf, resizedImg, nil)
	case "png":
		err = png.Encode(&resizedBuf, resizedImg)
	default:
		return nil, fmt.Errorf("unsupported image format to resize: %s", format)
	}
	if err != nil {
		return nil, err
	}

	return resizedBuf.Bytes(), nil
}

// GetNtfyPublisher returns a new NtfyPublisher
func GetNtfyPublisher() (*NtfyPublisher, error) {
	configs := config.GlobalConfigs.Ntfy

	server, err := url.Parse(configs.Address)
	if err != nil {
		return nil, err
	}

	customClient := &http.Client{
		Transport: &customNtfyTransport{
			ntfyToken: configs.Token,
		},
	}
	publisher, err := gotfy.NewPublisher(server, customClient)
	if err != nil {
		return nil, err
	}

	return &NtfyPublisher{
		Publisher: publisher,
		Topic:     configs.Topic,
		Token:     configs.Token,
	}, nil
}

type customNtfyTransport struct {
	ntfyToken string
}

func (t *customNtfyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.ntfyToken))

	return http.DefaultTransport.RoundTrip(req)
}

// NtfyPublisher is a wrapper around gotfy.Publisher
type NtfyPublisher struct {
	Publisher *gotfy.Publisher
	Topic     string
	Token     string
}

// SendMessage sends a message to the Ntfy server
func (t *NtfyPublisher) SendMessage(ctx context.Context, message *gotfy.Message) error {
	_, err := t.Publisher.SendMessage(ctx, message)

	return err
}

// GetRFC3339Datetime returns a time.Time from a RFC3339 formatted string
func GetRFC3339Datetime(date string) (time.Time, error) {
	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return time.Time{}, err
	}
	parsedDate = parsedDate.In(time.UTC)

	return parsedDate, err
}

// RequestUpdateMangasMetadata sends a request to the server to update all mangas metadata
func RequestUpdateMangasMetadata(notify bool) (*http.Response, error) {
	client := &http.Client{}

	url := "http://localhost:8080/v1/mangas/metadata"
	if notify {
		url += "?notify=true"
	}
	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("failed to update mangas metadata. Status code: %d", resp.StatusCode)
	}

	return resp, nil
}
