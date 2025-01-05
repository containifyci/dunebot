package cmd

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"

	"github.com/rs/zerolog/log"

	"github.com/alexedwards/scs"
	"github.com/palantir/go-githubapp/oauth2"
)

const (
	SessionKeyUsername = "username"
	SessionKeyRedirect = "redirect"
)

func Login(c config.GithubConfig, sessions *scs.Manager) oauth2.LoginCallback {
	return func(w http.ResponseWriter, r *http.Request, login *oauth2.Login) {
		client := github.Newclient(login.Client)

		// TODO(bkeyes): this should be in baseapp or something
		// I should be able to get a valid, parsed URL
		u, err := url.Parse(strings.TrimSuffix(c.V3APIURL, "/") + "/")
		if err != nil {
			log.Error().Err(err).Msgf("failed to parse github url: %s", err)
			return
		}
		client.BaseURL = u

		user, _, err := client.Users.Get(r.Context(), "")
		if err != nil {
			log.Error().Err(err).Msgf("failed to get github user: %s", err)
			return
		}

		sess := sessions.Load(r)
		if err := sess.PutString(w, SessionKeyUsername, user.GetLogin()); err != nil {
			log.Error().Err(err).Msgf("failed to save session: %s", err)
			return
		}

		// go to root or back to the previous page
		target, err := sess.GetString(SessionKeyRedirect)
		if err != nil {
			log.Error().Err(err).Msgf("failed to read session: %s", err)
			return
		}

		http.Redirect(w, r, target, http.StatusFound)
	}
}

func RequireLogin(sessions *scs.Manager, basePath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := sessions.Load(r)

			user, err := sess.GetString(SessionKeyUsername)
			if err != nil {
				log.Error().Err(err).Msgf("failed to read session: %s", err)
				return
			}

			if user == "" {
				if err := sess.PutString(w, SessionKeyRedirect, basePath+r.URL.String()); err != nil {
					log.Error().Err(err).Msgf("failed to save session: %s", err)
					return
				}

				http.Redirect(w, r, basePath+oauth2.DefaultRoute, http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
