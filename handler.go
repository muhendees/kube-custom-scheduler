package main

import (
	"context"
	"strings"

	"database/sql"
	"fmt"

	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

		launchK8sJob(&jobName, &containerImage, &entryCommand, id, name, url)
	}
	defer db.Close()
	defer rows.Close()
	if err != nil {
		log.Error().Msg("Error during iteration.")
		log.Err(err)
	}
}

func launchK8sJob(jobName *string, image *string, cmd *string, id int) {
	log := loggerGet()
	clientset := connectToKubernetes()
	jobs := clientset.BatchV1().Jobs(viper.GetString("NAMESPACE"))
	var backOffLimit int32 = 0
	var ttlSecondsAfterFinished int32 = 30

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *jobName,
			Namespace: viper.GetString("NAMESPACE"),
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Hostname: *jobName,
					NodeSelector: map[string]string{
						"operator-node": "true",
					},
					Containers: []v1.Container{
						{
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									"cpu":    resource.MustParse("4"),
									"memory": resource.MustParse("2000"),
									//TODO: Control resource requests from configuration
								},
							},
							Name:    *jobName,
							Image:   *image,
							Command: strings.Split(*cmd, " "),
							Env: []v1.EnvVar{
								{
									Name:  "DB_USER",
									Value: viper.GetString("PGUSER"),
								},
								{
									Name:  "DB_PWD",
									Value: viper.GetString("PGPASSWORD"),
								},
								{
									Name:  "DB_HOST",
									Value: viper.GetString("PGHOST"),
								},
								{
									Name:  "DB_NAME",
									Value: viper.GetString("PGDATABASE"),
								},
								{
									Name:  "ENV",
									Value: viper.GetString("ENV"),
								},
								{
									Name:  "DEBUG",
									Value: viper.GetString("DEBUG"),
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
					ImagePullSecrets: []v1.LocalObjectReference{
						{
							Name: "gitlabregistrypullsecret",
						},
					},
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

	log.Info().Msg(*jobName)
	log.Info().Msg("Checking if Job is already exists.")
	_, err := clientset.BatchV1().Jobs(viper.GetString("NAMESPACE")).Get(context.Background(), *jobName, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
	})
	if err != nil {
		log.Info().Msg("The Job does not exist. Creating one")
		_, err := jobs.Create(context.Background(), jobSpec, metav1.CreateOptions{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: "batch/v1",
			},
		})
		if statusError, isStatus := err.(*errors.StatusError); isStatus {
			log.Error().Msg("Error creating job")
			log.Error().Msg(statusError.ErrStatus.Message)
			panic(err.Error())
		}
		log.Info().Msgf("Successfully created a new Job: %v", *jobName)
		updateJobRecord(*jobName, id)

	} else {
		log.Info().Msgf("Job name %v already exists. Doing nothing", *jobName)
	}

}
