# github-actions-aws-s3-static-website
Github Action that creates a static website in AWS S3 bucket

### Usage

Github Action:
NOTE: It outputs `website_url` so you can use it in next steps (even if this site already exists)
```yaml
jobs:
  create-static-website:
    runs-on: ubuntu-latest
    steps:
      - name: Create static website
        id: create-bucket
        uses: yzinkovets/github-actions-aws-s3-static-website@v1
        with:
          region: ${{ secrets.AWS_REGION }}
          aws_access_key_id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws_secret_access_key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          domain: ${{ secrets.DOMAIN }}
          index_document: ${{ secrets.INDEX_DOCUMENT }}
          error_document: ${{ secrets.ERROR_DOCUMENT }}
      - name: Use website_url
        run: echo "Website URL: ${{ steps.create-bucket:.outputs.website_url }}"
```
