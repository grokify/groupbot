# GroupBot

[![Build Status][build-status-svg]][build-status-link]
[![Go Report Card][goreport-svg]][goreport-link]
[![Docs][docs-godoc-svg]][docs-godoc-link]
[![License][license-svg]][license-link]

 [build-status-svg]: https://api.travis-ci.org/grokify/groupbot.svg?branch=master
 [build-status-link]: https://travis-ci.org/grokify/groupbot
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/groupbot
 [goreport-link]: https://goreportcard.com/report/github.com/grokify/groupbot
 [docs-godoc-svg]: https://img.shields.io/badge/docs-godoc-blue.svg
 [docs-godoc-link]: https://godoc.org/github.com/grokify/groupbot
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-link]: https://github.com/grokify/groupbot/blob/master/LICENSE

## Overview

GroupBot is a bot that allows you to share information about yourself with the team. It was initially created to share information for ordering tshirts. It currently stores data in a Google Sheet.

## Configuration

Set the following environment variables:

* `GROUPBOT_ENGINE`            // `aws` or `nethttp`
* `GROUPBOT_PORT`              // for `net/http`
* `GOOGLE_SERVICE_ACCOUNT_JWT`
* `GOOGLE_SHEET_INDEX`
* `GOOGLE_SPREADSHEET_ID`
* `RINGCENTRAL_BOT_ID`
* `RINGCENTRAL_SERVER_URL`
* `RINGCENTRAL_TOKEN_JSON`

## Using the AWS Engine

To use the AWS Lambda engine, you need an AWS account. If you don't hae one, the [free trial account](https://aws.amazon.com/s/dm/optimization/server-side-test/free-tier/free_np/) includes 1 million free Lambda requests per month forever and 1 million free API Gateway requests per month for the first year.

### Installation via AWS Lambda

See the AWS docs for deployment:

https://docs.aws.amazon.com/lambda/latest/dg/lambda-go-how-to-create-deployment-package.html

Using the `aws-cli` you can use the following approach:

```
$ GOOS=linux go build main.go
$ zip main.zip ./main
# --handler is the path to the executable inside the .zip
$ aws lambda create-function --region us-east-1 --function-name Databot --memory 128 --role arn:aws:iam::account-id:role/execution_role --runtime go1.x --zip-file fileb://main.zip --handler main
```
