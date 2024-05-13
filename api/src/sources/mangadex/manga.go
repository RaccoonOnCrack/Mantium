package mangadex

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata returns the metadata of a manga given its URL
func (s *Source) GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	s.checkClient()

	errorContext := "Error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangadex.org"
	mangaReturn.URL = mangaURL

	mangadexMangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	mangaAPIURL := fmt.Sprintf("%s/manga/%s?includes[]=cover_art", baseAPIURL, mangadexMangaID)
	var mangaAPIResp getMangaAPIResponse
	_, err = s.client.Request(context.Background(), "GET", mangaAPIURL, nil, &mangaAPIResp)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	attributes := &mangaAPIResp.Data.Attributes

	mangaReturn.Name = attributes.Title["en"]
	if mangaReturn.Name == "" {
		mangaReturn.Name = attributes.Title["ja"]
		if mangaReturn.Name == "" {
			mangaReturn.Name = attributes.Title["ja-ro"]
			if mangaReturn.Name == "" {
				for _, title := range attributes.Title {
					mangaReturn.Name = title
					break
				}
			}
		}
	}

	lastUploadChapter, err := s.GetLastChapterMetadata(mangaURL)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	lastUploadChapter.Type = 1
	mangaReturn.LastUploadChapter = lastUploadChapter

	// Get cover img
	var coverFileName string
	for _, relationship := range mangaAPIResp.Data.Relationships {
		if relationship.Type == "cover_art" {
			coverFileName = relationship.Attributes["fileName"].(string)
		}
	}
	if coverFileName == "" {
		return nil, util.AddErrorContext(fmt.Errorf("Cover image not found"), errorContext)
	}
	coverURL := fmt.Sprintf("%s/covers/%s/%s", baseUploadsURL, mangadexMangaID, coverFileName)
	mangaReturn.CoverImgURL = coverURL

	coverImg, resized, err := util.GetImageFromURL(coverURL, 3, 1*time.Second)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	mangaReturn.CoverImgResized = resized
	mangaReturn.CoverImg = coverImg

	return mangaReturn, nil
}

type getMangaAPIResponse struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     struct {
		ID            string                `json:"id"`
		Type          string                `json:"type"`
		Attributes    mangaAttributes       `json:"attributes"`
		Relationships []genericRelationship `json:"relationships"`
	}
}

// getMangaID returns the ID of a manga given its URL
// URL should be like: https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi
func getMangaID(mangaURL string) (string, error) {
	errorContext := "Error while getting manga ID from URL"

	pattern := `/title/([0-9a-fA-F-]+)(?:/.*)?$`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", util.AddErrorContext(err, errorContext)
	}

	matches := re.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", util.AddErrorContext(fmt.Errorf("Manga ID not found"), errorContext)
	}

	return matches[1], nil
}
