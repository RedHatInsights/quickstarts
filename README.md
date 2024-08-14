# Quickstarts

Backend service for integrated quickstarts.
## [Help Topics contribution guide](https://github.com/RedHatInsights/quickstarts/blob/main/docs/help-topics/README.md)

## [Quickstarts (Learning resources) contribution guide](https://github.com/RedHatInsights/quickstarts/blob/main/docs/quickstarts/README.md)

## [Quickstarts Common Issues](https://github.com/RedHatInsights/frontend-components/blob/master/packages/docs/pages/quickstarts/common-issues.mdx)

## Run the service locally
1. There are environment variables required for the application to start. It's
recommended you copy `.env.example` to `.env` and set these appropriately for local development.
2. Migrate the database: `make migrate`. It will seed the BD with testing quickstart
3. Start the server: `go run main.go`
4. Query data:
```sh
curl --location --request GET 'http://localhost:8000/api/quickstarts/v1/quickstarts/'

curl --location --request GET 'http://localhost:8000/api/quickstarts/v1/quickstarts/?bundle[]=rhel&bundle[]=insights'
```

### IMPORTANT
`oc port-forward -n quickstarts svc/quickstarts-service 8000:8000`!

## Sample requests

### Create progress

```sh
curl --location --request POST 'http://localhost:8000/api/quickstarts/v1/progress' --header 'Content-Type: application/json' --data-raw '{
"accountId": 123, "quickstartName": "some-name", "progress": {"Some": "Progress-updated"}
}'

```
### Delete progress

```sh
curl --location --request DELETE 'http://localhost:8000/api/quickstarts/v1/progress/14'
```
