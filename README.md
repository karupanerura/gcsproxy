# gcsproxy

HTTP Proxy for Goolge Cloud Storage.

[![Run on Google Cloud](https://deploy.cloud.run/button.svg)](https://deploy.cloud.run)

## Motivation

Cloud Load Balancing can use Identity Aware Proxy (IAP) to restrict access.
IAPs can be applied to the Backend Service, but not to the Backend Bucket.
Therefore, access to Google Cloud Storage (GCS) cannot be restricted using IAP.

To apply IAP to Google Cloud Storage, you must access Google Cloud Storage (GCS) via the Backend Service.
Therefore, a Proxy to GCS acting as a Backend Service is required to restrict access for GCS.

## Usage

Deploy the built docker image `ghcr.io/karupanerura/gcsproxy:v0.0.5` to Cloud Run or any other services of Google Cloud Platform.

### Environment Variables

* GCS_PROXY_BUCKET: GCS bucket name. (required)
* GCS_PROXY_PATH_PREFIX: URL path prefix. use it as GCS object key from URL path exclude the path prefix. (default: "/")
* GCS_PROXY_INDEX_FILE: Index file name. (e.g. "index.html")
* GCS_PROXY_NOT_FOUND_PATH: Path for not found error. (e.g. "/404.html")
* GCS_PROXY_BASIC_AUTH: Basic Authentication Settings in .htpasswd format. (e.g. "usr1:kI6oFWZHn9A\nusr2:kI6oFWZHn9AJA")