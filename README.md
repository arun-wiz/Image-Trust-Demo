# Image Trust Demo

A small Go web application that displays **Hello from Wiz** and demonstrates
Wiz-gated build and deployment pipelines using the reusable workflows in
[`arun-wiz/wiz-workflows`](https://github.com/arun-wiz/wiz-workflows).

## Run it locally

```bash
docker build -t image-trust-demo .
docker run --rm -p 8080:8080 image-trust-demo
```

Open <http://localhost:8080>. The health endpoint is available at
<http://localhost:8080/healthz>.

## Branch and promotion workflow

The repository uses three long-lived branches:

| Branch | Purpose | Allowed pull request source |
| --- | --- | --- |
| `main` | Integrated development code | A feature or fix branch |
| `staging` | Code undergoing staging tests | `main` |
| `production` | Code approved for production | `staging` |

Use the following promotion path:

1. Create a feature or fix branch from the latest `main` and make changes there.
2. Open a pull request from that branch to `main`.
3. When the change is ready for staging, open a pull request from `main` to
   `staging` and complete staging tests.
4. After staging approval, open a pull request from `staging` to `production`.

Direct pushes to the three long-lived branches are blocked by GitHub branch
protection. `.github/workflows/validate-promotion.yml` rejects pull requests
that bypass the promotion path. Sync a local branch without creating merge
commits by using `git pull --ff-only`.

## Pipelines

### Build, scan, and publish

`.github/workflows/pr-security.yml` runs source and locally built image scans on
pull requests targeting `main`, `staging`, or `production`. The branch-aware
policy selection from `arun-wiz/wiz-workflows` is enabled with
`policy_profile: auto`: staging PRs use the staging policy set, while main and
production PRs use the production policy set. PR image scans do not authenticate
to AWS or push an image.

`.github/workflows/build.yml` runs after merges to `main`, `staging`, and
`production`, and on manual dispatch:

1. Scans the checked-out source with the central Wiz directory workflow.
2. Builds the commit-SHA image locally in this repository's workflow.
3. Calls the platform-agnostic Wiz image-scan composite action, which scans the
   local image and enforces centrally configured Wiz CI/CD policies before any
   registry push.
4. Only after a successful scan, assumes the AWS role through GitHub OIDC,
   authenticates to ECR, pushes the same local image, and captures its
   registry-assigned digest.
5. Calls the platform-agnostic Wiz image-tag composite action with that digest
   to add it to the Wiz Trusted Image Database.

Both its source and image scans use the same automatic branch-aware policy
selection. Images are published only by this post-merge workflow.

All image steps execute in the same caller-owned job, so the exact image
that passes the scan is the one pushed to ECR. A scan or policy failure skips
both the ECR push and Image Trust registration.

### Scan and deploy

`.github/workflows/deploy.yml` is manually dispatched with the commit SHA tag
and the target EKS cluster information. Its scan job assumes the AWS role,
resolves the tag to an immutable ECR digest, and pulls that digest. It then uses
the platform-agnostic Wiz scan and tag composite actions before allowing the
EKS deployment job to run.

After the policy and Image Trust operations pass, the workflow:

1. Connects to the requested EKS cluster using AWS OIDC credentials.
2. Creates the requested namespace when it does not exist.
3. Applies the Deployment, ClusterIP Service, and Ingress in
   `k8s/application.yaml` using the approved immutable image digest.
4. Waits for the Kubernetes rollout and for the public ALB hostname.

The workflow dispatch form requests:

| Input | Purpose |
| --- | --- |
| `image_tag` | ECR release tag, normally the build commit SHA |
| `eks_cluster_name` | Target EKS cluster name |
| `eks_region` | AWS region containing the EKS cluster |
| `kubernetes_namespace` | Namespace in which to deploy the application |

The Ingress uses `ingressClassName: alb`, IP targets, the `/healthz` health
check, and the `internet-facing` scheme. The resulting HTTP URL is written to
the GitHub Actions job summary.

Use GitHub's `production` environment protection rules if deployment approval
is required.

## GitHub configuration

Create these Actions secrets:

| Secret | Purpose |
| --- | --- |
| `WIZ_CLIENT_ID` | Wiz CI/CD service-account client ID |
| `WIZ_CLIENT_SECRET` | Wiz CI/CD service-account client secret |

`WIZ_CLIENT_ID` and `WIZ_CLIENT_SECRET` may be organization-level secrets. Give
this repository access to them in the `arun-wiz` organization; the explicit
secret mappings in the workflow will then resolve those organization secrets.
Repository-level duplicates are not required. Explicit mappings are retained
instead of `secrets: inherit` so unrelated organization secrets are not passed
to the reusable workflows.

Create these repository or environment variables:

| Variable | Example |
| --- | --- |
| `AWS_REGION` | `ap-southeast-1` |
| `AWS_ROLE_ARN` | `arn:aws:iam::123456789012:role/github-image-trust-demo` |
| `ECR_REGISTRY` | `123456789012.dkr.ecr.ap-southeast-1.amazonaws.com` |
| `ECR_REPOSITORY` | `image-trust-demo` |

The AWS role must trust this repository's GitHub OIDC subject. The build job
needs ECR authentication and upload permissions. The deploy job needs ECR
authentication and read access plus
`eks:DescribeCluster`. Its IAM principal must also have an EKS access entry (or
equivalent Kubernetes RBAC mapping) that can manage namespaces, Deployments,
Services, and Ingresses. The cluster node or Fargate execution role must be able
to pull this repository's images from ECR.

The target cluster must already have an `alb` IngressClass backed by the AWS
Load Balancer Controller, and its public subnets must be discoverable by that
controller. The cluster API endpoint must be reachable from the GitHub runner;
use a network-connected self-hosted runner for a private-only endpoint. The
demo exposes HTTP on port 80. Add an ACM certificate and HTTPS listener
annotations before using it for production traffic.

The reusable workflow and composite actions use the tested `v1` release of
`arun-wiz/wiz-workflows`, pinned to its full commit SHA so the references are
immutable.
