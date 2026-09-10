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

## Pipelines

### Build, scan, and publish

`.github/workflows/build.yml` runs on pushes to `main` and on manual dispatch:

1. Scans the checked-out source with the central Wiz directory workflow.
2. Builds the container and pushes it to ECR with a unique
   `candidate-<commit SHA>-<run ID>-<attempt>` quarantine tag.
3. Resolves the candidate to an immutable registry digest, then uses the
   central image workflow to pull and scan that exact digest.
4. Sets `tag_image: true`, so a successful scan adds the digest to the Wiz
   Trusted Image Database.
5. Requires the workflow's `trusted_image_tagged` output to be `true`, then
   promotes the same manifest to the release commit-SHA tag without rebuilding
   it.

The temporary candidate tag is necessary because ECR assigns the registry
digest used by Image Trust only after the image is pushed. A failed scan leaves
the candidate in ECR but never creates the release tag and never adds it to the
Wiz Trusted Image Database. An ECR lifecycle rule can expire `candidate-*`
images automatically.

### Scan and deploy

`.github/workflows/deploy.yml` is manually dispatched with the commit SHA tag
and the target EKS cluster information. It first resolves the tag to an
immutable ECR digest, then pulls and scans that digest through the central Wiz
image workflow. The workflow also refreshes the image's trusted status with
`tag_image: true`.

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
| `ECR_REGISTRY_PASSWORD` | Password used by the reusable workflow to pull the private ECR image |

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

`ECR_REGISTRY_PASSWORD` can be generated with:

```bash
aws ecr get-login-password --region ap-southeast-1
```

Amazon ECR authorization tokens expire after 12 hours, so refresh this secret
before running either workflow. This limitation comes from the current
central image workflow accepting registry username/password credentials but not
AWS OIDC credentials. For an unattended deploy pipeline, extend
`wiz-image-scan.yml` to authenticate to ECR with GitHub OIDC.

The AWS role must trust this repository's GitHub OIDC subject. The build job
needs ECR upload permissions. The deploy job needs ECR read access and
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

The reusable workflows currently reference `@main` so the demo always tests the
latest version of `arun-wiz/wiz-workflows`. Switch these references to a release
tag after the reusable workflows have been tested.
