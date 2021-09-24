podman run -d --name postgresql_database -e POSTGRESQL_USER=user -e POSTGRESQL_PASSWORD=pass -e POSTGRESQL_DATABASE=quickstarts -p 5432:5432 rhel8/postgresql-13
