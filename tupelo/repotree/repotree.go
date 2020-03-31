package repotree

import (
	"context"

	tupelo "github.com/quorumcontrol/tupelo-go-sdk/gossip/client"

	"github.com/quorumcontrol/dgit/tupelo/namedtree"
)

const repoSalt = "decentragit-0.0.0-alpha"

var namedTreeGen *namedtree.Generator

func init() {
	namedTreeGen = &namedtree.Generator{Namespace: repoSalt}
}

type Options struct {
	*namedtree.Options
}

func Find(ctx context.Context, repo string, client *tupelo.Client) (*namedtree.NamedTree, error) {
	namedTreeGen.Client = client
	return namedTreeGen.Find(ctx, repo)
}

func Create(ctx context.Context, opts *Options) (*namedtree.NamedTree, error) {
	namedTree, err := namedTreeGen.New(ctx, opts.Options)
	if err != nil {
		return nil, err
	}

	return namedTree, nil
}
