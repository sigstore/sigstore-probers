#!/usr/bin/env bash

set -e

echo "Searching for attestation with digest: $1"

UUIDS=$(rekor-cli search --sha $1 --rekor_server $REKOR_SERVER)
# Since there could be many of the entries there and it takes a bit of time
# to spin through all of these, grab the current time up front instead
# of each time through the loop so we want to check the freshness 5 minutes
# from the start of the script run, not each iteration.
current_time=$(date +%s)

for uuid in $UUIDS
do
    entry=$(rekor-cli get --uuid $uuid --format json --rekor_server $REKOR_SERVER)

    # make sure this entry was integrated within the last 5 minutes
    integrated_time=$(echo $entry | jq -r .IntegratedTime)
    five_minutes_ago=$(( $current_time - 360 )) # current_time is in seconds

    if ((  $five_minutes_ago < $integrated_time  )); then
        #  Make sure the attestation is not empty
        attestation=$(echo $entry | jq -r .Attestation)
        if [[ $attestation == "" ]]; then
            echo "Attestation field in entry $uuid is empty: $attestation"
            exit 1
        fi
        echo "Found valid attestation with uuid $uuid"
        exit 0
    fi

done

echo "Failed to find attestation in log"
exit 1
