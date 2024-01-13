package main

import (
	"database/sql"
	"fmt"

	"github.com/spf13/viper"
)

func getJob() {
	log := loggerGet()
	db, err := sql.Open("postgres", "")
	if err != nil {
		log.Error().Msg("Having hard time connecting to database")
		log.Err(err)
		defer db.Close()
	}
	rows, err := db.Query("SELECT id, name, url from jobs where status='queued'")
	if err != nil {
		log.Error().Msg("Error querying DB for new jobs")
		log.Err(err)
		defer rows.Close()
	}
	for rows.Next() {
		var id int
		var name string
		var url string
		err = rows.Scan(&id, &name, &url)
		if err != nil {
			log.Error().Msg("Error getting job")
			log.Err(err)
		}

		log.Info().Msgf("New job on a horizon! Job ID: %v URL: %v Name: %v", id, url, name)
		//TODO: check before creating

		var containerImage = viper.GetString("IMG_OPERATOR")
		jobName := fmt.Sprintf("print-operator-%v", id)
		entryCommand := "/bin/bash /opt/start.sh"

		launchK8sJobOperator(&jobName, &containerImage, &entryCommand, id, name, url)
	}
	defer db.Close()
	defer rows.Close()
	if err != nil {
		log.Error().Msg("Error during iteration.")
		log.Err(err)
	}
}
