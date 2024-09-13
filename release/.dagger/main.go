package main

import (
	"context"
	"dagger/release/internal/dagger"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/moby/buildkit/identity"
	"golang.org/x/sync/errgroup"
)

type Release struct {
	UnixInstallScript    *dagger.File // +private
	WindowsInstallScript *dagger.File // +private
	Tag                  string
	ChangeNotes          *dagger.Directory // +private
}

func New(
	ctx context.Context,
	// +optional
	gitRef string,
	// +optional
	// +defaultPath="/"
	// +ignore=["*", "!.git/HEAD", "!.git/refs", "!.git/config", "!.git/objects/*"]
	gitDir *dagger.Directory,
	// +optional
	// +defaultPath="install.sh"
	unixInstallScript *dagger.File,
	// +optional
	// +defaultPath="install.ps1"
	windowsInstallScript *dagger.File,
	// +optional
	// +defaultPath="get-ref.sh"
	getRefScript *dagger.File,
	// +optional
	// +defaultPath="/"
	// +ignore=["*", "!.changes/*.md", "!**/.changes/*.md"]
	changeNotes *dagger.Directory,
) (Release, error) {
	if gitRef == "" {
		// FIXME: this doesn't always work in github actions
		gitRef, err := dag.
			Wolfi().
			Container(dagger.WolfiContainerOpts{Packages: []string{"git"}}).
			WithMountedDirectory("/src", gitDir).
			WithWorkdir("/src").
			WithFile("/bin/get-ref.sh", getRefScript).
			WithExec([]string{"sh", "/bin/get-ref.sh"}).
			Stdout(ctx)
		if err != nil {
			return Release{}, err
		}
		gitRef = strings.TrimRight(gitRef, "\n")
	}
	return Release{
		UnixInstallScript:    unixInstallScript,
		WindowsInstallScript: windowsInstallScript,
		Tag:                  gitRef, // FIXME: is this correct?
		ChangeNotes:          changeNotes,
	}, nil
}

// Lint scripts files
func (r Release) Lint(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return dag.Shellcheck().
			Check(r.UnixInstallScript).
			Assert(ctx)
	})
	eg.Go(func() error {
		return dag.PsAnalyzer().
			Check(r.WindowsInstallScript, dagger.PsAnalyzerCheckOpts{
				// Exclude the unused parameters for now due because PSScriptAnalyzer treat
				// parameters in `Install-Dagger` as unused but the script won't run if we delete
				// it.
				ExcludeRules: []string{"PSReviewUnusedParameter"},
			}).
			Assert(ctx)
	})
	return eg.Wait()
}

// Test the release process
func (r Release) Test(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return dag.GoSDK().TestPublish(ctx, r.Tag)
	})
	eg.Go(func() error {
		// Test the release process of the Python SDK
		// FIXME: move this to ../sdk/python/dev
		return r.PublishPythonSDK(ctx, true, "", nil, "https://github.com/dagger/dagger.git", nil)
	})
	return eg.Wait()
}

// Publish the Python SDK
// FIXME: move this to ../sdk/python/dev
func (r Release) PublishPythonSDK(
	ctx context.Context,

	// +optional
	dryRun bool,

	// +optional
	pypiRepo string,
	// +optional
	pypiToken *dagger.Secret,

	// +optional
	// +default="https://github.com/dagger/dagger.git"
	gitRepoSource string,
	// +optional
	githubToken *dagger.Secret,
) error {
	version, isVersioned := strings.CutPrefix(r.Tag, "sdk/python/")
	if dryRun {
		version = "v0.0.0"
	}
	if pypiRepo == "" || pypiRepo == "pypi" {
		pypiRepo = "main"
	}

	// TODO: move this to PythonSDKDev
	result := dag.PythonSDKDev().
		Container().
		WithEnvVariable("SETUPTOOLS_SCM_PRETEND_VERSION", strings.TrimPrefix(version, "v")).
		WithEnvVariable("HATCH_INDEX_REPO", pypiRepo).
		WithEnvVariable("HATCH_INDEX_USER", "__token__").
		WithExec([]string{"uvx", "hatch", "build"})
	if !dryRun {
		result = result.
			WithSecretVariable("HATCH_INDEX_AUTH", pypiToken).
			WithExec([]string{"uvx", "hatch", "publish"})
	}
	_, err := result.Sync(ctx)
	if err != nil {
		return err
	}
	if isVersioned {
		return r.GithubRelease(
			ctx,
			r.Tag,
			r.ChangeNotesFile("sdk/python", version),
			gitRepoSource,
			githubToken,
			dryRun,
		)
	}
	return nil
}

func (r Release) ChangeNotesFile(component, version string) *dagger.File {
	return r.ChangeNotes.File(fmt.Sprintf("%s/.changes/%s.md", component, version))
}

