# check-consul-member

Checks if every instance under a load balancer joins a Consul cluster.

## Synopsis
```sh
check-consul-member --elb api
```

## Arguments
`--elb LB_NAME` specifies the name of an Elastic Load Balancer.

An access key ID and a secret access key shall be provided in the environment as `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` resp. You also have to provide `AWS_REGION` to specify the AWS region to which the load balancer belongs.

## Description

TODO
