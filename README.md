# quickstarts

Backed service for integrated quickstarts.

## TO test
1. star the server `go run main.go `
2. insert some data 
```sh
curl --location --request POST 'http://localhost:8000/api/quickstarts/v1/quickstarts/' \
  --data-raw $'{"bundles":["settings","insights"],"title":"Foo","content":{"apiVersion":"console.openshift.io/v1","kind":"QuickStarts","metadata":{"name":"monitor-sampleapp"},"spec":{"version":4.7,"displayName":"Monitoring your sample application","durationMinutes":10,"icon":{"key":null,"ref":null,"props":{"color":"currentColor","size":"sm","noVerticalAlign":false},"_owner":null},"description":"Now that you’ve created a sample application and added health checks, let’s monitor your application.","prerequisites":["You completed the \\"Getting started with a sample\\" quick start."],"introduction":"### This quick start shows you how to monitor your sample application.\\nYou should have previously created the **sample-app** application and **nodejs-sample** deployment via the **Get started with a sample** quick start. If you haven\'t, you may be able to follow these tasks with any existing deployment.","tasks":[{"title":"Viewing the monitoring details of your sample application","description":"### To view the details of your sample application:\\n1. Go to the project your sample application was created in.\\n2. In the **</> Developer** perspective, go to **Topology** view.\\n3. Click on the **nodejs-sample** deployment to view its details.\\n4. Click on the **Monitoring** tab in the side panel.\\nYou can see context sensitive metrics and alerts in the **Monitoring** tab.","review":{"instructions":"#### To verify you can view the monitoring information:\\n1. Do you see a **Metrics** accordion in the side panel?\\n2. Do you see a **View monitoring dashboard** link in the **Metrics** accordion?\\n3. Do you see three charts in the **Metrics** accordion: **CPU Usage**, **Memory Usage** and **Receive Bandwidth**?","failedTaskHelp":"This task isn’t verified yet. Try the task again."},"summary":{"success":"You have learned how you can monitor your sample app\u0021","failed":"Try the steps again."}},{"title":"Viewing your project monitoring dashboard","description":"### To view the project monitoring dashboard in the context of **nodejs-sample**:\\n1. Click on the **View monitoring dashboard** link in the side panel.\\n2. You can change the **Time Range** and **Refresh Interval** of the dashboard.\\n3. You can change the context of the dashboard as well by clicking on the drop-down list. Select a specific workload or **All Workloads** to view the dashboard in the context of the entire project.","review":{"instructions":"#### To verify that you are able to view the monitoring dashboard:\\nDo you see metrics charts in the dashboard?","failedTaskHelp":"This task isn’t verified yet. Try the task again."},"summary":{"success":"You have learned how to view the dashboard in the context of your sample app\u0021","failed":"Try the steps again."}},{"title":"Viewing custom metrics","description":"### To view custom metrics:\\n1. Click on the **Metrics** tab of the **Monitoring** page.\\n2. Click the **Select Query** drop-down list to see the available queries.\\n3. Click on **Filesystem Usage** from the list to run the query.","review":{"instructions":"#### Verify you can see the chart associated with the query:\\nDo you see a chart displayed with filesystem usage for your project?  Note: select **Custom Query** from the dropdown to create and run a custom query utilizing PromQL.\\n","failedTaskHelp":"This task isn’t verified yet. Try the task again."},"summary":{"success":"You have learned how to run a query\u0021","failed":"Try the steps again."}}],"conclusion":"You have learned how to access workload monitoring and metrics\u0021","nextQuickStart":[""]}}}' \
  --compressed
```
3. query data: 
```sh
curl --location --request GET 'http://localhost:8000/api/quickstarts/v1/quickstarts/'
```

### IMPORTANT
`oc port-forward -n quickstarts svc/quickstarts-service 8000:8000`!

## Sample requests

### Create progress

```sh
curl --location --request POST 'http://localhost:8000/api/quickstarts/v1/progress/2' --header 'Content-Type: application/json' --data-raw '{
"accountId": 1, "progress": {"Some": "Progress"}
}'
```
