#### Things to look into

- we gotta figure out how logging will look like and how users will get their logs
- what other sidecars may we need to deploy directly onto the user service?
- what will we need for monitoring purposes?
- related: abuse prevention?

- i think for monitoring and logging, we should start with our own container, aka loco-api
  and then apply similar logic for handling the user side of things

- tracing will be far into the future. but as much as possible we stick with open standards via prometheus/grafana/opentelem

- likely will need a ServiceAccount for loco-api since it interacts with the rest of the cluster
- add label onto gateway and namespace

- privileged: true, runAsNonRoot: true must be set when pulling in a user's docker image
- improve dashboards (later)
  - remove eg-addons, switch to the kube-prometheus-stack, and manually configure ServiceMonitor for Envoy
  - setup with non empty dir

#### notes ported from readme

- dont deploy the image if docker scout or vulnerabilities are detected
- make sure all of the TOML makes sense and is 100% validated against.
- docker layers should be cached by project or something. maybe docker will automatically do this
- Docker building must follow .Dockerignore or .gitignore
- maybe use gitlabs free account for Docker image hosting
  - only hold maybe the last 2 images per project, for rollback purposes
  - switch to Harbor eventually to overcome these limits
- what do i need to do to support podman/oci-based images
- implement lifecycle policy on registry? maybe 6 mths to start
- think more about security. gotta make sure other users cannot pull down other images. Images must be prefixed or something man and have some sort of random hash to avoid collisions.
- we have read/write on registry. should just be write tbh.
- use terraform to create the necessary kubernetes cluster and related resources
- need to restrict creation on certain namespaces
- with current github action setup, we get errors for already exists, maybe we should ignore those somehow

#### Things to Think About

- Simple, but super effective abusive stopper.
- Only whitelisted repos should be buildable, by certain users.
- Users should only be able to access logs for their projects. Nothing more, nothing less
- All unhappy paths are clearly reflected back to user, with obvious next steps on how to fix.
- Need to do things the kubernetes way with stuff like RBAC and whatnot.
- whatever we do, it needs to be extendible to the following deployments:
  - UI, cache (redis), database, blob
  - this covers like 90% of all possible deployments.

#### high prio

- i think we need to build out the database structure
  - 5 layer concept? teams, users, apps, deployments, events thats it
- oauth with github/google. we will not be managing that

#### medium prio

- actually reading the loco.toml file to determine
- enable securely passing env variables to

#### low prio

- so many nice things tbh

for cli

- potential cmds
  - docs, variables, redeploy, login, account
- brew package for auto sending completions as well
- deploy tokens from registry should not last more than 5 mins
- same thing for registry read tokens, 10 mins max for really large docker images
- need a max size for docker images since we are severly limited on our cluster :)
- multi port?
- currently each project is only subdomain driven, but we need to think of deploying on separate path prefixes as well. something like ui is just hosted on / and backend is on /api. thats a very common pattern

- lets create the mvp architecture diagram. that needs to be done.
- nicety: not need the user to provide the entire loco.toml, we should set a couple sensible defaults;
- namespace gen needs to be some sort of user-app hash to avoid conflict
