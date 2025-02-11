#!/bin/bash

# Exit on error
set -e

# Configuration
PROJECT_ID="fast-audio-449913-b6"
REGION="europe-north1"
REPOSITORY="qwen-repo"
SERVICE_NAME="qwen2-service"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Setting up Google Cloud project...${NC}"
gcloud config set project $PROJECT_ID

echo -e "${GREEN}Enabling required APIs...${NC}"
gcloud services enable \
    artifactregistry.googleapis.com \
    run.googleapis.com

echo -e "${GREEN}Creating Artifact Registry repository...${NC}"
gcloud artifacts repositories create $REPOSITORY \
    --repository-format=docker \
    --location=$REGION \
    --description="Repository for Qwen2.5 images" || true

echo -e "${GREEN}Building and pushing the container...${NC}"
gcloud builds submit \
    --tag ${REGION}-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$SERVICE_NAME \
    -f Dockerfile.qwen .

echo -e "${GREEN}Deploying to Cloud Run...${NC}"
gcloud run deploy $SERVICE_NAME \
    --image ${REGION}-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$SERVICE_NAME \
    --platform managed \
    --memory 4Gi \
    --cpu 2 \
    --port 8080 \
    --region $REGION \
    --allow-unauthenticated

echo -e "${GREEN}Deployment complete! Getting the service URL...${NC}"
gcloud run services describe $SERVICE_NAME \
    --region $REGION \
    --format='value(status.url)' 