#!/bin/sh
set -e

cd "${WORKING_DIR:-.}"

# Check if any files are not formatted.
set +e
test -z "$(stencil -input ${DOCKERFILE:-Dockerfile} >${DOCKERFILE_STENCIL:-Dockerfile.stencil})"
SUCCESS=$?
set -e

# Exit if `go fmt` passes.
if [ $SUCCESS -eq 0 ]; then
  exit 0
fi


# Post results back as comment.
COMMENT="#### \`cat /tmp/errors\`"
"
PAYLOAD=$(echo '{}' | jq --arg body "$COMMENT" '.body = $body')
COMMENTS_URL=$(cat /github/workflow/event.json | jq -r .pull_request.comments_url)
curl -s -S -H "Authorization: token $GITHUB_TOKEN" --header "Content-Type: application/json" --data "$PAYLOAD" "$COMMENTS_URL" > /dev/null

exit $SUCCESS
