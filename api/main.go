// Package main implements the init and main function
package main

import (
	"io"
	"time"

	"github.com/rs/zerolog"

	"github.com/diogovalentte/mantium/api/src"
	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/diogovalentte/mantium/api/src/util"
)

func init() {
	// You can set the path to use an .env file below.
	// It can be an absolute path or relative to this file (main.go)
	filePath := ""
	if err := config.SetConfigs(filePath); err != nil {
		panic(err)
	}

	log := util.GetLogger()

	log.Info().Msg("Trying to connect to DB...")
	_db, err := db.OpenConn()
	if err != nil {
		panic(err)
	}
	defer _db.Close()

	err = db.CreateTables(_db, log)
	if err != nil {
		panic(err)
	}

	setUpdateMangasMetadataPeriodicallyJob(log)
}

func main() {
	router := api.SetupRouter()
	router.SetTrustedProxies(nil)

	router.Run()
}

func setUpdateMangasMetadataPeriodicallyJob(log *zerolog.Logger) {
	configs := config.GlobalConfigs.PeriodicallyUpdateMangas
	if configs.Update {
		log.Info().Msg("Starting to update mangas metadata periodically...")

		if configs.Notify {
			log.Info().Msg("Will notify when updating mangas metadata")
		} else {
			log.Info().Msg("Will not notify when updating mangas metadata")
		}

		log.Info().Msgf("Will update mangas metadata every %d minutes", configs.Minutes)
		log.Info().Msgf("First update in %d minutes", configs.Minutes)

		go func() {
			for {
				time.Sleep(time.Duration(configs.Minutes) * time.Minute)

				log.Info().Msg("Updating mangas metadata...")
				res, err := util.RequestUpdateMangasMetadata(configs.Notify)
				if err != nil {
					log.Error().Msgf("Error updating mangas metadata: %s", err)
					log.Error().Msgf("Request response: %s", res)
					body, err := io.ReadAll(res.Body)
					if err != nil {
						log.Error().Msgf("Error while getting the response body: %s", err)
					}
					log.Error().Msgf("Request response text: %s", string(body))
				} else {
					log.Info().Msg("Mangas metadata updated")
				}
			}
		}()
	} else {
		log.Info().Msg("Not updating mangas metadata periodically")
	}
}
