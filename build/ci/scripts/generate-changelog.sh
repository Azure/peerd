#!/bin/bash
set -e

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~")
HEAD_TAG=$(git rev-parse HEAD)
if [ "$LAST_TAG" == "$HEAD_TAG" ]; then
  echo
  echo "No new commits since last release tag '$LAST_TAG'."
  exit 0
fi
echo
echo "Changelog from '$HEAD_TAG' to '$LAST_TAG':"
echo
echo "Date (UTC): $(date -u)"
echo "(Alphabetical order by author)"
echo "---------------------------------------------"

# Generate the changelog grouped by author with commit count
git log $LAST_TAG..HEAD --pretty=format:"%an%x00- %s (%h)" --no-merges | \
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

echo "---------------------------------------------"
echo