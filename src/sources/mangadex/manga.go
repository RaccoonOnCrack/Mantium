package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/diogovalentte/manga-dashboard-api/src/manga"
	"github.com/diogovalentte/manga-dashboard-api/src/util"
)

// GetMangaMetadata returns the metadata of a manga given its URL
func (s *Source) GetMangaMetadata(mangaURL string) (*manga.Manga, error) {
	s.checkClient()

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangadex.org"
	mangaReturn.URL = mangaURL

	mangadexMangaID, err := getMangaID(mangaURL)
	if err != nil {
		return nil, err
	}

	mangaAPIURL := fmt.Sprintf("%s/manga/%s?includes[]=cover_art", baseAPIURL, mangadexMangaID)
	resp, err := s.client.Request(context.Background(), "GET", mangaAPIURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mangaAPIResp getMangaAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&mangaAPIResp); err != nil {
		return nil, err
	}

	attributes := &mangaAPIResp.Data.Attributes

	mangaReturn.Name = attributes.Title.Values["en"]

	lastUploadedChapterURL := fmt.Sprintf("%s/chapter/%s", baseSiteURL, attributes.LatestUploadedChapter)
	lastUploadChapter, err := s.GetChapterMetadata("", 0, lastUploadedChapterURL)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("cover art not found")
	}
	coverURL := fmt.Sprintf("%s/covers/%s/%s", baseUploadsURL, mangadexMangaID, coverFileName)
	mangaReturn.CoverImgURL = coverURL

	coverImg, err := util.GetImageFromURL(coverURL)
	if err != nil {
		return nil, err
	}
	resizedCoverImg, err := util.ResizeImage(coverImg, uint(util.DefaultImageWidth), uint(util.DefaultImageHeight))
	if err != nil {
		// JPEG format that has an unsupported subsampling ratio
		// It's a valid image but the standard library doesn't support it
		// And other libraries use the standard library under the hood
		if err.Error() == "unsupported JPEG feature: luma/chroma subsampling ratio" {
			resizedCoverImg = coverImg
		} else {
			err = fmt.Errorf("error resizing image: %s", err)
			return nil, err
		}
	} else {
		mangaReturn.CoverImgResized = true
	}
	mangaReturn.CoverImg = resizedCoverImg

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

type mangaAttributes struct {
	Title                          localisedStrings   `json:"title"`
	AltTitles                      []localisedStrings `json:"altTitles"`
	Description                    localisedStrings   `json:"description"`
	IsLocked                       bool               `json:"isLocked"`
	Links                          localisedStrings   `json:"links"`
	OriginalLanguage               string             `json:"originalLanguage"`
	LastVolume                     string             `json:"lastVolume"`
	LastChapter                    string             `json:"lastChapter"`
	PublicationDemographic         string             `json:"publicationDemographic"`
	Status                         string             `json:"status"`
	Year                           int                `json:"year"`
	ContentRating                  string             `json:"contentRating"`
	Tags                           []tag              `json:"tags"`
	State                          string             `json:"state"`
	ChapterNumbersResetOnNewVolume bool               `json:"chapterNumbersResetOnNewVolume"`
	CreatedAt                      string             `json:"createdAt"`
	UpdatedAt                      string             `json:"updatedAt"`
	Version                        int                `json:"version"`
	AvailableTranslatedLanguages   []string           `json:"availableTranslatedLanguages"`
	LatestUploadedChapter          string             `json:"latestUploadedChapter"`
}

type coverAttributes map[string]interface{}

// type coverAttributes struct {
// 	Description string  `json:"description"`
// 	Volume      string  `json:"volume"`
// 	FileName    string  `json:"fileName"`
// 	Locale      string  `json:"locale"`
// 	CreatedAt   string  `json:"createdAt"`
// 	UpdatedAt   string  `json:"updatedAt"`
// 	Version     float64 `json:"version"`
// }

// getMangaID returns the ID of a manga given its URL
// URL should be like: https://mangadex.org/title/87ebd557-8394-4f16-8afe-a8644e555ddc/hirayasumi
func getMangaID(mangaURL string) (string, error) {
	pattern := `https://mangadex\.org/title/([0-9a-fA-F-]+)/`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(mangaURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("manga ID not found in URL")
	}

	return matches[1], nil
}
