# ec2-local-healthchecker

Runs as a daemon on an ec2 instance, executing a set of http or tcp checks periodically.
If there have been a consecutive number of failures, of any check, it sets the instance as unhealthy in the asg it is attached to.

The configuration of the check is defined in a yaml file in `/etc/sysconfig/ec2-local-healthchecker.yml`

This configuration looks like the following:

```
graceperiod: 10s
frequency: 10s
checks:
  memcached:
    type: tcp
    timeout: 1s
    endpoint: localhost:11211
    threshold: 4 
    frequency: 10s
  nginx:
    type: http
    timeout: 1s
    endpoint: http://localhost:80/ping.html
    threashold: 10
    frequency: 1s
```

The above specifies, that:

- The healthcheck will not start until 10seconds after the daemon as started
- We will check the status of each health check every 10 seconds, and determine if we need to terminate the instance
- There are 2 healthchecks running concurrently (nginx and memcached), each with different polling rates


----

# Building AMZ binary

Build the Image
```
docker build --no-cache -t golang .
```

Build the binary
```
docker run --name bob --rm -it -v $(pwd):/go/src/github.com/tootedom/ec2-local-healthchecker golang:latest /bin/bash -c "cd /go/src/github.com/tootedom/ec2-local-healthchecker/ && go build -o ec2-local-healthchecker-amd64"
```

