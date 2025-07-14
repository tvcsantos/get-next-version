package git_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Masterminds/semver"
	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thenativeweb/get-next-version/conventionalcommits"
	"github.com/thenativeweb/get-next-version/git"
	"github.com/thenativeweb/get-next-version/testutil"
)

type commit struct {
	message string
	tag     string
	files   []string // relative fake files associated with the commit
}

var DefaultFiles = []string{"src/main.go", "README.md", "CHANGELOG.md"}

func TestGetConventionalCommitTypesSinceLatestRelease(t *testing.T) {
	tests := []struct {
		commitHistory                   []commit
		doExpectError                   bool
		expectedLastVersion             *semver.Version
		expectedConventionalCommitTypes []conventionalcommits.Type
		annotateTags                    bool
		commitsFilterPathRegex          string
		tagsFilterRegex                 string
		versionRegex                    string
	}{
		{
			commitHistory:                   []commit{},
			doExpectError:                   true,
			expectedLastVersion:             nil,
			expectedConventionalCommitTypes: []conventionalcommits.Type{},
			annotateTags:                    false,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "chore: Do something", tag: "", files: DefaultFiles},
			},
			doExpectError:                   false,
			expectedLastVersion:             semver.MustParse("0.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{conventionalcommits.Chore},
			annotateTags:                    false,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "Last release", tag: "1.0.0", files: DefaultFiles},
				{message: "Do something", tag: "", files: DefaultFiles},
			},
			doExpectError:                   false,
			expectedLastVersion:             semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{conventionalcommits.Chore},
			annotateTags:                    false,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "chore: Do something", tag: "1.0.0", files: DefaultFiles},
			},
			doExpectError:                   false,
			expectedLastVersion:             semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{},
			annotateTags:                    false,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "chore: irrelevant", tag: "0.0.1", files: DefaultFiles},
				{message: "feat: because it is", tag: "", files: DefaultFiles},
				{message: "feat(scope)!: before the last tag", tag: "0.0.2", files: DefaultFiles},
				{message: "chore: Do something", tag: "1.0.0", files: DefaultFiles},
				{message: "chore: Do something else", tag: "", files: DefaultFiles},
			},
			doExpectError:                   false,
			expectedLastVersion:             semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{conventionalcommits.Chore},
			annotateTags:                    false,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "chore: irrelevant", tag: "v0.0.1", files: DefaultFiles},
				{message: "feat: because it is", tag: "", files: DefaultFiles},
				{message: "feat(scope)!: before the last tag", tag: "0.0.2", files: DefaultFiles},
				{message: "chore: Do something", tag: "v1.0.0", files: DefaultFiles},
				{message: "chore: Do something else", tag: "", files: DefaultFiles},
			},
			doExpectError:                   false,
			expectedLastVersion:             semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{conventionalcommits.Chore},
			annotateTags:                    false,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "chore: Do something", tag: "1.0.0", files: DefaultFiles},
				{message: "chore: non breaking", tag: "", files: DefaultFiles},
				{message: "fix: non breaking", tag: "", files: DefaultFiles},
				{message: "feat: non breaking", tag: "", files: DefaultFiles},
				{message: "chore!: breaking", tag: "", files: DefaultFiles},
				{message: "fix(with scope)!: breaking", tag: "", files: DefaultFiles},
				{message: "feat: breaking\n\nBREAKING-CHANGE: with footer", tag: "", files: DefaultFiles},
			},
			doExpectError:       false,
			expectedLastVersion: semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{
				conventionalcommits.Chore,
				conventionalcommits.Fix,
				conventionalcommits.Feature,
				conventionalcommits.BreakingChange,
				conventionalcommits.BreakingChange,
				conventionalcommits.BreakingChange,
			},
			annotateTags:           false,
			commitsFilterPathRegex: "",
			tagsFilterRegex:        "",
			versionRegex:           "",
		},
		{
			commitHistory: []commit{
				{message: "Last release", tag: "1.0.0", files: DefaultFiles},
				{message: "fix: Do something", tag: "", files: DefaultFiles},
			},
			doExpectError:                   false,
			expectedLastVersion:             semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{conventionalcommits.Fix},
			annotateTags:                    true,
			commitsFilterPathRegex:          "",
			tagsFilterRegex:                 "",
			versionRegex:                    "",
		},
		{
			commitHistory: []commit{
				{message: "chore: Do something", tag: "1.0.0", files: DefaultFiles},
				{message: "chore: non breaking", tag: "", files: DefaultFiles},
				{message: "fix: non breaking", tag: "", files: DefaultFiles},
				{message: "feat: non breaking", tag: "", files: DefaultFiles},
				{message: "chore!: breaking", tag: "", files: DefaultFiles},
				{message: "fix(with scope)!: breaking", tag: "", files: DefaultFiles},
				{message: "feat: breaking\n\nBREAKING-CHANGE: with footer", tag: "", files: DefaultFiles},
			},
			doExpectError:       false,
			expectedLastVersion: semver.MustParse("0.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{
				conventionalcommits.Chore,
				conventionalcommits.Chore,
				conventionalcommits.Fix,
				conventionalcommits.Feature,
				conventionalcommits.BreakingChange,
				conventionalcommits.BreakingChange,
				conventionalcommits.BreakingChange,
			},
			annotateTags:           false,
			commitsFilterPathRegex: "",
			tagsFilterRegex:        "component-.*",
			versionRegex:           "component-(.*)",
		},
		{
			commitHistory: []commit{
				{message: "chore: Do something", tag: "2.0.0", files: DefaultFiles},
				{message: "chore: non breaking", tag: "", files: DefaultFiles},
				{message: "fix: non breaking", tag: "component-1.0.0", files: DefaultFiles},
				{message: "feat: non breaking", tag: "", files: DefaultFiles},
				{message: "chore!: breaking", tag: "", files: DefaultFiles},
				{message: "fix(with scope)!: breaking", tag: "", files: DefaultFiles},
				{message: "feat: breaking\n\nBREAKING-CHANGE: with footer", tag: "", files: DefaultFiles},
			},
			doExpectError:       false,
			expectedLastVersion: semver.MustParse("1.0.0"),
			expectedConventionalCommitTypes: []conventionalcommits.Type{
				conventionalcommits.Feature,
				conventionalcommits.BreakingChange,
				conventionalcommits.BreakingChange,
				conventionalcommits.BreakingChange,
			},
			annotateTags:           false,
			commitsFilterPathRegex: "",
			tagsFilterRegex:        "component-.*",
			versionRegex:           "component-(.*)",
		},
	}

	for _, test := range tests {
		repoDir := t.TempDir()
		repository, err := gogit.PlainInit(repoDir, false)
		require.NoError(t, err)

		worktree, err := repository.Worktree()
		require.NoError(t, err)

		for _, commit := range test.commitHistory {
			filePaths := make([]string, len(commit.files))
			for j, file := range commit.files {
				filePaths[j] = filepath.Join(repoDir, file)
				err = os.MkdirAll(filepath.Dir(filePaths[j]), 0755)
				require.NoError(t, err)
				err = os.WriteFile(filePaths[j], []byte(commit.message), 0644)
				require.NoError(t, err)
				_, err = worktree.Add(file)
				require.NoError(t, err)
			}

			commitOptions := testutil.CreateCommitOptions()
			_, err = worktree.Commit(commit.message, commitOptions)
			require.NoError(t, err)

			if commit.tag == "" {
				continue
			}

			head, err := repository.Head()
			require.NoError(t, err)

			var createTagOpts *gogit.CreateTagOptions
			if test.annotateTags {
				createTagOpts = &gogit.CreateTagOptions{
					Message: "some message",
					Tagger:  commitOptions.Author,
				}
			}
			_, err = repository.CreateTag(commit.tag, head.Hash(), createTagOpts)
			require.NoError(t, err)
		}

		var versionRegex *regexp.Regexp
		var commitsFilterPathRegex *regexp.Regexp
		var tagsFilterRegex *regexp.Regexp

		if test.versionRegex != "" {
			versionRegex, err = regexp.Compile(test.versionRegex)
			require.NoError(t, err)
		}
		if test.commitsFilterPathRegex != "" {
			commitsFilterPathRegex, err = regexp.Compile(test.commitsFilterPathRegex)
			require.NoError(t, err)
		}
		if test.tagsFilterRegex != "" {
			tagsFilterRegex, err = regexp.Compile(test.tagsFilterRegex)
			require.NoError(t, err)
		}

		classifier := conventionalcommits.NewTypeClassifier()
		actual, err := git.GetConventionalCommitTypesSinceLastRelease(
			repository,
			classifier,
			commitsFilterPathRegex,
			tagsFilterRegex,
			versionRegex,
		)

		if test.doExpectError {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)

		// The test in the next line is not optimal. We rely on the Equal
		// function of the SemVer module here, which considers v1.0.0 and
		// 1.0.0 to be the same. In contrast to this, assert.Equal fails
		// when comparing these two versions, due to the leading v.
		assert.True(t, test.expectedLastVersion.Equal(actual.LatestReleaseVersion))
		assert.ElementsMatch(t, test.expectedConventionalCommitTypes, actual.ConventionalCommitTypes)
	}
}
