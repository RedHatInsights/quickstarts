# Creating a quickstart

1. Create new directory in `docs/quickstarts/`
2. Create `metadata.yaml`, `README.adoc` and `quickstarts.yaml`

## metadata.yml

Metadata file is used to define additional API information about the quickstart without changing the quickstart content itself.

Example:
```yaml
kind: QuickStarts
name: poc-quickstart
tags:
 - kind: bundle
   value: insights
 - kind: bundle
   value: settings
 - kind: application
   value: sources
 
```

### `kind`

Must be QuickStarts, identifies the content type.

### `name`

Content will be stored and referenced by this string

### `tags`

Tags are used for filtering. Allowed quickstarts tags kinds are bundle and application. Tags should be used to associate content with specific application(s) and/or bundle(s).

## README.adoc

Stores the quick start content. This is the primary file that you will work with. It consists of a number of procedures, one for each task in the quick start. You can think of the README as an assembly, and each procedure as a module.

## quickstart.yaml

An OpenShift CR that defines the metadata and structure of the quick start. It references the procedures in README.adoc.

Stores the quick start content in a specific format. This is used by the quickstarts UI module to render the content in HCC.

All quick starts must have an apiVersion: console.openshift.io/v1, and a kind: QuickStarts as well as an associate array metadata with a member with key name, which must be given the identifier as a value:

```yaml
apiVersion: console.openshift.io/v1
kind: QuickStarts
metadata:
  name: <identifier>
```
The spec associative array defines the quick start content. Start by defining the content type of the quickstart (Quick start / Documentation), the version of the quick start, the URL of an icon to use, and how long the quick start should take to complete.

```yaml
spec:
  version: <quick start version>
  type:
    text: Quick start // or Documentation if it has an external link
    color: green // orange for Documentation
  icon: <icon url>
  durationMinutes: <duration>
```

The displayName of the quick start is used both in the catalog and as the heading for the quick start drawer.

```yaml
spec:
  ...
  displayName: !snippet/title README.adoc#<id>
  ...
```

The !<tag name> syntax represents a custom data type in YAML. When the quickstart.yaml document is deserialized by the YAML parser, the quick start renderer is able to inject content. The quickstart.yaml parser makes use of custom data types to inject content from an AsciiDoc file into the quick start. This allows us to better comply with the DRY principle.

The prerequisites of the quick start are rendered in the quick start catalog.

```yaml
  prerequisites:
    - Requirement 1
    - Requirement 2
```
  
