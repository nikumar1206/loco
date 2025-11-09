- Make loco multi-tenant, multi-app setup thats backed by a database.
  - currently everything is stored in k8s; this needs to be moved to a Postgres or similar.
  - db will source of truth, in case k8s goes down, loco should be rebuildable through snapshots of the cluster?
- Metrics/Logging/Tracing
  - use [OpenObserve](https://openobserve.ai/) as the combined solution
  - for Loco-API itself, we will use auto-instrumentation.
  - All logs/tracing/metrics must include tenant and app-id combination
  - can potentially create dashboards dynamically, or atleast pull the data down.
  - deploy a self-hosted instance on monitoring.loco.deploy-app.com
- Profiles

  - introduce multi-profile deployments to handle dev, uat, prod deployments.
  - `loco deploy --profile=dev`
    - maybe profiles are just staging and production
  - profiles should be specifiable in loco.toml.

    ```toml
    [Profile.dev]
      CPU = "100m"
      Memory = "128Mi"
      BaseDomain = "dev.deploy-app.com"

    [Profile.prod]
      CPU = "500m"
      Memory = "1Gi"
      Replicas.Max = 5
    ```

- support for GRPC routes
  - there is potential this can only be tested
- support for grpc/http/cmd-baed health checks

- Logs Command

  - Support for following and tailing x logs.
  - CLI table should support a simple freeze as well.
  - -o should take json or table.

- Deploy Command

  - take a token non-interactively via std in, maybe with simple output as well. `loco deploy --non-interactive --token {GH-TOKEN}`
  - take an image id, so that loco doesnt build the image and we get to skip some steps.
  - introduce a wait command, that waits for everything to finish
  - users should always get back some sort of deployment-id or eventid that can be presented for debugging purposes.

- Scanning Docker Images; we have a TDD for this

- Pre-deployment loco needs to check if we can sustain the requested deployment (atleast 2x the requested resources to be safe.)

  - not sure how to do this.

- New commands:

  - loco web : opens loco website in browser.
    - --dashboard, --traces? --logs, -- docs, --account
  - loco env : syncs new env variables, without redeploying
  - loco map
    - generates a map of user's deployments to loco or an app's compute.
    - --appName or like tenantid name. this is just a nice to have.
    - project name based?

- seem to be some issues ensuring im grabbing the latest version of my own local packages.
- Sometimes docker is sleeping; we need to give better errors, and maybe tell users to just specify --image if stuff keeps going wrong.
- can we check if docker is sleeping before trying to build the image?
- are we validating that subdomains have not been taken ?
- similarly for grpc, we need to validate GRPC routes have not been taken

  - should be done on a per domain basis.
  - wish we had a database!

- introduce a project-id. Project id will be used to map loco.toml's together.
- on update, we should update the service as well; my ports were different, but it didnt get applied.
- In-Cluster Communication

  - Lets inject service URL via env variables: LOCO\_<APP_NAME>\_URL . (multiple of these, scoped to the project)
  - other env variables we can add:
    - LOCO_APP_NAME
    - LOCO_APP_VERSION ~ tied to git commit?
    - LOCO_PROFILE
    - LOCO_DEPLOYMENT_ID ~ loco's deployment id (once we have a DB and everything.)
    - LOCO_VERSION ~ loco version ? idk if we need to provide this
    - LOCO_TRACING_ENDPOINT ~ this is the openobserve endpoint to submit traces to
    - LOCO_METRICS_ENDPOINT ~ this is where loco will be scraping metrics from

- Resurrector
  - deployed separately from the cluster
  - continously monitors and pings cluster health status
  - if not healthy, try to diagnose? and rebuild whats broken?
  - needs to be done on a per provider basis

---

## Medium Priority

- Set registry lifecycle policy (start with 6 months)
- Require image prefixing with random hash
- Only allow registry writes from our infra, not reads
- Store only last 2 images per project
- Support Podman / OCI-based images
- Set max Docker image size (cluster limited)

---

## Low Priority

- Cleanup
  - that random config file that has too much? makes no sense
  - timestamps should all be same fmt
- Evaluate ArgoCD and others for better CD of kubernetes resources
- Gitlab Container Registry Token is only procured on loco deploy; should be re-procured in case node expires, ...
- Better handling of secrets related to Loco.
  - Need to be autorotated; stored in some secrets vault.
- Better handling of app secrets
- Review API contracts to make sure they make sense

- Docker image enhancements?

  - Consider a private registry that has tag-prefix/name-prefix based access-controls.
  - OSS solutions like Harbor / Quay exist.
  - come with different scanners like trivvy and multi tenant.
  - can look towards them, or for now just have a single set of deps
  - civo offers this
  - Potentially add artifact attestations to images

- Secrets
  - Kubernetes configmap of secrets needs to be created separately
  - Create RBAC to restrict secret visibility for env vars

---

Eventually...

- Support and test different deployment types: UI, cache (Redis), DB, Blob
- Respect/Allow specifying .dockerignore files / .gitignore files when building container images.
- Secrets integration

  - Secrets need to be pulled and injected
  - but user can also do this in their own container, just access aws ssm no?
  - but how are they gonna get the aws secret key and whatnot?

- how are we handling security patches?

  - depends on provider config, they will be auto managed for us if using things like fargate, otherwise our issue.
  - might need to do some sort of blue-green deployment for kubernetes.
  - what about bumping stuff like envoy gateway and things like that.
  - lets make a map of all the different projects loco is dependant on and how we can keep them updated.

- also gitlab fetch token is only valid at deployment. what if new node comes in and needs to pull down image, it cannot since gitlab token expires in like 5 mins.

may be nice to have some sort of secrets integration? like pull ur aws ssm, vault, secrets,
too much for MVP

- Next Steps:

  - Respect more of the loco.toml
    - allow setting GRPCServices and if provided, create a GRPC route, maybe we need a GRPCport?
  - loco init is chunky, introduce minimal vs full flag.

  - start design on profiles?
  - review API design; i think we are doing some funky things

---

we finally have basic logs/metrics popping up.

- organization/different streams, segregated dashboards for like workspace? project scope
- customized dashboards one for each service inside the project,
- maybe even eventually add alerts to an email.
- loco root password will need to be auto-rotated.
- switch to using grpc instead of http?
- tracing will be final step, if we even implement that piece. railway/heroku dont support tracing

sleep mode; if app not used in last 7 days or something. deployment is removed; can be recreated on request.

- who sleeps the app/ who rebuilds the app?
- actually maybe u point to actually the loco backend, and path rewrite to /revive-app?app-name=foobar123&og_url=foobar123.loco.deploy-app.com/cheesecake, and this revives app, and then redirects you to the correct domain again
- there is value to having an admin dashboard, for those who are planning to bring your own cloud. but need to figure out keys and roles and whatnot.

- some sort of env for configuring deployment behavior:

  - max_concurrent_app_deployments => 3

- resource management needs to be evaluated. how many resources are we using ? what are we wasting ?

- wondering if there is value in tests where we actually literally spin up a docker container and we start running stuff on it. like literally use minikube and firing away at tests, atleast i think thats the most accurate way to test the deployment piece.
- improve ci/cd pipelines for testing purposes

- remove host from persistent flag.
- update system design diagram to represent observability.
- deploy needs to do a diff of the previous deployment done on loco, vs the incoming, and only update the resources that need changing.

  - can likely do this client side as well

- should run cleanup resources if deployment fails anywhere.

  - simple implementation is done.

- need to configure a decent HPA for the nodes themselves on kubernetes.

- does loco need to store the local path the user deployed their app from?
  - maybe we need to warn them if the provided project path has changed to ensure they arent messing things up and referencing the wrong project?
  - store mapping under $HOME/.loco?
- if we wanna continue with some gitlab container registry, we can use the container registry

- Secrets we need to manage

  - Terraform Cloud secret
  - Gitlab secret
  - Digital Ocean / Cloud provder secret for provisioning.
  - GH Oauth Client Secret (to identify)
  - Cloudflare API token so that cert-manager can issue certsa and auto-renew

- deployment scripts need to actually have some tests lol
- generic webhook for notifying admins on failures.

- restrict network policies.
- oh man do we need to do an MCP server.

- otel logs, if structured, we should parse out the severity (level)

Clickhouse logs issues:

- clickhouse potential sql injection with this limits + query
- queries are also relatively slow; we should index on the app-id/wkspce-id
  - this will require custom schema definition, and some manual sql work.
- introduce a way to ignore some substrings
- introduce ascending/descending timestamp order
- arbitrary filters can be added no way?
- lol is stuff being ttl'ed?
- move clickhouse monitoring to admin dashboard only
- see how to show all the fields and not just the body?
- validate clickhousedb resources we gave it. 750mb might not be enuf?

- need a full load test on loco and its services.
  - default envoy doesnt have any scaling attached?
- on successful routing, we should add the loco-tenant-id, we will be able to pull it later in otel for dashboarding? not 100% what that looks like.

- loco admin dashboard

  - see how many apps are deployed on loco
  - how many requests are currently being handled.

- theres actually tons of metrics being exported into clickhouse currently
  - we should spend some time and optimize whats being sent.
  - we should do this when we revisit the otel table structures
