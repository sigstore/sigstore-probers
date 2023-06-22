#!/usr/bin/env bash
#
# Copyright 2023 The Sigstore Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

# Gets the name of the currently running workflow file.
# Note: this requires GITHUB_TOKEN to be set in the workflows.
this_file() {
    curl -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID | jq -r '.path' | cut -d '/' -f3
}

issue_body() {
    RUN_DATE=$(date --utc)
    EXTRA_MESSAGE=$1

    # see https://docs.github.com/en/actions/learn-github-actions/environment-variables
    # https://docs.github.com/en/actions/learn-github-actions/contexts.
    BODY=$(cat <<EOF
Repo: https://github.com/$GITHUB_REPOSITORY/tree/$GITHUB_REF_NAME
Run: https://github.com/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID
Workflow file: https://github.com/$GITHUB_REPOSITORY/tree/main/.github/workflows/$THIS_FILE
Workflow runs: https://github.com/$GITHUB_REPOSITORY/actions/workflows/$THIS_FILE
Trigger: $GITHUB_EVENT_NAME
Branch: $GITHUB_REF_NAME
Date: $RUN_DATE

$EXTRA_MESSAGE
EOF
)    
}

# creates a github issue
create_issue() {
    # if issue is not failure or success, error
    ISSUE_TYPE_UPPERCASE=$(echo "$ISSUE_TYPE" | tr '[:lower:]' '[:upper:]')    
    if [[ "$ISSUE_TYPE_UPPERCASE" != "FAILURE" && "$ISSUE_TYPE_UPPERCASE" != "SUCCESS" ]]; then
        echo "ISSUE_TYPE must be either 'FAILURE' or 'SUCCESS'"
        return 1
    fi
    THIS_FILE=$(this_file)
    ISSUE_ID=$(gh -R "$ISSUE_REPOSITORY" issue list --label "bug" --state open -S "$THIS_FILE" --json number | jq '.[0]' | jq -r '.number' | jq 'select (.!=null)')

    if [[ "$ISSUE_TYPE_UPPERCASE" == "FAILURE" ]]; then    
        # on failure create a new issue
        issue_body "Opening a new issue as tests are failing."

        if [[ -z "$ISSUE_ID" ]]; then
            # Replace `-`` by ` `, remove the last 4 characters `.yml`. Expected: "snapshot timestamp".
            TITLE=$(echo "$THIS_FILE" | sed -e 's/\-/ /g' | rev | cut -c5- | rev)
            echo gh -R "$ISSUE_REPOSITORY" issue create -t "[bug]: Updating workflow $TITLE" -F ./BODY --label "bug"
            GH_TOKEN=$GITHUB_TOKEN gh -R "$ISSUE_REPOSITORY" issue create -t "[bug]: Updating workflow $TITLE" -F ./BODY --label "bug"
        else
            echo gh -R "$ISSUE_REPOSITORY" issue comment "$ISSUE_ID" -F ./BODY
            GH_TOKEN=$GITHUB_TOKEN gh -R "$ISSUE_REPOSITORY" issue comment "$ISSUE_ID" -F ./BODY
        fi
    else
        issue_body "Tests are passing now. Closing this issue."

        # on success close it
        echo gh -R "$ISSUE_REPOSITORY" issue close "$ISSUE_ID" -c "$(cat ./BODY)"
        GH_TOKEN=$GITHUB_TOKEN gh -R "$ISSUE_REPOSITORY" issue close "$ISSUE_ID" -c "$(cat ./BODY)"    
    fi
}