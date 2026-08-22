package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Pratyay360/probot-go"
	"github.com/google/go-github/v88/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	defaultIssueTitle = "announcement"
	defaultIssueLabel = "newsletter"
	maxFileChars      = 20000
)

type pushConfig struct {
	track      string
	dest       string
	issueTitle string
	issueLabel string
}

func main() {
	loadDotEnv()

	opts, err := probot.OptionsFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid options from environment")
	}

	cfg := loadPushConfig()
	app, err := probot.New(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create app")
	}

	registerHandlers(app, cfg)

	fmt.Printf("probot %s listening on http://%s:%d%s\n",
		app.Version(), opts.Host, opts.Port, app.WebhookPath())

	server := probot.NewServer(probot.ServerOptions{Probot: app})
	if err := server.Start(); err != nil {
		log.Fatal().Err(err).Msg("server ")
	}
}

func loadDotEnv() {
	if err := probot.LoadEnvFile(".env"); err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msg("failed to load .env")
	}
	if cwd, err := os.Getwd(); err == nil {
		log.Info().Str("cwd", cwd).Msg(".env is read from and written to this directory")
	}
}

func loadPushConfig() pushConfig {
	return pushConfig{
		track:      os.Getenv("TRACK_REPO"),
		dest:       os.Getenv("DEST_REPO"),
		issueTitle: envOrDefault("ISSUE_TITLE", defaultIssueTitle),
		issueLabel: envOrDefault("ISSUE_LABEL", defaultIssueLabel),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func registerHandlers(app *probot.Probot, cfg pushConfig) {
	app.On("push", makePushHandler(cfg))

	app.OnAny(func(ctx *probot.Context) error {
		l := ctx.Log()
		l.Info().
			Str("name", ctx.Name()).
			Str("delivery_id", ctx.ID()).
			Msg("received webhook event")
		return nil
	})

	app.OnError(func(ctx *probot.Context, err error) {
		log.Error().Err(err).Str("event", ctx.Name()).Msg("webhook handler failed")
	})
}

func makePushHandler(cfg pushConfig) func(ctx *probot.Context) error {
	return func(ctx *probot.Context) error {
		l := ctx.Log()

		pushRepo, err := ctx.Repo()
		if err != nil {
			l.Error().Err(err).Msg("failed to get repository from context")
			return nil
		}
		if !shouldHandlePush(pushRepo, cfg.track, l) {
			return nil
		}

		commitSummary, changedFiles, err := pushSummary(ctx.Payload())
		if err != nil {
			l.Error().Err(err).Msg("failed to extract info from payload")
			return err
		}
		if commitSummary == "" && len(changedFiles) == 0 {
			return nil
		}

		destOwner, destRepo := resolveDestRepo(pushRepo, cfg.dest, l)
		body := buildAnnouncementBody(ctx, pushRepo, commitSummary, changedFiles, l)

		return upsertAnnouncementIssue(ctx, destOwner, destRepo, cfg.issueTitle, cfg.issueLabel, body, l)
	}
}

func shouldHandlePush(repo probot.Repo, track string, l zerolog.Logger) bool {
	if track == "" {
		return true
	}
	got := repo.Owner + "/" + repo.Repo
	if track != got {
		l.Debug().Str("track", track).Str("got", got).Msg("push is not tracked")
		return false
	}
	return true
}

func resolveDestRepo(pushRepo probot.Repo, dest string, l zerolog.Logger) (owner, repo string) {
	if dest == "" {
		return pushRepo.Owner, pushRepo.Repo
	}
	if o, r, ok := splitRepo(dest); ok {
		return o, r
	}
	l.Warn().Str("dest_repo", dest).Msg("DEST_REPO")
	return pushRepo.Owner, pushRepo.Repo
}

func buildAnnouncementBody(ctx *probot.Context, pushRepo probot.Repo, commitSummary string, changedFiles []string, l zerolog.Logger) string {
	var b strings.Builder
	b.WriteString(commitSummary)
	appendChangedFilesList(&b, changedFiles)
	appendFileContents(ctx, &b, pushRepo, changedFiles, l)
	return b.String()
}

func appendChangedFilesList(b *strings.Builder, files []string) {
	if len(files) == 0 {
		return
	}
	b.WriteString("\n**New Post:**\n")
	for _, f := range files {
		b.WriteString("- `")
		b.WriteString(f)
		b.WriteString("`\n")
	}
}

func appendFileContents(ctx *probot.Context, b *strings.Builder, pushRepo probot.Repo, files []string, l zerolog.Logger) {
	if len(files) == 0 {
		return
	}
	headSHA := getHeadSHA(ctx.Payload())
	for _, file := range files {
		content, err := fetchFileContent(ctx, pushRepo.Owner, pushRepo.Repo, file, headSHA)
		if err != nil {
			l.Warn().Err(err).Str("file", file).Msg("failed to fetch file content; skipping")
			continue
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		content, truncated := truncateContent(content, maxFileChars)
		content = strings.ReplaceAll(content, "```", "` `` `")

		b.WriteString(file)
		b.WriteString("\n\n")
		b.WriteString(content)
		if truncated {
			b.WriteString("\n... (truncated)\n")
		}
	}
}

func truncateContent(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	return s[:limit], true
}

func upsertAnnouncementIssue(ctx *probot.Context, destOwner, destRepo, issueTitle, issueLabel, body string, l zerolog.Logger) error {
	existingIssue, err := findOpenIssueByLabel(ctx, destOwner, destRepo, issueLabel)
	if err != nil {
		l.Error().Err(err).Msg("failed to search for existing issue")
	}

	if existingIssue != nil {
		return addCommentToExistingIssue(ctx, destOwner, destRepo, existingIssue, body, l)
	}
	return createNewAnnouncementIssue(ctx, destOwner, destRepo, issueTitle, issueLabel, body, l)
}

func addCommentToExistingIssue(ctx *probot.Context, owner, repo string, issue *github.Issue, body string, l zerolog.Logger) error {
	comment, err := createCommentWithLockHandling(ctx, owner, repo, issue, body)
	if err != nil {
		l.Error().Err(err).Msg("failed to add comment")
		return nil
	}
	if comment != nil {
		l.Info().Str("url", comment.GetHTMLURL()).Msg("added comment")
	}
	return nil
}

func createNewAnnouncementIssue(ctx *probot.Context, owner, repo, title, label, body string, l zerolog.Logger) error {
	issue, err := createIssue(ctx, owner, repo, title, label, body)
	if err != nil {
		l.Error().Err(err).Msg("failed to create issue")
		return nil
	}
	if issue != nil {
		l.Info().Str("url", issue.GetHTMLURL()).Msg("created issue")
	}
	return nil
}

func checkLocked(ctx *probot.Context, owner, repo string, issue *github.Issue) (bool, error) {
	if issue == nil || !issue.GetLocked() {
		return false, nil
	}
	if _, err := ctx.GitHub().Issues.Unlock(context.Background(), owner, repo, issue.GetNumber()); err != nil {
		return false, fmt.Errorf("failed to unlock issue #%d: %w", issue.GetNumber(), err)
	}
	l := ctx.Log()
	l.Info().Int("issue", issue.GetNumber()).Msg("unlocked locked issue for commenting")
	return true, nil
}

func lockIssue(ctx *probot.Context, owner, repo string, issue *github.Issue) error {
	var opts *github.LockIssueOptions
	if reason := issue.GetActiveLockReason(); reason != "" {
		opts = &github.LockIssueOptions{LockReason: reason}
	}
	if _, err := ctx.GitHub().Issues.Lock(context.Background(), owner, repo, issue.GetNumber(), opts); err != nil {
		return fmt.Errorf("failed to re-lock issue #%d: %w", issue.GetNumber(), err)
	}
	l := ctx.Log()
	l.Info().Int("issue", issue.GetNumber()).Msg("re-locked issue")
	return nil
}

func createCommentWithLockHandling(ctx *probot.Context, owner, repo string, issue *github.Issue, body string) (*github.IssueComment, error) {
	wasLocked, err := checkLocked(ctx, owner, repo, issue)
	if err != nil {
		return nil, err
	}

	comment, err := createComment(ctx, owner, repo, issue.GetNumber(), body)
	if err != nil {
		if wasLocked {
			if lockErr := lockIssue(ctx, owner, repo, issue); lockErr != nil {
				l := ctx.Log()
				l.Warn().Err(lockErr).Int("issue", issue.GetNumber()).Msg("failed to re-lock issue after comment error")
			}
		}
		return nil, err
	}

	if wasLocked {
		if err := lockIssue(ctx, owner, repo, issue); err != nil {
			l := ctx.Log()
			l.Warn().Err(err).Int("issue", issue.GetNumber()).Msg("comment added but failed to re-lock issue")
			return comment, err
		}
	}
	return comment, nil
}

func findOpenIssueByLabel(ctx *probot.Context, owner, repo, label string) (*github.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State:       "open",
		Labels:      []string{label},
		ListOptions: github.ListOptions{PerPage: 1},
	}
	issues, _, err := ctx.GitHub().Issues.ListByRepo(context.Background(), owner, repo, opts)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, nil
	}
	return issues[0], nil
}

func createComment(ctx *probot.Context, owner, repo string, issueNumber int, body string) (*github.IssueComment, error) {
	comment, _, err := ctx.GitHub().Issues.CreateComment(context.Background(), owner, repo, issueNumber, &github.IssueComment{
		Body: &body,
	})
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func createIssue(ctx *probot.Context, owner, repo, title, label, body string) (*github.Issue, error) {
	labels := []string{label}
	issueReq := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		Labels: &labels,
	}
	issue, _, err := ctx.GitHub().Issues.Create(context.Background(), owner, repo, issueReq)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func fetchFileContent(ctx *probot.Context, owner, repo, path, ref string) (string, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	file, _, _, err := ctx.GitHub().Repositories.GetContents(context.Background(), owner, repo, path, opts)
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", fmt.Errorf("file not found: %s", path)
	}
	if file.GetType() == "dir" {
		return "", nil
	}
	content, err := file.GetContent()
	if err != nil {
		return "", err
	}
	return content, nil
}

func splitRepo(s string) (owner, repo string, ok bool) {
	owner, repo, found := strings.Cut(strings.TrimSpace(s), "/")
	return owner, repo, found && owner != "" && repo != ""
}

func getHeadSHA(payload map[string]any) string {
	if sha, ok := payload["after"].(string); ok && sha != "" && sha != "0000000000000000000000000000000000000000" {
		return sha
	}
	if head, ok := payload["head_commit"].(map[string]any); ok {
		if id, ok := head["id"].(string); ok && id != "" {
			return id
		}
	}
	return ""
}

func pushSummary(payload map[string]any) (summary string, changedFiles []string, err error) {
	commits, ok := payload["commits"].([]any)
	if !ok || len(commits) == 0 {
		return "", nil, nil
	}

	var sb strings.Builder
	seen := make(map[string]bool)

	for _, c := range commits {
		commit, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if msg, ok := commit["message"].(string); ok && strings.TrimSpace(msg) != "" {
			short := strings.SplitN(msg, "\n", 2)[0]
			if author, ok := commit["author"].(map[string]any); ok {
				if name, ok := author["name"].(string); ok && name != "" {
					fmt.Fprintf(&sb, "- %s (%s)\n", short, name)
				} else {
					fmt.Fprintf(&sb, "- %s\n", short)
				}
			} else {
				fmt.Fprintf(&sb, "- %s\n", short)
			}
		}

		for _, field := range []string{"added", "removed", "modified"} {
			if files, ok := commit[field].([]any); ok {
				for _, f := range files {
					name, ok := f.(string)
					if ok && !seen[name] {
						seen[name] = true
						changedFiles = append(changedFiles, name)
					}
				}
			}
		}
	}
	return sb.String(), changedFiles, nil
}
