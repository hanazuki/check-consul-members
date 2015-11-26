# check-consul-members

Check if every EC2 instance joins the Consul cluster.

## Synopsis
```sh
check-consul-members --ec2-tag <ec2-tag> --consul-tag <consul-tag> --value <tag-value>
```

## Description
Checks if every EC2 instance tagged with the key `<ec2-tag>` and the value `<tag-value>` joins the Consul cluster and have a tag with the key `<consul-tag>` and the value `<tag-value>`.

## Options
- `--ec2-tag <ec2-tag>`
- `--consul-tag <consul-tag>`
- `--value <tag-value>`

An access key ID and a secret access key shall be provided in the environment as `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` resp. You also have to provide `AWS_REGION` to specify the AWS region to which the EC2 instances belong.

## Example
Checks all the EC2 instances tagged with `Role=www` join the Consul cluster with the tag `role=www`.

```sh
export AWS_REGION=ap-northeast-1
export AWS_ACCESS_KEY_ID=AKIAxxxxxxxxxxxxxxxx
export AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
check-consul-members --ec2-tag Role --consul-tag role --value www
```
