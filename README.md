# check-consul-members

Check if every EC2 instance joins the Consul cluster.

## Synopsis
```sh
check-consul-members --ec2-tag <ec2-tag-key> --ec2-value <ec2-tag-value> --consul-service <consul-service> [--consul-tag <consul-service-tag>]
```

## Description
Checks if every EC2 instance tagged with the key `<ec2-tag>` and the value `<ec2-tag-value>` joins the Consul cluster and have a service runnign with the name `<consul-serviec>` (and the tag `<consul-service-tag>`).

## Options
- `--ec2-tag <ec2-tag-key>`
- `--ec2-value <ec2-tag-value>`
- `--consul-service <consul-service>`
- `--consul-tag <consul-service-tag>` (optional)

An access key ID and a secret access key shall be provided in the environment as `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` resp. These values may be specified in the standard `~/.aws/credentials` file. You also have to provide `AWS_REGION` to specify the AWS region to which the EC2 instances belong.

## Example
Checks all the EC2 instances tagged with `Role=www` join the Consul cluster and running a service `rails`.

```sh
export AWS_REGION=ap-northeast-1
export AWS_ACCESS_KEY_ID=AKIAxxxxxxxxxxxxxxxx
export AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
check-consul-members --ec2-tag Role --ec2-value www --consul-service rails
```
