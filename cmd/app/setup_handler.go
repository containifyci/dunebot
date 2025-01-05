package cmd

import (
	"context"
	"net/http"
	"strconv"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	errorReporter interface {
		Errorf(err error, format string, args interface{})
	}

	logErrorReporter struct{
		logger zerolog.Logger
	}
)

var logError errorReporter = logErrorReporter{
	logger: log.Logger,
}

func (l logErrorReporter) Errorf(err error, format string, args interface{}) {
	l.logger.Error().Err(err).Msgf(format, args)
}

func registerSetup(cfg config.ConfigTransformer, appClient *github.Client, ghc github.GithubClientCreator, mux *http.ServeMux) {
	// Define a simple HTML template
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Title}}</h1>
    <a href="{{.VerificationLink}}" target="_blank">{{.Message}}</a>
    <p>{{.Code}}</p>
</body>
</html>
`
	mux.Handle("/app/github/setup", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		installIdStr := r.URL.Query().Get("installation_id")
		installId, err := strconv.ParseInt(installIdStr, 10, 64)

		if err != nil {
			http.Error(w, "Invalid installation ID", http.StatusBadRequest)
			return
		}

		install, _, err := appClient.Apps.GetInstallation(context.Background(), installId)
		if err != nil {
			log.Error().Err(err).Msgf("Error requesting installation: %v\n", err)
			http.Error(w, "Error requesting installation", http.StatusInternalServerError)
			return
		}
		log.Debug().Msgf("Valid installation: %v\n", install)

		flow, fnc, err := oauth2DeviceFlow(cfg, logError)
		if err != nil {
			log.Error().Err(err).Msgf("Error requesting device flow: %v\n", err)
			http.Error(w, "Error requesting device flow", http.StatusInternalServerError)
			return
		}
		go func() {
			fnc(ghc, install)
		}()
		// Data to be rendered in the HTML template
		page := Page{
			Title:            "In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.",
			Message:          "Please open",
			VerificationLink: flow.VerificationURI,
			Code:             flow.UserCode,
		}

		cnt, _, err := templateFnc(htmlTemplate, "index", page)
		if err != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
			return
		}
		_, err = w.Write([]byte(*cnt))
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write response: %v\n", err)
			return
		}
	}))
}
