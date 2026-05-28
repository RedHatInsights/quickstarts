#!/bin/bash
# Create the test database alongside the main quickstarts database.
# This script is mounted into the PostgreSQL container's init directory.
psql -U postgres -c "SELECT 1 FROM pg_database WHERE datname = 'quickstarts_test'" | grep -q 1 || \
  psql -U postgres -c "CREATE DATABASE quickstarts_test OWNER quickstarts;"
psql -U postgres -d quickstarts_test -c "CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;"
