# Storage Service Website Action

This GitHub Action automates the deployment of static websites to an AWS Storage Service (S3). It provides customizable options such as caching control, file handling, and pattern-based rules for more flexible deployment configurations.

## Features

- Deploy static websites to AWS S3 buckets
- Customizable caching rules (e.g., different cache times for HTML, images, etc.)
- Optional removal of `.html` extensions from URLs
- Ability to exclude specific files or folders during deployment

## Usage

Here's an example of how to use the `storage-service-website-action` in a GitHub workflow:

```yaml
name: Deploy Website
on:
  push:
    branches:
      - main

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v2

      - name: Deploy website to S3
        uses: rizaldiantoro/storage-service-website-action@v1
        with:
          folder: "public" # The folder containing your static site files
          bucket: ${{ secrets.AWS_BUCKET }} # Your S3 bucket name
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: "us-west-2"
          remove-html-extension: "true"
          object-rules: |
            - pattern: '_next/*'
              cache-control: 'max-age=31536000, immutable'
            - pattern: 'assets/*'
              cache-control: 'max-age=86400'
            - pattern: 'images/*'
              cache-control: 'max-age=86400'
```

## Inputs

| Input                              | Description                                                                        | Required | Default           |
| ---------------------------------- | ---------------------------------------------------------------------------------- | -------- | ----------------- |
| `folder`                           | The folder containing the static website files to upload                           | Yes      |                   |
| `bucket`                           | The name of the S3 bucket where the website will be deployed                       | Yes      |                   |
| `aws-access-key-id`                | AWS Access Key ID for authentication                                               | Yes      |                   |
| `aws-secret-access-key`            | AWS Secret Access Key for authentication                                           | Yes      |                   |
| `aws-session-token`                | AWS Session Token for temporary credentials                                        | No       |                   |
| `aws-region`                       | The AWS region where your S3 bucket is located                                     | Yes      |                   |
| `object-rules`                     | YAML configuration for cache-control and content-type rules based on file patterns | No       |                   |
| `exclude`                          | Files or folders to exclude from the upload                                        | No       |                   |
| `default-cache-control`            | Default Cache-Control value for files without specific rules                       | No       | `max-age=2592000` |
| `html-cache-control`               | Cache-Control value for HTML files                                                 | No       | `max-age=600`     |
| `image-cache-control`              | Cache-Control value for image files                                                | No       | `max-age=864000`  |
| `pdf-cache-control`                | Cache-Control value for PDF files                                                  | No       | `max-age=2592000` |
| `remove-html-extension`            | Remove `.html` extension from URLs                                                 | No       | `false`           |
| `duplicate-html-with-no-extension` | Duplicate HTML files with no extension for alternative URL formats                 | No       | `false`           |

## Acknowledgements

A huge thanks to fangbinwei/aliyun-oss-website-action for inspiring this action. Many ideas and concepts were borrowed from that repository in order to create this solution.