// Publish an SDK to a git repository
func (r Release) GitPublish(
	ctx context.Context,
	// Source repository URL
	// +optional
	source string,
	// Destination repository URL
	// +optional
	dest string,
	// Tag or reference in the source repository
	// +optional
	sourceTag string,
	// Tag or reference in the destination repository
	// +optional
	destTag string,
	// Path within the source repository to publish
	// +optional
	sourcePath string,
	// Filter to apply to the source files
	// +optional
	sourceFilter string,
	// Container environment for source operations
	// +optional
	sourceEnv *dagger.Container,
	// Git username for commits
	// +optional
	username string,
	// Git email for commits
	// +optional
	email string,
	// GitHub token for authentication
	// +optional
	githubToken *dagger.Secret,
	// Whether to perform a dry run without pushing changes
	// +optional
	dryRun bool,
) error {
	base := sourceEnv
	if base == nil {
		base = dag.Wolfi().
			Container(dagger.WolfiContainerOpts{
				Packages: []string{
					"git",
					"go",
					"python3",
				},
			})
	}
	// FIXME: move this into std modules
	git := base.
		WithExec([]string{"git", "config", "--global", "user.name", username}).
		WithExec([]string{"git", "config", "--global", "user.email", email})
	if !dryRun {
		githubTokenRaw, err := githubToken.Plaintext(ctx)
		if err != nil {
			return err
		}
		encodedPAT := base64.URLEncoding.EncodeToString([]byte("pat:" + githubTokenRaw))
		git = git.
			WithEnvVariable("GIT_CONFIG_COUNT", "1").
			WithEnvVariable("GIT_CONFIG_KEY_0", "http.https://github.com/.extraheader").
			WithSecretVariable("GIT_CONFIG_VALUE_0", dag.SetSecret("GITHUB_HEADER", fmt.Sprintf("AUTHORIZATION: Basic %s", encodedPAT)))
	}

	result := git.
		WithEnvVariable("CACHEBUSTER", identity.NewID()).
		WithWorkdir("/src/dagger").
		WithExec([]string{"git", "clone", source, "."}).
		WithExec([]string{"git", "fetch", "origin", "-v", "--update-head-ok", fmt.Sprintf("refs/*%[1]s:refs/*%[1]s", strings.TrimPrefix(sourceTag, "refs/"))}).
		WithEnvVariable("FILTER_BRANCH_SQUELCH_WARNING", "1").
		WithExec([]string{
			"git", "filter-branch", "-f", "--prune-empty",
			"--subdirectory-filter", sourcePath,
			"--tree-filter", sourceFilter,
			"--", sourceTag,
		})
	if !dryRun {
		result = result.WithExec([]string{
			"git",
			"push",
			// "--force", // NOTE: disabled to avoid accidentally rewriting the history
			dest,
			fmt.Sprintf("%s:%s", sourceTag, destTag),
		})
	} else {
		// on a dry run, just test that the last state of dest is in the current branch (and is a fast-forward)
		history, err := result.
			WithExec([]string{"git", "log", "--oneline", "--no-abbrev-commit", sourceTag}).
			Stdout(ctx)
		if err != nil {
			return err
		}

		destCommit, err := git.
			WithEnvVariable("CACHEBUSTER", identity.NewID()).
			WithWorkdir("/src/dagger").
			WithExec([]string{"git", "clone", dest, "."}).
			WithExec([]string{"git", "fetch", "origin", "-v", "--update-head-ok", fmt.Sprintf("refs/*%[1]s:refs/*%[1]s", strings.TrimPrefix(destTag, "refs/"))}).
			WithExec([]string{"git", "checkout", destTag, "--"}).
			WithExec([]string{"git", "rev-parse", "HEAD"}).
			Stdout(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "invalid reference: "+destTag) {
				// this is a ref that only exists in the source, and not in the
				// dest, so no overwriting will occur
				return nil
			}
			return err
		}
		destCommit = strings.TrimSpace(destCommit)

		if !strings.Contains(history, destCommit) {
			return fmt.Errorf("publish would rewrite history - %s not found\n%s", destCommit, history)
		}
		return nil
	}

	_, err := result.Sync(ctx)
	return err
}

// Publish a Github release
func (r Release) GithubRelease(
	ctx context.Context,
	// Tag for the GitHub release
	// +optional
	tag string,
	// File containing release notes
	// +optional
	notes *dagger.File,
	// GitHub repository URL
	// +optional
	gitRepo string,
	// GitHub token for authentication
	// +optional
	githubToken *dagger.Secret,
	// Whether to perform a dry run without creating the release
	// +optional
	dryRun bool,
) error {
	u, err := url.Parse(gitRepo)
	if err != nil {
		return err
	}
	if u.Host != "github.com" {
		return fmt.Errorf("git repo must be on github.com")
	}
	githubRepo := strings.TrimPrefix(strings.TrimSuffix(u.Path, ".git"), "/")

	if dryRun {
		// sanity check tag is in target repo
		_, err = dag.
			Git(fmt.Sprintf("https://github.com/%s", githubRepo)).
			Ref(tag).
			Tree().
			Sync(ctx)
		if err != nil {
			return err
		}

		// sanity check notes file exists
		notesContent, err := notes.Contents(ctx)
		if err != nil {
			return err
		}
		fmt.Println(notesContent)

		return nil
	}

	gh := dag.Gh(dagger.GhOpts{
		Repo:  githubRepo,
		Token: githubToken,
	})
	return gh.Release().Create(
		ctx,
		tag,
		tag,
		dagger.GhReleaseCreateOpts{
			VerifyTag: true,
			Draft:     true,
			NotesFile: notes,
			// Latest:    false,  // can't do this yet
		},
	)
}
