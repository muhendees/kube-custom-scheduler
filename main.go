package main

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/spf13/viper"
)

// initConf uses "github.com/spf13/viper"
// based on a given config, collects ENV_VARS
// from deployed environment. In this case from
// container and pod. And sets during run time.
func initConf() {
	viper.AutomaticEnv()

	// PGSQL
	viper.SetDefault("PGPORT", 5433)
	viper.SetDefault("PGHOST", "localhost")
	viper.SetDefault("PGUSER", "customuser")
	viper.SetDefault("PGPASSWORD", "ubersecretpassword")
	viper.SetDefault("PGSSLMODE", "disable")
	//TODO: Add SSL support for DB connection
}

// Sets configuration values for main loop and runs tasks
func main() {

	// Initialize config from ENV variables.
	initConf()
	log := loggerGet()
	cleanUpTick := time.NewTicker(10 * time.Minute)
	jobPingTick := time.NewTicker(3 * time.Minute)
	jobCheckTick := time.NewTicker(2 * time.Second)

	// Attempt connecting to DB.
	_, err := sql.Open("postgres", "")
	if err != nil {
		log.Error().Msg("Database connection failed.")
		log.Err(err)
	} else {
		log.Info().Msg("Database connection successful.")
	}

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Err(err)
		}
	}
	// Set listener params.
	minReconn := 10 * time.Second
	maxReconn := 15 * time.Second
	listener := pq.NewListener("", minReconn, maxReconn, reportProblem)
	err = listener.Listen("custom_scheduler")
	if err != nil {
		log.Error().Msg("Error calling listener for new task \n%v")
		log.Err(err)
		panic(err)
	}
	// Listen to PostgreSQL events and call referenced functions.
	// Very nice feature from `pq` that allows us to Query database only when it is needed.
	for {
		select {
		case <-listener.Notify:
			log.Info().Msg("Received a notification, checking for jobs/events")
			getJob()
			time.Sleep(2 * time.Second)
		case <-jobPingTick.C:
			go listener.Ping()
			log.Info().Msg("Pinging PostgreSQL channel custom_scheduler.")
		case <-cleanUpTick.C:
			jobCleanup()
		case <-jobCheckTick.C:
			dummyCheck(eventCheckEndpoint)
		}
	}
}
