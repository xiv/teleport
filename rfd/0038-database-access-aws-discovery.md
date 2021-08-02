---
authors: Roman Tkachenko (roman@goteleport.com)
state: draft
---

# RFD 38 - Automatic registration and discovery of databases on AWS

## What

Proposes a way for the database service agent to automatically discover and
register AWS-hosted RDS, Aurora and Redshift databases.

## Why

Using database access with AWS-hosted databases is currently somewhat painful
for a couple of reasons:

- Users have to manually enable IAM authentication and configure/attach IAM
  policies for each database which is very error and typo-prone, or create
  overly permissive IAM policies which is insecure.
- Users who have large numbers of AWS-hosted databases (we've had prospects with
  hundreds) or create/delete them on demand have no good way to automatically
  plug them into the cluster.

Teleport database agent should be able to automatically do both of these.

## Scope

This RFD focuses on AWS-hosted databases. Similar ideas will be explored for
GCP Cloud SQL in future.

## IAM

Currently, to allow IAM authentication with RDS instances, users have to enable
IAM authentication on the instance and create an IAM policy like this:

```json
{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Effect": "Allow",
         "Action": [
           "rds-db:connect"
         ],
         "Resource": [
           "arn:aws:rds-db:us-east-2:1234567890:dbuser:db-ABCDEFGHIJKL01234/*"
         ]
      }
   ]
}
```

This policy needs to exist for each RDS (or similar one for Redshift) instance
that is proxied by database access and be attached to the IAM identity the
database agent is using.

Instead of making the user enable IAM auth and create/attach the policy manually
(or build an automation for it), the database agent can do it itself.

### Required permissions

To support this, the database agent must have certain IAM permissions.

- `rds:ModifyDBCluster` and `rds:ModifyDBInstance` to be able to enable IAM
  authentication. For Redshift clusters IAM auth is always enabled.
- `iam:CreatePolicy` to be able to create policies for IAM RDS/Redshift
  authentication.
- `iam:AttachUserPolicy` or `iam:AttachRolePolicy` to be able to attach the
  created policies to the IAM identity (user/role) it's running as.

To know how to attach the policy to itself, the agent will use `sts:GetCallerIdentity`[1]
call to determine IAM identity it's running as. This operation does not require
any permissions.

In addition, the agent will need to have `iam:DeletePolicy` and `iam:Detach*Policy`
permissions to support deprovisioning of AWS databases removed from the cluster.

### Privilege escalation

Database agent being able to create and attach policies to itself raises
privilege escalation concerns. Can somebody who has access to the database
agent instance give itself unlimited IAM permissions?

To prevent privilege escalation users can use IAM [permission
boundaries](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_boundaries.html).
Permission boundary is an IAM policy that does not grant any privileges by
itself but defines the maximum privileges that can be granted to a user/role.
For example, if permission boundary is set and doesn't explicitly allow to
create EC2 instances, the user won't be able to create them even if their
attached IAM policies allow it. The effective permissions set of an IAM identity
is the intersection between their attached IAM policy and permission boundary.

Applied to the database agent scenario, users will create a permission boundary
that only allows the agent to create/attach IAM policies and connect to an RDS
instance:

```json
{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Effect": "Allow",
         "Action": [
           "iam:CreatePolicy",
           "iam:DeletePolicy",
           "iam:Attach*",
           "iam:Detach*"
         ],
         "Resource": "*",
      },
      {
         "Effect": "Allow",
         "Action": [
           "rds-db:connect"
         ],
         "Resource": [
           "arn:aws:rds-db:*:1234567890:dbuser:*/*"
         ]
      }
   ]
}
```

With this permission boundary, whoever has access to the database agent instance
will be able to create any policy but not use it for anything besides connecting
to an RDS instance.

## Discovery

Database agent can auto-discover RDS and Aurora instances in the AWS account
it has credentials for by matching resource tags. Auto-discovery is disabled
by default. It can be enabled by providing a `selector` introduced in RFD 31 in
the agent config:

```yaml
db_service:
  enabled: "yes"
  selectors:
  - match_rds:
      regions: ["us-east-1"]
      tags:
        'env': 'prod'
  - match_redshift:
      regions: ["us-east-1", "us-west-2"]
      tags:
        'env': 'stage'
```

The agent will use RDS' `DescribeDBClusters`[2] and `DescribeDBInstances`[3]
APIs and Redshift's `DescribeClusters`[4] API for discovery. It will require the
agent's IAM policy and permission boundary to include `rds:DescribeDBClusters`,
`rds:DescribeDBInstances` and `redshift:DescribeClusters` permissions.

AWS API doesn't appear to support any sort of resource watchers so the database
agent will scan the resources on a regular interval to see if any new instances
matching the criteria have appeared, or any existing instances have been
deleted.

The agent will create/delete Teleport database resources based on the detected
RDS/Redshift instances. A database resource will have the same name as
RDS/Redshift instance it represents and static labels taken from the instance's
AWS tags.

## Footnotes

[1] https://docs.aws.amazon.com/sdk-for-go/api/service/sts/#STS.GetCallerIdentity
[2] https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#RDS.DescribeDBClusters
[3] https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#RDS.DescribeDBInstances
[4] https://docs.aws.amazon.com/sdk-for-go/api/service/redshift/#Redshift.DescribeClusters
