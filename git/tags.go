package git

import (
	"errors"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"regexp"
	"strings"
)

type Tags = map[plumbing.Hash]*semver.Version

type tagCandidate struct {
	originalName string
	version      *semver.Version
}

func getTagSpecificity(tagName string) int {
	cleanTag := tagName
	if strings.HasPrefix(cleanTag, "v") {
		cleanTag = cleanTag[1:]
	}
	return strings.Count(cleanTag, ".")
}

func areCompatibleGranularities(leftVersion, rightVersion *semver.Version) bool {
	if leftVersion.Equal(rightVersion) {
		return true
	}

	if leftVersion.Major() != rightVersion.Major() {
		return false
	}

	// Allow major-only tags (e.g., "v4" represented as v4.0.0)
	if (leftVersion.Minor() == 0 && leftVersion.Patch() == 0) || (rightVersion.Minor() == 0 && rightVersion.Patch() == 0) {
		return true
	}

	if leftVersion.Minor() == rightVersion.Minor() {
		// Allow major.minor tags (e.g., "v4.5" represented as v4.5.0)
		if leftVersion.Patch() == 0 || rightVersion.Patch() == 0 {
			return true
		}
		return leftVersion.Patch() == rightVersion.Patch()
	}

	return false
}

func selectMostSpecificTag(candidates []tagCandidate) *semver.Version {
	if len(candidates) == 1 {
		return candidates[0].version
	}

	mostSpecific := candidates[0]
	maxSpecificity := getTagSpecificity(mostSpecific.originalName)

	for _, candidate := range candidates[1:] {
		specificity := getTagSpecificity(candidate.originalName)
		if specificity > maxSpecificity {
			mostSpecific = candidate
			maxSpecificity = specificity
		}
	}

	return mostSpecific.version
}

func GetAllSemVerTags(repository *git.Repository, tagsFilterPathRegex *regexp.Regexp, versionRegex *regexp.Regexp) (Tags, error) {
	// Algorithm: When multiple tags exist on the same commit, this function distinguishes
	// between acceptable granularity variations (e.g., v4, v4.5, v4.5.14) and conflicting
	// versions (e.g., v4.1.0, v4.2.0). For granularity variations, it selects the most
	// specific tag. For conflicting versions, it returns an error.
	tagsIterator, err := repository.Tags()
	if err != nil {
		return Tags{}, err
	}

	tagsIterator = storer.NewReferenceFilteredIter(func(ref *plumbing.Reference) bool {
		if tagsFilterPathRegex == nil {
			return true
		}
		return tagsFilterPathRegex.MatchString(ref.Name().Short())
	}, tagsIterator)

	var commitTags = make(map[plumbing.Hash][]tagCandidate)

	err = tagsIterator.ForEach(func(tag *plumbing.Reference) error {
		var commitHash plumbing.Hash
		tagObject, err := repository.TagObject(tag.Hash())
		switch err {
		case nil:
			commit, err := tagObject.Commit()
			if err != nil {
				return err
			}
			commitHash = commit.Hash
		case plumbing.ErrObjectNotFound:
			commitHash = tag.Hash()
		default:
			return err
		}

		tagName := tag.Name().Short()

		if versionRegex != nil {
			matches := versionRegex.FindStringSubmatch(tagName)
			if len(matches) > 1 {
				tagName = matches[1]
			}
		}

		version, err := semver.NewVersion(tagName)
		if err != nil {
			// Skip non-semver tags
			return nil
		}

		commitTags[commitHash] = append(commitTags[commitHash], tagCandidate{
			originalName: tag.Name().Short(),
			version:      version,
		})
		return nil
	})
	if err != nil {
		return Tags{}, err
	}

	var tags = make(Tags)
	for commitHash, candidates := range commitTags {
		if len(candidates) > 1 {
			firstVersion := candidates[0].version
			hasDifferentVersions := false

			for _, candidate := range candidates[1:] {
				if !areCompatibleGranularities(firstVersion, candidate.version) {
					hasDifferentVersions = true
					break
				}
			}

			if hasDifferentVersions {
				return Tags{}, errors.New(fmt.Sprintf("commit %s was tagged with multiple semver versions", commitHash.String()))
			}
		}

		tags[commitHash] = selectMostSpecificTag(candidates)
	}

	return tags, nil
}
