#!/bin/bash
set -e


HEAD_SHA=$(git rev-parse HEAD)
LAST_TAG=$(git describe --tags --abbrev=0 --match "v*" HEAD^)

if [ -z "$LAST_TAG" ]; then
  # Use the first commit
  FIRST_COMMIT=$(git rev-list --max-parents=0 HEAD)
  RANGE="$FIRST_COMMIT..$HEAD_SHA"
  START=$FIRST_COMMIT
else
  RANGE="$LAST_TAG..$HEAD_SHA"
  START=$LAST_TAG
fi

echo
echo "Date (UTC): $(date -u)"
echo "Alphabetical order by author."
echo "---------------------------------------------"
echo

# Generate the changelog grouped by author with commit count
git log $RANGE --pretty=format:"%an%x00- %s (%h)" --no-merges | \
  awk -F '\0' '
  {
    author = $1
    commit = $2
    commits_by_author[author] = commits_by_author[author] "\n" commit
    commit_count[author]++
    authors[author] = author
  }
  END {
    n = asorti(authors, sorted_authors)  # Sort authors alphabetically
    for (i = 1; i <= n; i++) {
      author = sorted_authors[i]
      print author " (" commit_count[author] " commits):"
      print commits_by_author[author]
      print ""
    }
  }'

echo
echo "Changelog built starting from '$START' to '$HEAD_SHA'."
echo "---------------------------------------------"
echo