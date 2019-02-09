# shove

This is a CLI utility (and Go library) that reacts to GitHub webhooks in various ways.
The name reminds you to keep this ready for when `git push` comes to _shove_. :)

## Usage as an application

Build with `make && make install`. Only [Go](https://golang.org) is required as a build-time dependency.

Invoke `shove` without any arguments, with the following environment variables set:

- `SHOVE_PORT` defines on which port shove will listen for HTTP requests.
- `SHOVE_SECRET` contains a secret key which you also need to enter in GitHub's
  webhook UI, so that GitHub can sign webhook events.
- `SHOVE_CONFIG` contains the path to a configuration file. If not,
  `./shove.yaml` is used instead.

The configuration file uses YAML syntax and looks like this:

```yaml
actions:
  - name: react to push on github.com/foo/bar and github.com/foo/baz
    run:
      command: [ /bin/echo, "Hello World" ]
    on:
      - events: [ push ]
        repos:  [ foo/bar, foo/baz ]
```

Each action can have multiple triggers (in the `on`) section, matching
different webhook events and repositories.

There is also a pseudo-event `shove-startup` that triggers once at application
startup. A trigger matching `shove-startup` may not include any repositories.
For example, the following config pulls the content for a website from a GitHub
repo. The trigger on `shove-startup` ensures that the action gets executed
immediately to ensure the availability of the checked-out files.

```yaml
actions:
  - name: checkout github.com/foo/website
    on:
      - events: [ shove-startup ]
      - events: [ push ]
        repos:  [ foo/website ]
    run:
      command:
        - /bin/sh
        - -c
        - |
          set -euo pipefail
          if [ -d /var/lib/webroot/example.com ]; then
            git -C /var/lib/webroot/example.com pull
          else
            git clone https://github.com/foo/website /var/lib/webroot/example.com
          fi
```

While `actions[].run.command` is executed, depending on the type of event, several environment variables are available which contain the event payload.

## Supported events

### `push`

This event occurs ewhenever a branch or tag gets pushed to a repository.

**Environment variables:**

- `SHOVE_VAR_REF`: The ref that was pushed to, e.g. `refs/heads/master` or `refs/tags/v2.1.2`.
- `SHOVE_VAR_BRANCH`: The name of the branch that was pushed, if applicable (e.g. `master` if the ref was `refs/heads/master`). If something other than a branch (e.g. a tag) was pushed, this variable is empty.
- `SHOVE_VAR_COMMIT`: The new head commit that the ref now points to.
- `SHOVE_VAR_REPO_NAME`: The name of the repository, e.g. `bar` for `github.com/foo/bar`.
- `SHOVE_VAR_REPO_OWNER`: The name of the repository owner, e.g. `foo` for `github.com/foo/bar`.
- `SHOVE_PAYLOAD`: The entire event payload sent by the server. This is a JSON document, so it can be inspected e.g. with [`jq(1)`](https://stedolan.github.io/jq/) to find any attributes that have not been provided in their own environment variables.

### `shove-startup`

This pseudo-event occurs once when Shove starts up, before it starts listening on the `SHOVE_PORT`.

**Environment variables:** None.
