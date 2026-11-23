# Atlassian External Hooks 
A Bitbucket Server app that enables you to add repo hooks in non-Java languages.

If you're looking for the ready-to-use app, it is available on the [Atlassian Marketplace](https://marketplace.atlassian.com/1211631).

[Docs](https://external-hooks.reconquest.io/)

## Table of Contents 
* [Getting Started](#getting-started)
* [Features](#features)
* [Support](#support)
* [License](#license)

## Getting Started
If you'd like to use this app with your Bitbucket Server or Data Center installation, you'll need to install this [app](https://marketplace.atlassian.com/1211631) and then [configure](https://external-hooks.reconquest.io/docs/how_to_configure_external_hooks/) it.

For further customizations of this app, you can fork it.

## Features
Supports the following hooks:
* Async Post Receive
* Pre-Receive
* Merge Check

## Support

Please [open an issue](https://github.com/reconquest/atlassian-external-hooks/issues/new) for support.

You can always join our Slack community and drop us a line there: [slack.reconquest.io](https://slack.reconquest.io/)

### Bitbucket 9 and 10 builds

`task atlas:package` builds only the current Bitbucket 10 add-on in `target/`.
For Bitbucket 9, the test harness downloads release 13.1.3 from Atlassian
Marketplace and caches it in `builds/`.
`./run_tests` exercises the clustered hook suites on 9.6.5, 10.0.0, 10.2.8 and 10.5.2.
The test harness permits Basic authentication and unsigned local builds on its
Bitbucket 10 containers only; these are not production configuration requirements.
