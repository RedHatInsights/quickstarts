# Creating quick starts (Learning resources) for the Hybrid Cloud Console

These instructions explain how to create content for Learning resources pages in the [Red Hat Hybrid Cloud Console](https://console.redhat.com) using the PatternFly [quick starts](https://www.patternfly.org/extensions/quick-starts/) extension. 

A **Learning resources** page ([example](https://console.redhat.com/insights/learning-resources)) on the Hybrid Cloud Console is a central location that contains links to different content types related to a collection of console services (or _bundle_) such as product documentation, interactive quick starts, learning paths, and more. These links help users find the information they need from within the console user interface, in the most suitable format. Each resource is represented by a tile on a Learning resources page. The source files are created in YAML.

You can use the PatternFly quick starts extension in this repository to create:

- _Learning resources tiles_ in the Hybrid Cloud Console that link to an external resource, such as documentation or a learning path. The resource opens in a new browser tab. 
- _Interactive quick starts_, which are step-by-step instructions and tasks presented in a side panel embedded in the Hybrid Cloud Console UI. Interactive quick starts remain open on the right side of the console window while the user completes tasks.

## Creating a Learning resources tile on the Hybrid Cloud Console

A Learning resources _tile_ provides a link to educational content published outside of the Hybrid Cloud Console, such as product documentation (full guides or individual topics), learning paths, and more, to help users easily find the information they need from within the console UI. 

Each tile on the Learning resources page in your bundle contains:

- A title for the resource - in sentence case. For documentation items, you can shorten the title from the full guide title on the Customer Portal.
- A short description of the resource - a complete sentence of up to 115 characters. 
- A link to the content.

## Creating interactive quick starts for the Hybrid Cloud Console

An _interactive quick start_ is a set of step-by-step instructions and tasks presented in a side panel embedded within a product’s UI. Quick starts can help users get started with a product by providing installation and setup guidance. Quick starts also allow users to quickly complete a task without the need to refer to external documentation.

A quick start has multiple tasks, each task with multiple steps.

Quick starts should be brief and simple to follow. Red Hat recommends quick starts take no longer than 10 minutes for a user to complete, and have a maximum of 5 tasks (2-7 steps per task is a good length). You can find more detailed guidelines below under _Best practices for creating quick starts_. 

**IMPORTANT**:
The quick starts content in this repository is not validated by the content team. We are working on defining a formal process. Please be aware that you might be required to update or change the content.

You can read more about the PatternFly quickstarts UI extension in the official [PatternFly documentation](https://github.com/patternfly/patternfly-quickstarts/tree/main/packages/module#quick-starts-format).

## Preview tool

You can use this simple [preview tool](https://quickstarts-content-preview.surge.sh/) to view your content. To preview your content, copy and paste the contents of your YAML file in the window and click **Add quickstart to list**. Be aware the tool is not official and was hastily put together. If it crashes, please refresh the page.


## Summary of steps
This is an overview of the steps you will need to complete to publish an interactive quick start or Learning resource tile in the Hybrid Cloud Console.

1. Write your draft quick start content, and get review and approval from product stakeholders.

2. When the content is ready, create your YAML quick start files in a new directory in the `docs` directory of [**Red Hat Insights quickstarts**](https://github.com/RedHatInsights/quickstarts/tree/main/docs/quickstarts), following the detailed instructions in this `README.md` file. 

3. Get your quick start or Learning resource tile added to the [Hybrid Cloud Console](https://console.redhat.com):
 
    a. Create a request in [the RHCLOUD Jira project](https://issues.redhat.com/projects/RHCLOUD/issues/) for engineering to merge the content. Add the `platform-experience-services` label to your Jira.
    
    b. Create a pull request (PR) to the `main` branch in the [**Red HatInsights/quickstarts**](https://github.com/RedHatInsights/quickstarts/) repository.
    
4. The engineering team then merges the pull request to the Hybrid Cloud Console. The content will appear first on the [stage](https://console.stage.redhat.com/) of the Hybrid Cloud Console, and later in [production](https://console.redhat.com/).

When these steps are complete, you will be able to view your quick start or Learning resource tile in the [Hybrid Cloud Console](https://console.redhat.com/).

## Detailed steps

### Initial setup 
You will need to complete these steps the first time you are contributing to the `quickstarts` repository. If you have already forked and cloned the `quickstarts` repository, continue to the next section, _Create a working branch and make your docs updates_.

**Prerequisites**
- You have a GitHub account (https://github.com) 
- You have configured your SSH keys on your system following the instructions here: https://help.github.com/articles/connecting-to-github-with-ssh/

1. Create a fork of the `Red Hat Insights quick starts` repository:

    a. Navigate to https://github.com/RedHatInsights/quickstarts.

    b. Click **Fork** and follow the prompts to add the fork to your namespace.

    c. Click **Create fork** to confirm.

2. Clone `https://github.com/<your-namespace>/quickstarts.git` in your terminal. For example:
    ```
    git clone git@github.com:nancydrew/quickstarts.git
    ```
3. Change to your local `quickstarts` directory: 
    ```
    cd quickstarts
    ```
4. Add this repository as remote:
    ```
    git remote add -f upstream git@github.com:RedHatInsights/quickstarts.git
    ```


### Create a working branch and make your docs updates 
If you have created quick starts in the Hybrid Cloud Console before, start here:
1. Fetch the latest content from the upstream repository to make sure you are working with the latest code base:
    ```
    git fetch upstream
    ```
2. Add the changes to your copy of main branch (in your fork):
    ```
    git rebase upstream/main
    ```
3. Refresh your main branch (this step is optional but ensures `main` is fully up to date):
    ```
    git push origin main
    ```
4. Check out a topic branch from the upstream main branch. If the branch does not exist, this command will create one:
    ```
    git checkout -b TOPIC_BRANCH_NAME upstream/main
    ```
    **IMPORTANT**: Don’t forget to add `upstream/main` to the end of this command, otherwise you might not be working with the latest source from the upstream `main` branch.

5. Create the following quick start files and format your content into YAML:

    a. Create a new directory with an identifiable name in `docs/quickstarts/<name>` in [**this repository**](https://github.com/RedHatInsights/quickstarts/tree/main/docs/quickstarts) to contain your quick start files.

    b. In the new directory, create a `metadata.yml` file. Ensure you have a blank line at the end of the file:
  
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

    c. In the new directory, create a `<name>.yml` file. The *name* must match the `name` attribute from your `metadata.yml` file.
    
    d. Add your draft quickstart content to the new file. You can follow this [template](https://github.com/patternfly/patternfly-quickstarts/blob/main/packages/dev/src/quickstarts-data/yaml/template.yaml) and find more Markdown snippets in the _Useful Markdown snippets_ section of these instructions.

    NOTE: If you are creating a Learning resources tile only (such as a link to external documentation), refer to [this example](https://github.com/RedHatInsights/quickstarts/tree/main/docs/quickstarts/ansible-getting-started-hub) to understand the information required. Learning resources tiles require minimal information in the YAML files, compared to interactive quick starts. 

    e. Preview and validate the YAML content by copying and pasting your YAML into the [preview tool](https://quickstarts-content-preview.surge.sh/). Make changes as needed until you are ready to push your files to the remote branch for review.

6. Push your files to the remote branch for review by stakeholders as needed.

    a. Add your files to tracked changes: 

    ```
    git add <files>
    ```

    b. Make sure you have no uncommitted changes on your issue branch:       

    ```
    git status
    ``` 
    c. Commit your quick start files:

    ```
    git commit -am "descriptive commit message"
    ```

    d. Push your quick start files to the remote branch:

    ```
    git push origin <branch> 
    ```
9. Open a pull request (PR) in the [GitHub repository](https://github.com/RedHatInsights/quickstarts) containing your update. Get your quick start reviewed by stakeholders and peer reviewers, as needed. Once you and any reviewers are happy with the content, work with engineering to add the content to the Hybrid Cloud Console.


### Adding your quick start (or Learning resource tile) to the Hybrid Cloud Console

Open a Jira for the engineering team to add the quick start to the Hybrid Cloud Console source code.

1. Create a Jira for the platform development team in the [RHCLOUD Jira project](https://issues.redhat.com/projects/RHCLOUD/issues/RHCLOUD-15910?filter=allopenissues). Add the **platform-experience-services** label to the issue.
2. In the PR you opened for the quick start, add the developer contacts `@Hyperkid123` or `@ryelo` in the PR description, and add the Jira link in a comment.

### Engineering tasks

From here, it is up to the engineering team to merge the PR to the Hybrid Cloud Console. If you are writing a quickstart, you can continue to the next section, _Close the loop_.

Your quick start will show up in the Hybrid Cloud Console [stage environment](https://console.stage.redhat.com/) first - 
To publish live on the Hybrid Cloud Console [production environment](https://console.redhat.com/), engineering must create an `app-sre` pull request.

#### Query quickstarts for a specific application `/api/v1/quickstarts?application={appname}`

To get quickstarts for the `new-application`, use the following query:

```
/api/v1/quickstarts?application=new-application
```

You can also query for multiple applications quickstarts:

`/api/v1/quickstarts?application[]={appnameone}&application[]={appnametwo}`

#### Query by multiple tags

You can also combine the tags:
```
/api/v1/quickstarts?application[]={appnameone}&application[]={appnametwo}&bundle={bundlename}
```

### Close the loop

When your quick start is live in the Hybrid Cloud Console, close your original docs Jira and update any stakeholders about the completed work.

## Best practices for creating quick starts

* Length of quick starts:
  * 3-5 minutes in duration (maximum 10 minutes)
  * Maximum 5 tasks
  * 2-7 steps per task (maximum 10 steps)
* Each step should have a “Check your work” section.

For more best practices and tips, see: 

* [Creating quick start tutorials](https://docs.openshift.com/container-platform/4.14/web_console/creating-quick-start-tutorials.html) in the OpenShift documentation 
* [Design guidelines for quick starts](https://www.patternfly.org/extensions/quick-starts/design-guidelines/) in the PatternFly documentation

## Assigning your quick start to a console location using the `bundle` tag 

The `bundle` tag in the quick start's `metadata.yml` file tells the console on which **Learning Resources** page to show the quick start or tile. A bundle is a console-internal term that refers to a collection of services.

You can use more than one `bundle` tag to show the quick start in multiple locations in the console. For example, this would show the quick start on **Learning Resources** pages for both **Insights** and **Settings**:
  
    ```yml
    ...
    tags: # If you want to use more granular filtering add tags to the quickstart
      - kind: bundle # use bundle tag for a topic to be accessed from a whole bundle eg. console.redhat.com/insights
        value: insights
      - kind: bundle
        value: settings
    ...
    ```

See the below list for the value tags to use for each bundle:

| Bundle  |  value tag |
|---|---|
|  Application Services | application-services  |
|  OpenShift |  openshift |
|  Ansible Automation Platform | ansible  |
| Red Hat Insights  | insights  |
| Edge management  | edge  |
|  Home |  landing |
|  All Services | allservices  |
| Identity & Access Management  | iam  |
| Internal  |  internal |
|  Settings | settings  |


## Useful Markdown snippets

* A nice Markdown summary from the [Red Hat Customer Portal](https://access.redhat.com/help/markdown)
* [Creating quick start tutorials](https://docs.openshift.com/container-platform/4.15/web_console/creating-quick-start-tutorials.html) in the OpenShift documentation

**Bold**
* Use `**bold text**` for UI labels, buttons, menu names.

**Italics**
* Use `*italicized text*` for variable or replaceable text.

**Images**

There are a few ways to include visual elements in a quick start. 

**IMPORTANT**: Avoid screenshots wherever possible as these require a lot of maintenance, as the user interface can change anytime.

--> **Pointing to a UI element (highlighting)**

1. Find out the ID of the UI element in the console:

    a. In your browser, right-click on the element, then click **Inspect**.

    b. Copy the value of the `id`. For example, if you inspect the gear icon, you see `id=”SettingsMenu”`. 

2. In the quick start YAML, add the text to be highlighted and the ID of the element formatted like `[Quick starts nav item]{{highlight quick starts}}`. For example:

```
 Click the [Settings icon]{{highlight SettingsMenu}} to open **Settings**.
```

--> **Including an inline icon in a step**

Use this method if you can’t point to a specific UI icon in the Hybrid Cloud Console: for example, a repeated icon in a list, in instances where a user’s setup might be customized, or when an icon has no name when you hover over it in the UI (such as the ellipsis icon signifying ‘options’ or ‘more options’).

1. Find the name of the PatternFly icon in [the PatternFly docs](https://www.patternfly.org/design-foundations/icons/#font-awesome-solid-fas-vs-font-awesome-regular-far). All of these icons are included in the PatternFly code so you don’t need to upload an image, you can just point to it with HTML.
2. Use the markup to specify the library the icon comes from (PatternFly, Font Awesome solid, Font Awesome regular) and the icon name. Here are some examples:
    - For PatternFly icons: `<i class="pf-icon [insert-icon-name]"></i>`
    - For Font Awesome solid icons: `<i class="fas [insert-icon-name]"></i>`
    - For Font Awesome regular icons: `<i class="far [insert-icon-name]"></i>`
3. Add the name of the icon in brackets after the icon.
4. Test that the icon renders correctly in the [React preview tool](https://quickstarts-content-preview.surge.sh/).

_Example_

```
 Next to **Vulnerability administrator**, click <i class="fas fa-ellipsis-v"></i> (more options) > **Remove** to revoke this permission from all users in your organization.
```

**Admonitions**

The syntax for rendering admonition blocks (for example, text in a Note or an Important box) to PatternFly React Alerts is:
- Bracketed alert text contents
- The admonition keyword, followed by the alert variant you want
- Variants are: note, tip, important, caution, and warning

_Examples_

```
[This is the note contents]{{admonition note}}

[This is the tip contents]{{admonition tip}}

[This is the important contents]{{admonition important}}

[This is the caution contents]{{admonition caution}}

[This is the warning contents]{{admonition warning}}
```

**Adding extra spaces**

If the content is rendering incorrectly (for example, if spacing doesn't look how you want it in the text around an icon) you can insert extra spaces using the `&nbsp;` character in HTML. You can also add several `&nbsp;` characters in a row if you still need more spacing.

_Example_
* Between words and/or icon markup, add one or more `&nbsp;` characters if extra spaces are needed:
```
Click **Remove** to revoke this&nbsp;&nbsp;&nbsp;permission
```
* This will add 3 spaces to render like this:

  Click **Remove** to revoke this&nbsp;&nbsp;&nbsp;permission


