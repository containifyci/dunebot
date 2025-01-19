package cmd

import (
	// "html/template"
	"fmt"
	"net/http"

	appoauth "github.com/containifyci/dunebot/oauth2"
	"github.com/containifyci/oauth2-storage/pkg/proto"

	"github.com/alexedwards/scs"
	"github.com/containifyci/dunebot/pkg/auth"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"

	// "github.com/containifyci/dunebot/pkg/storage"
	"github.com/containifyci/dunebot/pkg/template"
	"github.com/rs/zerolog/log"
)

var templateFnc = template.New

type WebHandler struct {
	sessions *scs.Manager
}

type SessionHandler interface {
	GetString(req *http.Request, key string) (string, error)
}

func NewWebHandler(sessions *scs.Manager) *WebHandler {
	return &WebHandler{sessions: sessions}
}

func (wh *WebHandler) GetString(req *http.Request, key string) (string, error) {
	sess := wh.sessions.Load(req)
	return sess.GetString(key)
}

// TODO implement rendering of login user token dashboard
func DashboardHandler(cfg *config.Config, appClient *github.Client, ghc github.GithubClientCreator, sessions SessionHandler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log.Debug().Msgf("Dashboard Request %v+\n", r)
		user, err := sessions.GetString(r, SessionKeyUsername)
		if err != nil {
			log.Error().Err(err).Msgf("failed to read session: %v\n", err)
			http.Error(w, "Error reading session", http.StatusInternalServerError)
			return
		}
		log.Debug().Msgf("Dashboard Request For User %v+\n", user)

		installion, err := FindInstallation(ctx, appClient, user)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to retrieve installation: %v\n", err)
			http.Error(w, "Error reading installation", http.StatusInternalServerError)
			return
		}

		installationId := fmt.Sprintf("%d", installion.GetID())

		// installationId, err := sess.GetInt64(SessionKeyInstallation)
		// if err != nil {
		// 	fmt.Printf("failed to read session: %v\n", err)
		// }
		// fmt.Printf("Dashboard Request For Installation %v+\n", installationId)

		authSrv := auth.NewSigningService(cfg.JWT.PrivateKey)
		accessToken := authSrv.CreateTokenFnc(auth.ServiceClaims{ServiceName: "dunebot"})

		tokenClient, close, err := appoauth.NewClient(*appoauth.NewAuthInterceptor(accessToken), cfg.JWT.Address)
		defer close()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to connect to gRPC server: %v\n", err)
			http.Error(w, "Failed to connect to gRPC server", http.StatusInternalServerError)
			return
		}
		install, err := tokenClient.RetrieveInstallation(r.Context(), &proto.Installation{InstallationId: installationId})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to retrieve installation: %v\n", err)
			http.Error(w, "Failed to retrieve installation", http.StatusInternalServerError)
			return
		}
		users := make([]string, 0)
		for _, token := range install.Tokens {
			users = append(users, token.User)
		}

		// Data to be rendered in the HTML template
		page := struct {
			Users []string
		}{
			Users: users,
		}

		oauth2cfg := appoauth.Config{
			InstallationId:  installationId,
			User:            user,
			Ctx:             ctx,
			OAuth2Config:    appoauth.GetConfig(cfg),
			AuthInterceptor: *appoauth.NewAuthInterceptor(accessToken),
			Addr:            cfg.JWT.Address,
		}

		ghu := ghc.NewClient(github.WithTokenSource(oauth2cfg.TokenSourceFrom(ctx)), github.WithContext(ctx), github.WithConfig(github.NewConfig()) /*, github.WithConfig(github.NewRepositoryConfig(repoOwner, repoName))*/)

		githubUser, _, err := ghu.Client.Users.Get(ctx, "")
		if err != nil {
			log.Error().Err(err).Msgf("Error getting user from token source: %v\n", err)
			http.Error(w, "Error getting user from token source", http.StatusInternalServerError)
			return
		}

		log.Debug().Msgf("Token Source User: %v\n", githubUser)

		//TODO how to retrieve user orgs
		orgs, _, err := ghu.Client.Organizations.List(ctx, "", nil)
		// orgs, _, err := gh.Organizations.List(ctx, "", nil)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to retrieve user orgs: %v\n", err)
			http.Error(w, "Failed to retrieve user orgs", http.StatusInternalServerError)
			return
		}
		log.Debug().Msgf("Orgs %v+\n", orgs)

		// Create a new template and parse the HTML template
		cnt, _, err := templateFnc(htmlTemplate, "dashboard", page)
		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			return
		}
		_, err = w.Write([]byte(*cnt))
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write response: %v\n", err)
			return
		}
	})
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
		<title>Authenticated Users</title>
</head>
<body>
		<h1>Authenticated Users</h1>
		{{- range $i, $user := .Users -}}
			<div>
					<p>{{$user}}</p>
					<hr class="horizontal_line_class" />
			</div>
		{{- end -}}
</body>
</html>
`
