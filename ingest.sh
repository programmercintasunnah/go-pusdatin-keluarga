#!/bin/sh
echo "🌦️ Running weather ingest at $(date)"
/app/main --ingest
