# Help topic

Read more about help topics in the official [documentation](https://github.com/patternfly/patternfly-quickstarts/tree/main/packages/module#in-app--in-context-help-panel)


## Add new help topic

To create a new help topic, follow these steps.

### Create metadata.yml for help topics

1. Create new directory in `docs/help-topics/<topic-name>`
2. Create `metadata.yml` file in the new directory

```yml
kind: HelpTopic # kind must always be "HelpTopic"
name: <name> # this name will be used as an identifier. The file with the content must use the same name `new-topic.yml`
tags: # If you want to use more granular filtering add tags to the topic
  - kind: bundle # use bundle tag for a topic to be accessed from a whole bundle eg. console.redhat.com/insights
    value: insights
  - kind: application # use application tag for topics used by specific application
    value: inventory

```
### Create `<name>.yml` file in new directory

1. Create a `<name>`.yml file in the new directory. Then **name** must be equal to the `name` attribute from your `metadata.yml` file
2. Fill the new file with the topic content. You can follow this [example](https://github.com/patternfly/patternfly-quickstarts/tree/main/packages/module#example-help-topic-in-yaml-with-markdown-support-for-content-and-links).

### Open a PR

1. Open a PR against the [GH repository](https://github.com/RedHatInsights/quickstarts).
2. Add @Hyperkid123 or @ryelo in the PR description

## Querying help topics

This metadata file will be used to describe the examples

```yml
kind: HelpTopic
name: poc-topic
tags:
  - kind: bundle
    value: settings
  - kind: bundle
    value: insights
  - kind: application
    value: new-application
```

### Query help topic by name: `/api/v1/topics/{name}`

You can query a specific topic by name

```
/api/v1/topics/poc-topic
```

### Query help topics for specific bundle: `/api/v1/topics?bundle={bundle}`

If you want to query a topic for the `settings bundle` (meaning applications under `console.redhat.com/settings`), you can call the following endpoint:

```
/api/v1/topics?bundle=settings
``` 

### Query help topics for multiple bundles: `/api/v1/topics?bundle[]={bundleone}&bundle[]={bundletwo}`

You can also request help topics for more bundles
```
/api/v1/topics?bundle[]=settings&bundle[]=insights
```

### Query help topics for a specific application `/api/v1/topics?application={appname}`

If we want to get topics for the `new-application` we can use the following query

```
/api/v1/topics?application=new-application
```

You can also query for multiple applications topics

`/api/v1/topics?application[]={appnameone}&application[]={appnametwo}`

### Query by multiple tags

You can also combine the tags:
```
/api/v1/topics?application[]={appnameone}&application[]={appnametwo}&bundle={bundlename}
```
