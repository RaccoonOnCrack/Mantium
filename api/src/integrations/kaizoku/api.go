package kaizoku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/diogovalentte/mantium/api/src/manga"
	"github.com/diogovalentte/mantium/api/src/util"
)

func (k *Kaizoku) Request(method, url string, body io.Reader) (*http.Response, error) {
	errorContext := "Error while making '%s' request"

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, method))
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := k.c.Do(req)
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, method))
	}

	return resp, nil
}

func validateResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Non-200 status code -> (%d). Body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (k *Kaizoku) GetSources() ([]string, error) {
	errorContext := "Error while getting manga sources"

	url := fmt.Sprintf("%s/api/trpc/manga.sources", k.Address)
	resp, err := k.Request(http.MethodGet, url, nil)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	var mangas getMangaSources
	err = json.NewDecoder(resp.Body).Decode(&mangas)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	return mangas.Result.Data.JSON, nil
}

func (k *Kaizoku) GetMangas() ([]*Manga, error) {
	errorContext := "Error while getting mangas"

	url := fmt.Sprintf("%s/api/trpc/manga.query", k.Address)
	resp, err := k.Request(http.MethodGet, url, nil)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	var mangas getMangasResponse
	err = json.NewDecoder(resp.Body).Decode(&mangas)
	if err != nil {
		return nil, util.AddErrorContext(err, errorContext)
	}

	return mangas.Result.Data.JSON, nil
}

func (k *Kaizoku) GetManga(mangaName string) (*Manga, error) {
	errorContext := "Error while getting manga with name '%s'"

	mangas, err := k.GetMangas()
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(errorContext, mangaName))
	}

	for _, m := range mangas {
		if m.Title == mangaName {
			return m, nil
		}
	}

	return nil, util.AddErrorContext(fmt.Errorf("Manga not found in Kaizoku"), fmt.Sprintf(errorContext, mangaName))
}

func (k *Kaizoku) AddManga(manga *manga.Manga) error {
	errorContext := "Error while adding manga '%s'"

	mangaTitle := manga.Name
	mangaInterval := k.DefaultInterval
	mangaSource, err := k.getKaizokuSource(manga.Source)
	if err != nil {
		return util.AddErrorContext(err, fmt.Sprintf(errorContext, manga))
	}
	reqBody := fmt.Sprintf(`{"0":{"json":{"title":"%s","source":"%s","interval":"%s"}}}`, mangaTitle, mangaSource, mangaInterval)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error while marshalling request body"), fmt.Sprintf(errorContext, manga)).Error())
	}

	url := fmt.Sprintf("%s/api/trpc/manga.add?batch=1", k.Address)
	resp, err := k.Request(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(err, fmt.Sprintf(errorContext, manga))
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		if util.ErrorContains(err, fmt.Sprintf("Cannot find the %s.", mangaTitle)) {
			return util.AddErrorContext(fmt.Errorf("Cannot find manga. Maybe there is no Anilist page for this manga (Kaizoku can't add mangas that don't have one): Error: %s", err.Error()), fmt.Sprintf(errorContext, manga))
		}
		return util.AddErrorContext(err, fmt.Sprintf(errorContext, manga))
	}

	return nil
}

func (k *Kaizoku) RemoveManga(mangaId int, removeFiles bool) error {
	errorContext := "Error while removing manga with id '%d' (removeFiles: %v)"

	reqBody := fmt.Sprintf(`{"0":{"json":{"id": %d, "shouldRemoveFiles": %v}}}`, mangaId, removeFiles)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error while marshalling request body"), fmt.Sprintf(errorContext, mangaId, removeFiles)).Error())
	}

	url := fmt.Sprintf("%s/api/trpc/manga.remove?batch=1", k.Address)
	resp, err := k.Request(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(err, fmt.Sprintf(errorContext, mangaId, removeFiles))
	}
	defer resp.Body.Close()

	// It returns 500 when the manga is removed with success (weird)
	if resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return util.AddErrorContext(fmt.Errorf("Non-200/500 status code -> (%d). Body: %s", resp.StatusCode, string(body)), fmt.Sprintf(errorContext, mangaId, removeFiles))
	}

	return nil
}

func (k *Kaizoku) CheckOutOfSyncChapters() error {
	errorContext := "Error while checking out of sync chapters"

	reqBody := fmt.Sprintf(`{"0":{"json":{"id": null}}}`)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error while marshalling request body"), errorContext).Error())
	}

	url := fmt.Sprintf("%s/api/trpc/manga.checkOutOfSyncChapters?batch=1", k.Address)
	resp, err := k.Request(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		if util.ErrorContains(err, "There is another active job running. Please wait until it finishes") {
			return util.AddErrorContext(fmt.Errorf("There is another active job running."), errorContext)
		}
		return util.AddErrorContext(err, errorContext)
	}

	return nil
}

func (k *Kaizoku) FixOutOfSyncChapters() error {
	errorContext := "Error while fixing out of sync chapters"

	reqBody := fmt.Sprintf(`{"0":{"json":{"id": null}}}`)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return util.AddErrorContext(err, util.AddErrorContext(fmt.Errorf("Error while marshalling request body"), errorContext).Error())
	}

	url := fmt.Sprintf("%s/api/trpc/manga.fixOutOfSyncChapters?batch=1", k.Address)
	resp, err := k.Request(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return util.AddErrorContext(err, errorContext)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		if util.ErrorContains(err, "There is another active job running. Please wait until it finishes") {
			return util.AddErrorContext(fmt.Errorf("There is another active job running."), errorContext)
		}
		return util.AddErrorContext(err, errorContext)
	}

	return nil
}

type getMangaSources struct {
	Result struct {
		Data struct {
			JSON []string `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

type getMangasResponse struct {
	Result struct {
		Data struct {
			JSON []*Manga `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

func (k *Kaizoku) getKaizokuSource(source string) (string, error) {
	errorContext := "Error while getting Kaizoku source"
	kaizokuSources, err := k.GetSources()
	if err != nil {
		return "", util.AddErrorContext(err, errorContext)
	}

	var returnSource string
	switch source {
	case "mangadex.org":
		returnSource = "MangaDex"
	case "comick.xyz":
		returnSource = "ComicK"
	case "mangahub.io":
		return "", util.AddErrorContext(fmt.Errorf("MangaHub source is not implemented"), errorContext)
	default:
		return "", util.AddErrorContext(fmt.Errorf("Unknown source"), errorContext)
	}

	for _, s := range kaizokuSources {
		if s == returnSource {
			return returnSource, nil
		}
	}

	return "", util.AddErrorContext(fmt.Errorf("Source not found in Kaizoku"), errorContext)
}
