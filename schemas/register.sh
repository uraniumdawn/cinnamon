#!/bin/sh
set -e

# Install dependencies (if needed)
apk add --no-cache curl jq

echo "Waiting for Schema Registry..."
timeout=60
while ! curl -sf http://schema-registry:8081/subjects && [ $timeout -gt 0 ]; do
  echo "Still waiting..."; sleep 2; timeout=$((timeout - 2))
done
if [ $timeout -le 0 ]; then
  echo "Schema Registry did not become ready in time."
  exit 1
fi

# Function to register a schema file to a subject
register_schema() {
  SCHEMA_FILE="$1"
  SUBJECT="$2"

  if [ ! -f "$SCHEMA_FILE" ]; then
    echo "Error: Schema file '$SCHEMA_FILE' not found!"
    exit 1
  fi

  echo "Registering schema for subject: $SUBJECT"
  ESCAPED=$(jq -Rs . < "$SCHEMA_FILE")

  RESPONSE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://schema-registry:8081/subjects/$SUBJECT/versions \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d "{\"schema\":$ESCAPED}")

  if [ "$RESPONSE_CODE" -ne 200 ]; then
    echo "Failed to register schema for subject '$SUBJECT'. HTTP status: $RESPONSE_CODE"
    exit 1
  else
    echo "Schema for '$SUBJECT' registered successfully."
  fi
}

# Register specific schemas
register_schema /schemas/ad-impression.avsc ad-impressions-value
register_schema /schemas/ad-click.avsc ad-clicks-value

echo "Schemas registered successfully."