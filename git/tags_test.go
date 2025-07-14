package git_test

import (
	"regexp"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tvcsantos/get-next-version/git"
	"github.com/tvcsantos/get-next-version/testutil"
)

func TestGetAllSemVerTags(t *testing.T) {
	tests := []struct {
		tagsPerBranch    map[string][][]string
		doesExpectError  bool
		expectedTagNames []string
		tagsFilterRegex  string
		versionRegex     string
	}{
		{
			tagsPerBranch:    map[string][][]string{"main": {{"1.0.0"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"1.0.0"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"v1.0.0"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"1.0.0"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"1.0.0"}, {"2.0.0"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"1.0.0", "2.0.0"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"1.0.0"}, {"2.0.0"}}, "feature": {{"3.0.0"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"1.0.0", "2.0.0", "3.0.0"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"1.0.0"}, {"2.0.0"}, {"feature-tag"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"1.0.0", "2.0.0"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			// The 'v' prefix in 'vsomething-else' is intentionally used here because it is the prefix for 'version'.
			tagsPerBranch:    map[string][][]string{"main": {{"1.0.0"}, {"v2.0.0"}, {"vsomething-else"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"1.0.0", "2.0.0"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"1.0.0", "2.0.0"}}},
			doesExpectError:  true,
			expectedTagNames: []string{},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"v4", "v4.5", "v4.5.14"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"4.5.14"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"v4.0.0", "v5.0.0"}}},
			doesExpectError:  true,
			expectedTagNames: []string{},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"v4.1.0", "v4.2.0"}}},
			doesExpectError:  true,
			expectedTagNames: []string{},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"4", "v4.5", "4.5.14"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"4.5.14"},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"v4", "v4.5", "v4.5.14"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"4.5.14"},
			tagsFilterRegex:  "v\\d+\\.\\d+\\.\\d+",
			versionRegex:     "v(\\d+\\.\\d+\\.\\d+)",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"v4", "v4.5", "4.5.14"}}},
			doesExpectError:  false,
			expectedTagNames: []string{},
			tagsFilterRegex:  "v\\d+\\.\\d+\\.\\d+",
			versionRegex:     "v(\\d+\\.\\d+\\.\\d+)",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"common-4.5.14"}}},
			doesExpectError:  false,
			expectedTagNames: []string{},
			tagsFilterRegex:  "",
			versionRegex:     "",
		},
		{
			tagsPerBranch:    map[string][][]string{"main": {{"common-4.5.14"}}},
			doesExpectError:  false,
			expectedTagNames: []string{"4.5.14"},
			tagsFilterRegex:  "",
			versionRegex:     "common-(.+)",
		},
	}

	for _, test := range tests {
		repository, err := testutil.SetUpInMemoryRepository()
		require.NoError(t, err)

		for branchName, tagNames := range test.tagsPerBranch {
			worktree, err := repository.Worktree()
			require.NoError(t, err)

			worktree.Checkout(&gogit.CheckoutOptions{
				Create: true,
				Branch: plumbing.ReferenceName(branchName),
			})

			for _, tagNamesForCommit := range tagNames {
				worktree.Commit("some message", testutil.CreateCommitOptions())
				head, err := repository.Head()
				require.NoError(t, err)

				for _, tagName := range tagNamesForCommit {
					repository.CreateTag(tagName, head.Hash(), nil)
				}
			}
		}

		var versionRegex *regexp.Regexp
		var tagsFilterRegex *regexp.Regexp

		if test.versionRegex != "" {
			versionRegex, err = regexp.Compile(test.versionRegex)
			require.NoError(t, err)
		}
		if test.tagsFilterRegex != "" {
			tagsFilterRegex, err = regexp.Compile(test.tagsFilterRegex)
			require.NoError(t, err)
		}

		tags, err := git.GetAllSemVerTags(repository, tagsFilterRegex, versionRegex)

		if test.doesExpectError {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)
		var tagNames []string
		for _, tag := range tags {
			tagNames = append(tagNames, tag.String())
		}
		assert.ElementsMatch(t, test.expectedTagNames, tagNames)
	}
}
