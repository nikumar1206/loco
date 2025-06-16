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
