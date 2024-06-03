package mangahub

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/diogovalentte/mantium/api/src/errors"
	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

// GetMangaMetadata scrapes the manga page and return the manga data
func (s *Source) GetMangaMetadata(mangaURL string, ignoreGetLastChapterError bool) (*manga.Manga, error) {
	s.resetCollector()

	errorContext := "Error while getting manga metadata"

	mangaReturn := &manga.Manga{}
	mangaReturn.Source = "mangahub.io"
	mangaReturn.URL = mangaURL

	lastChapter := &manga.Chapter{
		Type: 1,
	}
	mangaReturn.LastUploadChapter = lastChapter

	var sharedErr error

	// manga name
	s.c.OnHTML("h1._3xnDj", func(e *colly.HTMLElement) {
		// The h1 tag with the manga's name
		// has a small tag inside it with the
		// manga description that we don't want.
		// It can also have an <a> tag with the
		// manga's name and the word "Hot".
		name := e.Text
		smallTagValue := e.DOM.Find("small").Text()
		aTagValue := e.DOM.Find("a").Text()
		name = strings.Replace(name, smallTagValue, "", -1)
		name = util.RemoveLastOccurrence(name, aTagValue)

		mangaReturn.Name = name
	})

	// manga cover
	s.c.OnHTML("img.manga-thumb", func(e *colly.HTMLElement) {
		mangaReturn.CoverImgURL = e.Attr("src")
	})

	// last upload chapter
	isFirstUL := true
	s.c.OnHTML("ul.MWqeC:first-of-type > li:first-child a._3pfyN", func(e *colly.HTMLElement) {
		if !isFirstUL {
			return
		}
		isFirstUL = false

		chapterURL := e.Attr("href")

		chapterStr := e.DOM.Find("span._3D1SJ").Text()
		chapter := strings.TrimSpace(strings.Replace(chapterStr, "#", "", -1))

		chapterName := e.DOM.Find("span._2IG5P").Text()
		chapterName = strings.TrimSpace(strings.Replace(chapterName, "- ", "", -1))

		uploadedAt := e.DOM.Find("small.UovLc").Text()
		uploadedTime, err := getMangaUploadedTime(uploadedAt)
		if err != nil {
			sharedErr = err
			return
		}

		mangaReturn.LastUploadChapter = &manga.Chapter{
			URL:       chapterURL,
			Chapter:   chapter,
			Name:      chapterName,
			Type:      1,
			UpdatedAt: uploadedTime,
		}
	})

	err := s.c.Visit(mangaURL)
	if err != nil {
		if err.Error() == "Not Found" {
			return nil, util.AddErrorContext(fmt.Errorf("Manga not found"), errorContext)
		}
		return nil, util.AddErrorContext(util.AddErrorContext(err, "Error while visiting manga URL"), errorContext)
	}
	if sharedErr != nil {
		return nil, util.AddErrorContext(sharedErr, errorContext)
	}

	if mangaReturn.LastUploadChapter == nil && !ignoreGetLastChapterError {
		return nil, util.AddErrorContext(errors.ErrLastReleasedChapterNotFound, errorContext)
	}

	// get cover image
	coverImg, resized, err := s.getCoverImg(mangaReturn.CoverImgURL, 3, 1*time.Second)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	mangaReturn.CoverImgResized = resized
	mangaReturn.CoverImg = coverImg

	return mangaReturn, nil
}
