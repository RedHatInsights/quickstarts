# quickstarts

Backed service for integrated quickstarts.

## TO test
1. run the `run_docker.sh`
2. star the server `go run main.go `
3. insert some data 
```sh
curl --location --request POST 'http://localhost:8888/api/quickstarts/v1/quickstarts/' --header 'Content-Type: application/json' --data-raw '{
"Title": "New quickstart"
}'

```
4. query data: 
```sh
curl --location --request GET 'http://localhost:8888/api/quickstarts/v1/quickstarts/'
```