# Quickstarts

Read more about the quickstarts UI module in the official [documentation](https://github.com/patternfly/patternfly-quickstarts/tree/main/packages/module#quick-starts-format)

## Preview tool

You can use a simple preview tool [here](https://quickstarts-content-preview.surge.sh/). Be aware the tool is not official and was hastily put together. If it crashes please refresh the page.

## Please read

Quickstarts content in this repository is not validated by the content team. We are working on defining a formal process. Please be aware that you might be required to update/change the content.

## Add a new quickstart content

### Create metadata.yml for a new quickstarts content

1. Create new directory in `docs/quickstarts/<name>`
2. In the new directory create a `metadata.yml` file

```yml
kind: QuickStarts # kind must always be "QuickStarts"
name: <name> # this name will be used as an identifier. The file with the content must use the same name `<name>.yml`
tags: # If you want to use more granular filtering add tags to the quickstart
  - kind: bundle # use bundle tag for a topic to be accessed from a whole bundle eg. console.redhat.com/insights
    value: insights
  - kind: bundle
    value: settings
  - kind: application # use application tag for quickstart used by specific application
    value: sources

```

### Create `<name>.yml` file in new directory

1. Create a `<name>`.yml file in the new directory. Then **name** must be equal to the `name` attribute from your `metadata.yml` file
2. Fill the new file with the quickstart content. You can follow this [template](https://github.com/patternfly/patternfly-quickstarts/blob/main/packages/dev/src/quickstarts-data/yaml/template.yaml).

### Open a PR

1. Open a PR against the [GH repository](https://github.com/RedHatInsights/quickstarts).
2. Add @Hyperkid123 or @ryelo in the PR description

## Query quickstarts for a specific application `/api/v1/quickstarts?application={appname}`

If we want to get quickstarts for the `new-application` we can use the following query

```
/api/v1/quickstarts?application=new-application
```

You can also query for multiple applications quickstarts

`/api/v1/quickstarts?application[]={appnameone}&application[]={appnametwo}`

### Query by multiple tags

You can also combine the tags:
```
/api/v1/quickstarts?application[]={appnameone}&application[]={appnametwo}&bundle={bundlename}
```
