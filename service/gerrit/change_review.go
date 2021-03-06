package gerrit

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/serviceutil"
	"golang.org/x/build/gerrit"
)

var _ reviewdog.CommentService = &ChangeReviewCommenter{}

// ChangeReviewCommenter is a comment service for Gerrit Change Review
// API:
// 	https://gerrit-review.googlesource.com/Documentation/rest-api-changes.html#set-review
// 	POST /changes/{change-id}/revisions/{revision-id}/review
type ChangeReviewCommenter struct {
	cli        *gerrit.Client
	changeID   string
	revisionID string

	muComments   sync.Mutex
	postComments []*reviewdog.Comment

	// wd is working directory relative to root of repository.
	wd string
}

// NewChangeReviewCommenter returns a new NewChangeReviewCommenter service.
// ChangeReviewCommenter service needs git command in $PATH.
func NewChangeReviewCommenter(cli *gerrit.Client, changeID, revisionID string) (*ChangeReviewCommenter, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("ChangeReviewCommenter needs 'git' command: %v", err)
	}

	return &ChangeReviewCommenter{
		cli:          cli,
		changeID:     changeID,
		revisionID:   revisionID,
		postComments: []*reviewdog.Comment{},
		wd:           workDir,
	}, nil
}

// Post accepts a comment and holds it. Flush method actually posts comments to Gerrit
func (g *ChangeReviewCommenter) Post(_ context.Context, c *reviewdog.Comment) error {
	c.Result.Path = filepath.Join(g.wd, c.Result.Path)
	g.muComments.Lock()
	defer g.muComments.Unlock()
	g.postComments = append(g.postComments, c)
	return nil
}

// Flush posts comments which has not been posted yet.
func (g *ChangeReviewCommenter) Flush(ctx context.Context) error {
	g.muComments.Lock()
	defer g.muComments.Unlock()

	return g.postAllComments(ctx)
}

func (g *ChangeReviewCommenter) postAllComments(ctx context.Context) error {
	review := gerrit.ReviewInput{
		Comments: map[string][]gerrit.CommentInput{},
	}
	for _, c := range g.postComments {
		if !c.Result.InDiffFile {
			continue
		}
		review.Comments[c.Result.Path] = append(review.Comments[c.Result.Path], gerrit.CommentInput{
			Line:    c.Result.Lnum,
			Message: c.Body,
		})
	}

	return g.cli.SetReview(ctx, g.changeID, g.revisionID, review)
}
