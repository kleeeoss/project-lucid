#!/bin/bash
awslocal sqs create-queue --queue-name lucid-ci-scan-queue --region us-east-1
awslocal sqs create-queue --queue-name lucid-ci-scan-dlq --region us-east-1