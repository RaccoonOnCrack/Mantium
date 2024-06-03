// Package util implements utility functions
package util

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src/config"
)

var logger *zerolog.Logger

// GetLogger returns the zerolog logger instance
func GetLogger(logLevel zerolog.Level) *zerolog.Logger {
	if logger == nil {
		l := zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
		logger = &l
	}

	return logger
}

// AddErrorContext adds context to an error, like:
// "Error downloading image: Get "https://example.com/image.jpg": dial tcp: lookup example.com: no such host".
// Should be used in functions that can return multiple errors without a spefic origin/context.
func AddErrorContext(context string, err error) error {
	return fmt.Errorf("%s: %w", context, err)
}

// ErrorContains checks if an error contains a specific string
func ErrorContains(err error, s string) bool {
	return strings.Contains(err.Error(), s)
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

// GetImageFromURL downloads an image from a URL and tries to resize it.
// If the image is not resized, it returns the original image.
func GetImageFromURL(url string, retries int, retryInterval time.Duration) (imgBytes []byte, resized bool, err error) {
	contextError := "Error downloading image '%s'"

	imageBytes := make([]byte, 0)
	for i := 0; i < retries; i++ {
		resp, err := http.Get(url)
		if err != nil {
			if i == retries-1 {
				return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), err)
			}
			time.Sleep(retryInterval)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if i == retries-1 {
				return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), fmt.Errorf("Status code is not OK, instead it's %d", resp.StatusCode))
			}
			time.Sleep(retryInterval)
			continue
		}

		imageBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			if i == retries-1 {
				return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), AddErrorContext("Could not read the image data from request body", err))
			}
			time.Sleep(retryInterval)
			continue
		}
	}

	img, err := ResizeImage(imageBytes, uint(DefaultImageWidth), uint(DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// and other libraries use the standard library under the hood
		if ErrorContains(err, "unsupported JPEG feature: luma/chroma subsampling ratio") {
			img = imageBytes
		} else {
			return nil, resized, AddErrorContext(fmt.Sprintf(contextError, url), err)
		}
	} else {
		resized = true
	}

	return img, resized, nil
}

// ResizeImage resizes an image to the specified width and height
func ResizeImage(imgBytes []byte, width, height uint) ([]byte, error) {
	contextError := "Error resizing image to width %d and height %d"

	_, format, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), err)
	}

	resizedImg := resize.Resize(width, height, img, resize.Lanczos3)

	var resizedBuf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&resizedBuf, resizedImg, nil)
	case "png":
		err = png.Encode(&resizedBuf, resizedImg)
	default:
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), fmt.Errorf("Unsupported image format to resize: %s", format))
	}
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextError, width, height), err)
	}

	return resizedBuf.Bytes(), nil
}

// IsImageValid checks if an image is valid by decoding it
func IsImageValid(imgBytes []byte) bool {
	_, _, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	return err == nil
}

// GetRFC3339Datetime returns a time.Time from a RFC3339 formatted string.
// Also truncate the time to seconds.
func GetRFC3339Datetime(date string) (time.Time, error) {
	contextError := "Error parsing RFC3339 datetime '%s'"

	parsedDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return time.Time{}, AddErrorContext(fmt.Sprintf(contextError, date), err)
	}
	parsedDate = parsedDate.In(time.Local).Truncate(time.Second)

	return parsedDate, nil
}

// RequestUpdateMangasMetadata sends a request to the server to update all mangas metadata
func RequestUpdateMangasMetadata(notify bool) (*http.Response, error) {
	contextErrror := "Error requesting to update mangas metadata (notify is %v)"

	client := &http.Client{}

	apiPort := config.GlobalConfigs.API.Port
	if apiPort == "" {
		apiPort = "8080"
	}

	url := fmt.Sprintf("http://localhost:%s/v1/mangas/metadata", apiPort)
	if notify {
		url += "?notify=true"
	}
	req, err := http.NewRequest("PATCH", url, nil)
	if err != nil {
		return nil, AddErrorContext(fmt.Sprintf(contextErrror, notify), err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, AddErrorContext(fmt.Sprintf(contextErrror, notify), err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp, AddErrorContext(fmt.Sprintf(contextErrror, notify), fmt.Errorf("Status code is not OK, instead it's %d", resp.StatusCode))
	}

	return resp, nil
}
