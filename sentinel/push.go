package sentinel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v31/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type PushHandler struct {
	githubapp.ClientCreator
}

func (h *PushHandler) Handles() []string {
	return []string{"push"}
}

func (h *PushHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PushEvent

	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse push event payload")
	}

	var owner string

	if event.Repo.Organization != nil {
		owner = *event.Repo.Organization
	} else if event.Repo.Owner != nil {
		owner = *event.Repo.Owner
	} else {
		return errors.New("expected either the organization or owner to be set, but none were on " + *event.Repo.Name)
	}

	// TODO: Actually check for .argoci.yaml
	if false {
		zerolog.Ctx(ctx).Debug().Msgf("no .argoci.yaml was found on %s/%s", owner, *event.Repo.Name)
		return nil
	}

	// TODO, Check the .argoci.yaml for branch name matching
	if false {

	}

	// TODO: Verify Workflow either exists if fileRef, or is valid if inline source

	if !event.GetIssue().IsPullRequest() {
		zerolog.Ctx(ctx).Debug().Msg("Issue comment event is not for a pull request")
		return nil
	}

	repo := event.GetRepo()
	prNum := event.GetIssue().GetNumber()
	installationID := githubapp.GetInstallationIDFromEvent(&event)

	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, event.GetIssue().GetNumber())

	logger.Debug().Msgf("Event action is %s", event.GetAction())
	if event.GetAction() != "created" {
		return nil
	}

	client, err := h.NewInstallationClient(installationID)
	if err != nil {
		return err
	}

	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	author := event.GetComment().GetUser().GetLogin()
	body := event.GetComment().GetBody()

	if strings.HasSuffix(author, "[bot]") {
		logger.Debug().Msg("Issue comment was created by a bot")
		return nil
	}

	logger.Debug().Msgf("Echoing comment on %s/%s#%d by %s", repoOwner, repoName, prNum, author)
	msg := fmt.Sprintf("%s\n%s said\n```\n%s\n```\n", h.preamble, author, body)
	prComment := github.IssueComment{
		Body: &msg,
	}

	if _, _, err := client.Issues.CreateComment(ctx, repoOwner, repoName, prNum, &prComment); err != nil {
		logger.Error().Err(err).Msg("Failed to comment on pull request")
	}

	return nil
}
