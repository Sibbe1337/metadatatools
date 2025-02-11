#!/bin/bash

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "gcloud is not installed. Please install the Google Cloud SDK."
    exit 1
fi

# Install the pubsub emulator if not already installed
gcloud components install pubsub-emulator

# Start the pubsub emulator in the background
gcloud beta emulators pubsub start --host-port=localhost:8085 &

# Wait for emulator to start
sleep 5

# Export the environment variable
export PUBSUB_EMULATOR_HOST=localhost:8085

echo "Pub/Sub emulator is running at localhost:8085" 