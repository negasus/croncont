# Croncont 

Simple docker image for run cron task with call remote URL.

## Docker image

```
```

## Config (env variables)

- `CRON_URL` request url, default `http://localhost`
- `CRON_METHOD` request http method, default `POST`
- `CRON_BODY` request body, default empty
- `CRON_HEADERS` request headers, format: `Name=Value|Name=Value`, default empty
- `CRON_TIMEOUT` request timeout, ms. default `3000`
- `CRON_EXPECTEDSTATUS` expected response status. If 0 - not checked. If not equals, log message will be sent
- `CRON_SPEC` cron spec with seconds, default `0 * * * * *`
- `CRON_VERBOSE` send log message on every call, default `false`
- `CRON_LISTEN` listen adderss for health checker (`/`) and metrics (`/metrics`) routes. Not use if empty, default `0.0.0.0:8001`

## Usage