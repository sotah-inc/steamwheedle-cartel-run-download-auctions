package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/compute/metadata"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/act"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/logging"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/logging/stackdriver"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/sotah/codes"
	"github.com/sotah-inc/steamwheedle-cartel/pkg/state/run"
)

var port int
var serviceName string
var projectId string
var state run.DownloadAuctionsState

func init() {
	parsedPort, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("Failed to get port: %s", err.Error())

		return
	}

	port = parsedPort
	serviceName = os.Getenv("K_SERVICE")
	projectId, err = metadata.Get("project/project-id")
	if err != nil {
		log.Fatalf("Failed to get port: %s", err.Error())

		return
	}

	state, err = run.NewDownloadAuctionsState(run.DownloadAuctionsStateConfig{ProjectId: projectId})
	if err != nil {
		log.Fatalf("Failed to generate download-auctions state: %s", err.Error())

		return
	}
}

func main() {
	// establishing log verbosity
	logVerbosity, err := logrus.ParseLevel("info")
	if err != nil {
		logging.WithField("error", err.Error()).Fatal("Could not parse log level")

		return
	}
	logging.SetLevel(logVerbosity)

	// adding stackdriver hook
	logging.WithField("project-id", projectId).Info("Creating stackdriver hook")
	stackdriverHook, err := stackdriver.NewHook(projectId, serviceName)
	if err != nil {
		logging.WithFields(logrus.Fields{
			"error":     err.Error(),
			"projectID": projectId,
		}).Fatal("Could not create new stackdriver logrus hook")

		return
	}
	logging.AddHook(stackdriverHook)

	logging.WithField("service", serviceName).Info("Initializing service")

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logging.Info("Received request")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			act.WriteErroneousErrorResponse(w, "Could not read request body", err)

			logging.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Could not read request body")

			return
		}

		msg := state.Run(body)
		switch msg.Code {
		case codes.Ok:
			w.WriteHeader(http.StatusCreated)

			if _, err := fmt.Fprint(w, msg.Data); err != nil {
				logging.WithField("error", err.Error()).Error("Failed to return response")

				return
			}
		case codes.NoAction:
			w.WriteHeader(http.StatusNotModified)
		default:
			act.WriteErroneousMessageResponse(w, "State run code was invalid", msg)

			logging.WithFields(logrus.Fields{
				"code":  msg.Code,
				"error": msg.Err,
				"data":  msg.Data,
			}).Error("State run code was invalid")
		}

		logging.Info("Sent response")
	}).Methods("POST")
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
