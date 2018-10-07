# CLI metrics

[![GoDoc](https://godoc.org/github.com/henvic/climetrics?status.svg)](https://godoc.org/github.com/henvic/climetrics) [![Build Status](https://travis-ci.org/henvic/climetrics.svg?branch=master)](https://travis-ci.org/henvic/climetrics) [![Coverage Status](https://coveralls.io/repos/henvic/climetrics/badge.svg)](https://coveralls.io/r/henvic/climetrics) [![codebeat badge](https://codebeat.co/badges/0f69eea8-4ac2-40f5-9848-e931b5faf186)](https://codebeat.co/projects/github-com-henvic-climetrics-master) [![Go Report Card](https://goreportcard.com/badge/github.com/henvic/climetrics)](https://goreportcard.com/report/github.com/henvic/climetrics)

CLI metrics is a software used for gathering diagnostics and metrics data for the [WeDeploy](https://www.wedeploy.com/) Command-Line Interface tool.

## Dependencies

* Go ≥ 1.11 to generate the server binary.
* [PostgreSQL](https://www.postgresql.org) 10 or greater.

## Database
Create a database named `climetrics` and import the schema to it with:

```bash
createdb climetrics
psql climetrics < file.sql
```

To generate a new schema you can use:

```bash
pg_dump -U username climetrics --schema-only --no-owner > climetrics.pgsql
```

## Commands

* **cmd/adduser** can be used to add users to the database
* **cmd/fixgeoip** should be used regularly to fix any missing geolocation information (i.e., crontab)
* **cmd/password** can be used to hash passwords using bcrypt

## Running

After creating the database as indicated above, you need to configure it with the `-dsn` flag. Be aware that this also applies for the **fixgeoip** program when using it on a crontab schedule.

This system was designed to work behind a reverse proxy (such as [nginx](https://nginx.com)), this is why it doesn't handle HTTPS termination. Make sure to [forward IP addresses](https://www.nginx.com/resources/wiki/start/topics/examples/forwarded/), in any case.

A simple reverse proxy configuration for nginx is:

```
# proxy configuration file /etc/nginx/proxy.conf
proxy_set_header Host $http_host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;

# climetrics virtual host configuration file on sites directory
server {
        listen   *:443;
        server_name  climetrics.example.com;

        access_log  /var/log/nginx/access.log;

        location / {
                proxy_pass      http://127.0.0.1:8080/;
                include         /etc/nginx/proxy.conf;
        }
}
```

The Request IP is calculated assuming the first public IP from the list considering immediate Remote Address, X-Real-IP, and X-Forwarded-For list.

It is recommended to use the `-expose-debug` flag to expose debugging data (from packages expvar and pprof) on HTTP local port 8081 (including on production environments), allowing you to run commands such as:

```
$ go tool pprof -web http://localhost:8081/debug/pprof/heap
$ curl http://localhost:8081/debug/vars
```

Use environment variable DEBUG=true to set the log level to debug and expose the debug entrypoints described above.

## Contributing
You can get the latest CLI source code with `go get -u github.com/henvic/climetrics`

The following commands are available and require no arguments:

* **make test**: run tests

In lieu of a formal style guide, take care to maintain the existing coding style. Add unit tests for any new or changed functionality. Integration tests should be written as well.

## Committing and pushing changes
The master branch of this repository on GitHub is protected:
* force-push is disabled
* tests MUST pass on Travis before merging changes to master
* branches MUST be up to date with master before merging

Keep your commits neat and [well documented](https://wiki.openstack.org/wiki/GitCommitMessages). Try to always rebase your changes before publishing them.

## Maintaining code quality
[goreportcard](https://goreportcard.com/report/github.com/henvic/climetrics) can be used online or locally to detect defects and static analysis results from tools with a great overview.

Using go test and go cover are essential to make sure your code is covered with unit tests.

Always run `make test` before submitting changes.
