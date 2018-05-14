# ec2-local-healthchecker

Runs as a daemon on an ec2 instance, executing a set of http or tcp checks periodically.
If there have been a consecutive number of failures, of any check, it sets the instance as unhealthy in the asg it is attached to.

The configuration of the check is defined in a yaml file in `/etc/sysconfig/ec2-local-healthchecker.yml`

An Example Configuration looks like the following:

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

# Usage

The go binary uses this [daemon library](https://github.com/takama/daemon).  This allows you to "install" the daemon into you environment.
For example:

```
./ec2-local-healthchecker-amd64 -command install
```

On an amazon linux instance this will install the healthchecker as a "upstart" service (note only tried on amz linux 1 - not the new 2 version).

Once installed as a "upstart" service you can start the service with:

```
start ec2-local-healthchecker
```

This reads the configuration at:

- /etc/sysconfig/ec2-local-healthchecker.yml
- Uses the ec2 metadata system to obtain the instance id, and aws region
- Uses the instances iam role to obtain credentials to talk to AWS
-- This means the instance needs `autoscaling:SetInstanceHealth` permissions
- If the healthcheck fails calls autoscaling SetInstanceHealth to mark the instance as unhealthy
- The daemon logs to `/var/log/ec2-local-healthchecker.log` and ``/var/log/ec2-local-healthchecker.err`


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

