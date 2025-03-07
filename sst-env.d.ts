/* This file is auto-generated by SST. Do not edit. */
/* tslint:disable */
/* eslint-disable */
/* deno-fmt-ignore-file */

declare module "sst" {
  export interface Resource {
    "API": {
      "type": "sst.aws.ApiGatewayV2"
      "url": string
    }
    "AppClientId": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "AuthClient": {
      "name": string
      "type": "sst.aws.Function"
    }
    "AuthServer": {
      "type": "sst.aws.Auth"
      "url": string
    }
    "BcvFunction": {
      "name": string
      "type": "sst.aws.Function"
      "url": string
    }
    "BcvQueue": {
      "type": "sst.aws.Queue"
      "url": string
    }
    "GithubClientId": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "GithubClientSecret": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "GoogleClientId": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "GoogleClientSecret": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "HtmlToPdf": {
      "name": string
      "type": "sst.aws.Function"
    }
    "MainApiFunction": {
      "name": string
      "type": "sst.aws.Function"
    }
    "ProcessBcvFile": {
      "name": string
      "type": "sst.aws.Function"
    }
    "ProcessUser": {
      "name": string
      "type": "sst.aws.Function"
    }
    "ReceiptPdfChangesQueue": {
      "type": "sst.aws.Queue"
      "url": string
    }
    "ReceiptsBucket": {
      "name": string
      "type": "sst.aws.Bucket"
    }
    "SecretBcvFileStartPath": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "SecretBcvUrl": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "SecretTursoUrl": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "UploadBackupBucket": {
      "name": string
      "type": "sst.aws.Bucket"
    }
    "UploadBackupQueue": {
      "type": "sst.aws.Queue"
      "url": string
    }
    "VerifyAccess": {
      "name": string
      "type": "sst.aws.Function"
    }
    "WebApp": {
      "type": "sst.aws.StaticSite"
      "url": string
    }
    "WebUrl": {
      "type": "sst.sst.Secret"
      "value": string
    }
    "bcv-bucket": {
      "name": string
      "type": "sst.aws.Bucket"
    }
  }
}
/// <reference path="sst-env.d.ts" />

import "sst"
export {}